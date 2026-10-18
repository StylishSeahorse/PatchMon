-- Scheduled (recurring) runs of a policy can reboot the host after a successful
-- patch run, but only when the host reports a pending reboot. Manual runs are
-- never affected.
ALTER TABLE patch_policies ADD COLUMN IF NOT EXISTS auto_reboot BOOLEAN NOT NULL DEFAULT false;
