DROP INDEX IF EXISTS idx_apps_deleted_at;
ALTER TABLE apps DROP COLUMN IF EXISTS deleted_at;
