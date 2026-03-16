-- Enhanced env vars with source tracking (supplements existing app_env_vars table)
-- The existing SetEnvVars/GetEnvVars in AppRepository uses a simple key-value store.
-- This table adds source tracking for compose-import and managed-service auto-injection.
CREATE TABLE IF NOT EXISTS app_env_vars_v2 (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    is_secret   BOOLEAN NOT NULL DEFAULT false,
    source      TEXT NOT NULL DEFAULT 'manual',
    source_id   TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, key)
);

CREATE INDEX IF NOT EXISTS idx_env_vars_v2_app ON app_env_vars_v2(app_id);
CREATE INDEX IF NOT EXISTS idx_env_vars_v2_source ON app_env_vars_v2(app_id, source);
