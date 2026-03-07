-- Projects: logical grouping of resources under a user
CREATE TABLE IF NOT EXISTS projects (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_user_slug ON projects(user_id, slug);
CREATE INDEX IF NOT EXISTS idx_projects_user ON projects(user_id);

-- Default project for every existing user (deterministic ID for idempotency)
INSERT INTO projects (id, user_id, name, slug)
SELECT 'proj-' || id, id, 'Default', 'default' FROM users
ON CONFLICT DO NOTHING;

-- Add project_id to apps
ALTER TABLE apps ADD COLUMN IF NOT EXISTS project_id TEXT;
UPDATE apps SET project_id = 'proj-' || user_id WHERE project_id IS NULL;
ALTER TABLE apps ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE apps ADD CONSTRAINT fk_apps_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;
DROP INDEX IF EXISTS idx_apps_user_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_apps_project_name ON apps(project_id, name);
CREATE INDEX IF NOT EXISTS idx_apps_project ON apps(project_id);

-- Add project_id to user_databases
ALTER TABLE user_databases ADD COLUMN IF NOT EXISTS project_id TEXT;
UPDATE user_databases SET project_id = 'proj-' || user_id WHERE project_id IS NULL;
ALTER TABLE user_databases ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE user_databases ADD CONSTRAINT fk_user_databases_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;
DROP INDEX IF EXISTS idx_user_databases_user_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_databases_project_name ON user_databases(project_id, name);
CREATE INDEX IF NOT EXISTS idx_user_databases_project ON user_databases(project_id);

-- Add project_id to user_buckets
ALTER TABLE user_buckets ADD COLUMN IF NOT EXISTS project_id TEXT;
UPDATE user_buckets SET project_id = 'proj-' || user_id WHERE project_id IS NULL;
ALTER TABLE user_buckets ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE user_buckets ADD CONSTRAINT fk_user_buckets_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;
DROP INDEX IF EXISTS idx_user_buckets_user_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_buckets_project_name ON user_buckets(project_id, name);
CREATE INDEX IF NOT EXISTS idx_user_buckets_project ON user_buckets(project_id);

-- Add project_id to gateways
ALTER TABLE gateways ADD COLUMN IF NOT EXISTS project_id TEXT;
UPDATE gateways SET project_id = 'proj-' || user_id WHERE project_id IS NULL;
ALTER TABLE gateways ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE gateways ADD CONSTRAINT fk_gateways_project FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_gateways_project ON gateways(project_id);
