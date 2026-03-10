CREATE TABLE IF NOT EXISTS gateway_groups (
    id            TEXT PRIMARY KEY,
    gateway_id    TEXT NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    app_id        TEXT NOT NULL,
    app_subdomain TEXT NOT NULL,
    plugins       JSONB NOT NULL DEFAULT '[]',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_gw_groups_gw_name ON gateway_groups(gateway_id, name);
CREATE INDEX idx_gw_groups_gateway ON gateway_groups(gateway_id);
CREATE INDEX idx_gw_groups_app ON gateway_groups(app_id);

-- Add optional group_id to gateway_routes
ALTER TABLE gateway_routes ADD COLUMN group_id TEXT REFERENCES gateway_groups(id) ON DELETE SET NULL;
CREATE INDEX idx_gw_routes_group ON gateway_routes(group_id);

-- Make app_id nullable on routes (when route belongs to a group, app comes from group)
ALTER TABLE gateway_routes ALTER COLUMN app_id DROP NOT NULL;
ALTER TABLE gateway_routes ALTER COLUMN app_subdomain DROP NOT NULL;
