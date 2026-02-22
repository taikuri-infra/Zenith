-- App releases: image versions pushed by zenith-actions or CI
CREATE TABLE IF NOT EXISTS app_releases (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    image      TEXT NOT NULL,              -- full image URL incl. tag
    git_sha    TEXT NOT NULL DEFAULT '',
    branch     TEXT NOT NULL DEFAULT 'main',
    message    TEXT NOT NULL DEFAULT '',   -- commit message
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_releases_app ON app_releases(app_id, created_at DESC);
