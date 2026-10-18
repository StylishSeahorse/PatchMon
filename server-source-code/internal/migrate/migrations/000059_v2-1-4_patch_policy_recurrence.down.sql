DROP INDEX IF EXISTS patch_policies_recurrence_next_run_at_idx;
ALTER TABLE patch_policies DROP COLUMN IF EXISTS recurrence_last_run_at;
ALTER TABLE patch_policies DROP COLUMN IF EXISTS recurrence_next_run_at;
ALTER TABLE patch_policies DROP COLUMN IF EXISTS recurrence_cron;
