-- Deploy tokens: secure tokens for CI/CD (GitHub Actions) with scoped permissions.
-- Separate from api_keys — deploy tokens use Argon2id hashing and token_id/secret split.
CREATE TABLE IF NOT EXISTS deploy_tokens (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id          TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    token_id            TEXT NOT NULL UNIQUE,
    token_prefix        TEXT NOT NULL,
    token_hash          TEXT NOT NULL,
    scopes              TEXT[] NOT NULL DEFAULT '{}',
    last_used_at        TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ,
    previous_hash       TEXT,
    previous_expires_at TIMESTAMPTZ,
    rotated_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_deploy_tokens_token_id ON deploy_tokens(token_id);
CREATE INDEX IF NOT EXISTS idx_deploy_tokens_user ON deploy_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_deploy_tokens_project ON deploy_tokens(project_id);
