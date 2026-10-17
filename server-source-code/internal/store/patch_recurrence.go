package store

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/PatchMon/PatchMon/server-source-code/internal/db"
	"github.com/PatchMon/PatchMon/server-source-code/internal/pgtime"
	"github.com/robfig/cron/v3"
)

// ValidateRecurrenceCron checks the cron expression of a recurring patch policy.
// It must be a standard 5-field expression whose minute and hour are fixed
// numbers, so a policy patches at most once per day and never more often.
func ValidateRecurrenceCron(expr string) error {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return errors.New("recurrence must be a 5-field cron expression (minute hour day-of-month month day-of-week)")
	}
	if n, err := strconv.Atoi(fields[0]); err != nil || n < 0 || n > 59 {
		return errors.New("recurrence minute must be a single number 0-59")
	}
	if n, err := strconv.Atoi(fields[1]); err != nil || n < 0 || n > 23 {
		return errors.New("recurrence hour must be a single number 0-23")
	}
	if _, err := cron.ParseStandard(strings.Join(fields, " ")); err != nil {
		return errors.New("recurrence is not a valid cron expression")
	}
	return nil
}

// NextRecurrence returns the first instant after from at which expr fires,
// with the schedule read in the IANA zone tz. The result is UTC.
func NextRecurrence(expr, tz string, from time.Time) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	sched, err := cron.ParseStandard(expr)
	if err != nil {
		return time.Time{}, err
	}
	next := sched.Next(from.In(loc))
	if next.IsZero() {
		return time.Time{}, errors.New("recurrence never fires")
	}
	return next.UTC(), nil
}

// SetRecurrence sets or clears (expr nil) the recurring schedule of a policy.
// tz is the organisation timezone the schedule is read in.
func (s *PatchPoliciesStore) SetRecurrence(ctx context.Context, id string, expr *string, tz string) error {
	params := db.SetPatchPolicyRecurrenceParams{ID: id}
	if expr != nil {
		next, err := NextRecurrence(*expr, tz, time.Now())
		if err != nil {
			return err
		}
		params.RecurrenceCron = expr
		params.RecurrenceNextRunAt = pgtime.From(next)
	}
	return s.db.DB(ctx).Queries.SetPatchPolicyRecurrence(ctx, params)
}
