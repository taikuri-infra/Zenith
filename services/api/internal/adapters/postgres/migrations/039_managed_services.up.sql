-- Managed services: platform-provisioned databases and caches for customer projects
CREATE TABLE IF NOT EXISTS managed_services (
    id                TEXT PRIMARY KEY,
    project_id        TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id           TEXT NOT NULL REFERENCES users(id),
    service_type      TEXT NOT NULL,
    name              TEXT NOT NULL,
    version           TEXT NOT NULL,
    connection_url    TEXT,
    internal_host     TEXT,
    port              INTEGER,
    username          TEXT,
    password          TEXT,
    database_name     TEXT,
    k8s_namespace     TEXT,
    k8s_resource_name TEXT,
    status            TEXT NOT NULL DEFAULT 'provisioning',
    status_message    TEXT,
    storage_gb        INTEGER DEFAULT 5,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_managed_services_project ON managed_services(project_id);
CREATE INDEX IF NOT EXISTS idx_managed_services_user ON managed_services(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_managed_services_project_name ON managed_services(project_id, name);
