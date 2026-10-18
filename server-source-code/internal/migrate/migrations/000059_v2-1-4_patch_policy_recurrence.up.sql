-- Recurring patch runs: a policy can carry a cron schedule (interpreted in the
-- organisation timezone). When recurrence_next_run_at passes, a patch_all run is
-- created for every host the policy applies to.
ALTER TABLE patch_policies ADD COLUMN IF NOT EXISTS recurrence_cron TEXT;
ALTER TABLE patch_policies ADD COLUMN IF NOT EXISTS recurrence_next_run_at TIMESTAMP(3);
ALTER TABLE patch_policies ADD COLUMN IF NOT EXISTS recurrence_last_run_at TIMESTAMP(3);

CREATE INDEX IF NOT EXISTS patch_policies_recurrence_next_run_at_idx ON patch_policies(recurrence_next_run_at) WHERE recurrence_cron IS NOT NULL;
