-- Apps: core table for user-deployed applications
CREATE TABLE IF NOT EXISTS apps (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    repo_url    TEXT NOT NULL,
    branch      TEXT NOT NULL DEFAULT 'main',
    framework   TEXT NOT NULL DEFAULT 'unknown',
    status      TEXT NOT NULL DEFAULT 'pending',
    subdomain   TEXT NOT NULL,
    port        INTEGER NOT NULL DEFAULT 8080,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_apps_user ON apps(user_id);
CREATE UNIQUE INDEX idx_apps_subdomain ON apps(subdomain);
CREATE UNIQUE INDEX idx_apps_user_name ON apps(user_id, name);

-- Deployments: deployment history per app
CREATE TABLE IF NOT EXISTS deployments (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    image_tag   TEXT DEFAULT '',
    git_sha     TEXT DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    build_log   TEXT DEFAULT '',
    error       TEXT DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deployments_app ON deployments(app_id, created_at DESC);

-- Per-app environment variables
CREATE TABLE IF NOT EXISTS app_env_vars (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(app_id, key)
);

CREATE INDEX idx_env_vars_app ON app_env_vars(app_id);
