CREATE TABLE IF NOT EXISTS gateway_custom_domains (
    id TEXT PRIMARY KEY,
    gateway_id TEXT NOT NULL REFERENCES gateways(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    domain TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending',
    tls_ready BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_gw_custom_domains_gateway ON gateway_custom_domains(gateway_id);
