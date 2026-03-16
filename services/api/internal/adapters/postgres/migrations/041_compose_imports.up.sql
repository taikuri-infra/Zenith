-- Compose import audit trail: records every docker-compose parse attempt
CREATE TABLE IF NOT EXISTS compose_imports (
    id              TEXT PRIMARY KEY,
    project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL,
    compose_content TEXT NOT NULL,
    parsed_result   JSONB NOT NULL,
    ai_result       JSONB,
    status          TEXT NOT NULL DEFAULT 'success',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compose_imports_project ON compose_imports(project_id);
CREATE INDEX IF NOT EXISTS idx_compose_imports_user ON compose_imports(user_id);
