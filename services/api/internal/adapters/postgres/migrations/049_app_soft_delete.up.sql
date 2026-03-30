ALTER TABLE apps ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_apps_deleted_at ON apps(deleted_at) WHERE deleted_at IS NOT NULL;
