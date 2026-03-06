-- Add app_type, command, and cron_schedule to apps table.
-- app_type: "web" (default), "worker", or "cron"
-- command: override entrypoint for worker apps
-- cron_schedule: cron expression for cron apps

ALTER TABLE apps ADD COLUMN IF NOT EXISTS app_type TEXT NOT NULL DEFAULT 'web';
ALTER TABLE apps ADD COLUMN IF NOT EXISTS command TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN IF NOT EXISTS cron_schedule TEXT NOT NULL DEFAULT '';
