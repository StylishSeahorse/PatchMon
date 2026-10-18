package commands

import (
	"patchmon-agent/internal/logutil"
)

// postPatchRebootDelayMinutes gives logged-in users the shutdown notice and lets
// the agent's final exchanges with the server drain before the host goes down.
const postPatchRebootDelayMinutes = 1

const postPatchRebootReason = "PatchMon post-patch reboot (policy)"

// rebootController is the part of system.Detector a post-patch reboot needs.
type rebootController interface {
	CheckRebootRequired() (bool, string)
	ExecuteReboot(delayMinutes int, reason string) error
}

// maybeRebootAfterPatch reboots the host after a patch run when the server asked
// for it (a scheduled run of a policy with auto-reboot). It reboots only when
// the run succeeded, was not a dry run, and the host still reports a pending
// reboot, so a run that changed nothing relevant leaves the host up. It reports
// whether a reboot was scheduled.
func maybeRebootAfterPatch(host rebootController, requested, dryRun bool, runErr error) bool {
	if !requested || dryRun || runErr != nil {
		return false
	}
	needed, reason := host.CheckRebootRequired()
	if !needed {
		logger.Info("Post-patch reboot requested by policy but the host does not need one; skipping")
		return false
	}
	logger.WithField("reason", logutil.Sanitize(reason)).Info("Post-patch reboot required by policy; scheduling reboot")
	if err := host.ExecuteReboot(postPatchRebootDelayMinutes, postPatchRebootReason); err != nil {
		logger.WithError(err).Warn("Post-patch reboot could not be scheduled")
		return false
	}
	return true
}
