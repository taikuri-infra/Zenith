CREATE TABLE ai_usage (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    usage_type  TEXT NOT NULL,
    model       TEXT NOT NULL,
    tokens_in   INTEGER NOT NULL,
    tokens_out  INTEGER NOT NULL,
    cost_usd    DECIMAL(10,6),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_usage_user ON ai_usage(user_id);
CREATE INDEX idx_ai_usage_month ON ai_usage(user_id, created_at);
