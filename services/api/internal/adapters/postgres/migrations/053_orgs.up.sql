CREATE TABLE IF NOT EXISTS orgs (
    id         TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    namespace  TEXT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_orgs_project_slug ON orgs(project_id, slug);
CREATE UNIQUE INDEX IF NOT EXISTS idx_orgs_namespace ON orgs(namespace);
CREATE UNIQUE INDEX IF NOT EXISTS idx_orgs_project_default ON orgs(project_id) WHERE is_default;

-- Every existing project gets a default Org living in the legacy shared namespace,
-- so running workloads are not moved.
INSERT INTO orgs (id, project_id, name, slug, namespace, is_default)
SELECT 'org-' || p.id, p.id, 'Default', 'default', 'zenith-apps', TRUE FROM projects p
WHERE NOT EXISTS (SELECT 1 FROM orgs o WHERE o.project_id = p.id AND o.is_default);

ALTER TABLE apps           ADD COLUMN IF NOT EXISTS org_id TEXT REFERENCES orgs(id) ON DELETE RESTRICT;
ALTER TABLE user_databases ADD COLUMN IF NOT EXISTS org_id TEXT REFERENCES orgs(id) ON DELETE RESTRICT;
ALTER TABLE user_buckets   ADD COLUMN IF NOT EXISTS org_id TEXT REFERENCES orgs(id) ON DELETE RESTRICT;

UPDATE apps           t SET org_id = 'org-' || t.project_id WHERE org_id IS NULL;
UPDATE user_databases t SET org_id = 'org-' || t.project_id WHERE org_id IS NULL;
UPDATE user_buckets   t SET org_id = 'org-' || t.project_id WHERE org_id IS NULL AND project_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_apps_org ON apps(org_id);
CREATE INDEX IF NOT EXISTS idx_user_databases_org ON user_databases(org_id);
CREATE INDEX IF NOT EXISTS idx_user_buckets_org ON user_buckets(org_id);
