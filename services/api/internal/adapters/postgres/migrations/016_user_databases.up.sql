CREATE TABLE IF NOT EXISTS user_databases (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL DEFAULT '',
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    engine      TEXT NOT NULL DEFAULT 'postgresql',
    db_name     TEXT NOT NULL,
    db_user     TEXT NOT NULL,
    host        TEXT NOT NULL DEFAULT '',
    port        INTEGER NOT NULL DEFAULT 5432,
    size_mb     INTEGER NOT NULL DEFAULT 0,
    max_size_mb INTEGER NOT NULL DEFAULT 100,
    status      TEXT NOT NULL DEFAULT 'provisioning',
    provisioner TEXT NOT NULL DEFAULT 'shared',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_databases_app_engine ON user_databases(app_id, engine);
CREATE INDEX IF NOT EXISTS idx_user_databases_user ON user_databases(user_id);
