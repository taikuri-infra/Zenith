-- App secrets: encrypted key-value store per app
CREATE TABLE IF NOT EXISTS app_secrets (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    key        TEXT NOT NULL,
    value_enc  BYTEA NOT NULL,  -- AES-256-GCM encrypted value
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(app_id, key)
);

CREATE INDEX idx_secrets_app ON app_secrets(app_id);
