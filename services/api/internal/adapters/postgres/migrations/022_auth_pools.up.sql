CREATE TABLE auth_pools (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id    TEXT NOT NULL DEFAULT '',
    name          TEXT NOT NULL,
    realm_name    TEXT NOT NULL UNIQUE,
    client_id     TEXT NOT NULL,
    client_secret TEXT NOT NULL,
    issuer_url    TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'provisioning',
    user_count    INTEGER NOT NULL DEFAULT 0,
    max_users     INTEGER NOT NULL DEFAULT 1000,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_auth_pools_user_name ON auth_pools(user_id, name);
CREATE INDEX idx_auth_pools_user ON auth_pools(user_id);
