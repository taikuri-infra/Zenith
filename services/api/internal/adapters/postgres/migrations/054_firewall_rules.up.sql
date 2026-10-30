CREATE TABLE IF NOT EXISTS firewall_rules (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    app_id     TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    direction  TEXT NOT NULL CHECK (direction IN ('ingress', 'egress')),
    action     TEXT NOT NULL CHECK (action IN ('allow', 'deny')),
    enabled    BOOLEAN NOT NULL DEFAULT true,
    priority   INT NOT NULL DEFAULT 0,
    config     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_firewall_rules_app ON firewall_rules (app_id);
