-- user_buckets: standalone + app-scoped S3 buckets
CREATE TABLE IF NOT EXISTS user_buckets (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL DEFAULT '',
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    access      TEXT NOT NULL DEFAULT 'private',
    region      TEXT NOT NULL DEFAULT 'fsn1',
    size_mb     INTEGER NOT NULL DEFAULT 0,
    max_size_mb INTEGER NOT NULL DEFAULT 1024,
    objects     INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'active',
    endpoint    TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_buckets_user_name ON user_buckets(user_id, name);
CREATE INDEX IF NOT EXISTS idx_user_buckets_user ON user_buckets(user_id);
CREATE INDEX IF NOT EXISTS idx_user_buckets_app ON user_buckets(app_id) WHERE app_id != '';
