package queue

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"time"

	"github.com/PatchMon/PatchMon/server-source-code/internal/config"
	hostctx "github.com/PatchMon/PatchMon/server-source-code/internal/context"
	"github.com/PatchMon/PatchMon/server-source-code/internal/database"
	"github.com/PatchMon/PatchMon/server-source-code/internal/db"
	"github.com/PatchMon/PatchMon/server-source-code/internal/pgtime"
	"github.com/PatchMon/PatchMon/server-source-code/internal/store"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	TypePatchPolicyScheduleDispatch = "patch-policy-schedule-dispatch"
	QueuePatchPolicySchedule        = "patch-policy-schedule"

	// patchPolicyCatchUpWindow is how late a recurring occurrence may still run.
	// After downtime longer than this the occurrence is skipped, so a nightly
	// maintenance window never fires in the middle of the working day.
	patchPolicyCatchUpWindow = 30 * time.Minute
)

// PatchPolicyScheduleHandler turns due recurring patch policies into patch_all
// runs, one per host the policy applies to.
type PatchPolicyScheduleHandler struct {
	defaultDB     *database.DB
	poolCache     *hostctx.PoolCache
	patchRuns     *store.PatchRunsStore
	patchPolicies *store.PatchPoliciesStore
	qc            *asynq.Client
	log           *slog.Logger
}

// NewPatchPolicyScheduleHandler creates the handler.
func NewPatchPolicyScheduleHandler(defaultDB *database.DB, poolCache *hostctx.PoolCache, patchRuns *store.PatchRunsStore, patchPolicies *store.PatchPoliciesStore, qc *asynq.Client, log *slog.Logger) *PatchPolicyScheduleHandler {
	return &PatchPolicyScheduleHandler{defaultDB: defaultDB, poolCache: poolCache, patchRuns: patchRuns, patchPolicies: patchPolicies, qc: qc, log: log}
}

// patchPolicyOccurrenceMissed reports whether an occurrence due at expected is
// too old to run at now.
func patchPolicyOccurrenceMissed(now, expected time.Time) bool {
	return now.Sub(expected) > patchPolicyCatchUpWindow
}

// ProcessTask implements asynq.Handler.
func (h *PatchPolicyScheduleHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	if payload := t.Payload(); len(payload) > 0 {
		ctx, d, err := resolveDBFromPayload(ctx, payload, h.defaultDB, h.poolCache)
		if err != nil {
			return err
		}
		h.processDB(ctx, d, tenantHostFromPayload(payload))
		return nil
	}
	h.processDB(ctx, h.defaultDB, "")
	if h.poolCache == nil {
		return nil
	}
	for _, host := range h.poolCache.ListHosts() {
		hctx, d, _ := resolveDBForHost(ctx, host, nil, h.poolCache)
		if d == nil {
			continue
		}
		h.processDB(hctx, d, host)
	}
	return nil
}

func (h *PatchPolicyScheduleHandler) processDB(ctx context.Context, d *database.DB, tenantHost string) {
	ctx = hostctx.WithDB(ctx, d)
	now := time.Now().UTC()
	due, err := d.Queries.ListPatchPoliciesRecurrenceDue(ctx, pgtime.From(now))
	if err != nil {
		h.log.Error("patch_policy_schedule: list due policies", "error", err)
		return
	}
	if len(due) == 0 {
		return
	}
	var dbTZ *string
	if s, err := d.Queries.GetFirstSettings(ctx); err == nil {
		dbTZ = s.Timezone
	}
	orgTZ := config.ResolveTimezone(dbTZ, nil)
	for _, p := range due {
		h.runPolicy(ctx, d, p, tenantHost, orgTZ, now)
	}
}

func (h *PatchPolicyScheduleHandler) runPolicy(ctx context.Context, d *database.DB, p db.PatchPolicy, tenantHost, orgTZ string, now time.Time) {
	if p.RecurrenceCron == nil {
		return
	}
	next, err := store.NextRecurrence(*p.RecurrenceCron, orgTZ, now)
	if err != nil {
		h.log.Error("patch_policy_schedule: bad recurrence", "policy_id", p.ID, "recurrence", *p.RecurrenceCron, "error", err)
		return
	}
	missed := patchPolicyOccurrenceMissed(now, p.RecurrenceNextRunAt.Time)
	lastRun := p.RecurrenceLastRunAt
	if !missed {
		lastRun = pgtime.From(now)
	}
	// Move the schedule on before dispatching: only the worker that wins this
	// compare-and-swap fires the occurrence, and a crash mid-dispatch can never
	// make it fire twice.
	claimed, err := d.Queries.ClaimPatchPolicyRecurrence(ctx, db.ClaimPatchPolicyRecurrenceParams{
		ID:                p.ID,
		NextRunAt:         pgtime.From(next),
		LastRunAt:         lastRun,
		ExpectedNextRunAt: p.RecurrenceNextRunAt,
	})
	if err != nil {
		h.log.Error("patch_policy_schedule: claim occurrence", "policy_id", p.ID, "error", err)
		return
	}
	if claimed == 0 {
		return
	}
	if missed {
		h.log.Warn("patch_policy_schedule: occurrence missed, skipping", "policy_id", p.ID, "policy", p.Name,
			"was_due", p.RecurrenceNextRunAt.Time, "next", next)
		return
	}

	hostIDs, err := h.targetHostIDs(ctx, d, p)
	if err != nil {
		h.log.Error("patch_policy_schedule: resolve target hosts", "policy_id", p.ID, "error", err)
		return
	}
	started := 0
	for _, hostID := range hostIDs {
		if h.startRun(ctx, d, p, hostID, tenantHost) {
			started++
		}
	}
	h.log.Info("patch_policy_schedule: dispatched", "policy_id", p.ID, "policy", p.Name,
		"hosts", len(hostIDs), "runs_started", started, "next", next)
}

// targetHostIDs returns the connected hosts with outdated packages for which p
// is the effective policy, in a stable order. Resolving the effective policy
// per host applies the same precedence and exclusions as a manual run.
func (h *PatchPolicyScheduleHandler) targetHostIDs(ctx context.Context, d *database.DB, p db.PatchPolicy) ([]string, error) {
	assignments, err := d.Queries.ListPatchPolicyAssignmentsByPolicy(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	candidates := map[string]struct{}{}
	for _, a := range assignments {
		switch a.TargetType {
		case "host":
			candidates[a.TargetID] = struct{}{}
		case "host_group":
			ids, err := d.Queries.ListHostIDsInGroup(ctx, a.TargetID)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				candidates[id] = struct{}{}
			}
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(candidates))
	for id := range candidates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	patchable, err := d.Queries.ListPatchableHostIDsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, id := range patchable {
		eff, err := h.patchPolicies.ResolveEffectivePolicy(ctx, id)
		if err != nil || eff == nil || eff.ID != p.ID {
			continue
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// startRun creates and enqueues a patch_all run for one host. It reports
// whether a run was started.
func (h *PatchPolicyScheduleHandler) startRun(ctx context.Context, d *database.DB, p db.PatchPolicy, hostID, tenantHost string) bool {
	host, err := d.Queries.GetHostByID(ctx, hostID)
	if err != nil {
		h.log.Error("patch_policy_schedule: load host", "policy_id", p.ID, "host_id", hostID, "error", err)
		return false
	}
	var opts *store.CreateRunOpts
	if store.IsWindowsOSType(host.OsType) {
		if !store.WindowsAgentCanPatch(host.AgentVersion) {
			h.log.Warn("patch_policy_schedule: Windows agent too old, skipping host", "policy_id", p.ID, "host_id", hostID,
				"min_agent_version", store.MinWindowsPatchAgentVersion)
			return false
		}
		all := true
		opts = &store.CreateRunOpts{WindowsIncludeUpdates: &all, WindowsIncludeApps: &all}
	}

	snapshot, _ := json.Marshal(map[string]interface{}{
		"name":             p.Name,
		"patch_delay_type": p.PatchDelayType,
		"delay_minutes":    p.DelayMinutes,
		"fixed_time_utc":   p.FixedTimeUtc,
		"timezone":         p.Timezone,
		"recurrence_cron":  p.RecurrenceCron,
		"trigger":          "scheduled",
	})
	runID := uuid.New().String()
	jobID := "patch-run-" + runID
	policyID, policyName := p.ID, p.Name
	_, err = h.patchRuns.CreateRun(ctx, runID, hostID, jobID, "patch_all", nil, nil, nil, false, nil, &policyID, &policyName, snapshot, opts)
	if errors.Is(err, store.ErrHostHasActivePatchRun) {
		h.log.Info("patch_policy_schedule: host busy, skipping", "policy_id", p.ID, "host_id", hostID)
		return false
	}
	if err != nil {
		h.log.Error("patch_policy_schedule: create run", "policy_id", p.ID, "host_id", hostID, "error", err)
		return false
	}
	task, err := NewRunPatchTask(RunPatchPayload{
		HostID:     hostID,
		Host:       tenantHost,
		ApiID:      host.ApiID,
		PatchRunID: runID,
		PatchType:  "patch_all",
	})
	if err == nil {
		_, err = h.qc.Enqueue(task)
	}
	if err != nil {
		h.log.Error("patch_policy_schedule: enqueue run", "policy_id", p.ID, "host_id", hostID, "run_id", runID, "error", err)
		if _, cerr := h.patchRuns.Cancel(context.WithoutCancel(ctx), runID, "Failed to queue patch run"); cerr != nil {
			h.log.Error("patch_policy_schedule: cancel orphaned run", "run_id", runID, "error", cerr)
		}
		return false
	}
	return true
}
