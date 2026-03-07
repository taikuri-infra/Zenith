CREATE TABLE IF NOT EXISTS gateways (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'provisioning',
    route_count INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_gateways_slug ON gateways(slug);
CREATE INDEX idx_gateways_user ON gateways(user_id);

CREATE TABLE IF NOT EXISTS gateway_routes (
    id            TEXT PRIMARY KEY,
    gateway_id    TEXT NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    path          TEXT NOT NULL,
    methods       TEXT NOT NULL DEFAULT 'GET',
    app_id        TEXT NOT NULL,
    app_subdomain TEXT NOT NULL,
    strip_prefix  BOOLEAN NOT NULL DEFAULT FALSE,
    auth          TEXT NOT NULL DEFAULT 'none',
    plugins       JSONB NOT NULL DEFAULT '[]',
    priority      INTEGER NOT NULL DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_gw_routes_gw_name ON gateway_routes(gateway_id, name);
CREATE INDEX idx_gw_routes_gateway ON gateway_routes(gateway_id);
CREATE INDEX idx_gw_routes_app ON gateway_routes(app_id);
