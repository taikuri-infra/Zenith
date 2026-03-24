-- Environments: each project has one or more environments (production, staging).
-- Pro+ users get a staging environment auto-created alongside production.
CREATE TABLE IF NOT EXISTS environments (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'provisioning',
    is_default      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_environments_project_name ON environments(project_id, name);
CREATE INDEX IF NOT EXISTS idx_environments_project ON environments(project_id);

-- Link existing tables to environments
ALTER TABLE apps ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id);
ALTER TABLE managed_services ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id);
