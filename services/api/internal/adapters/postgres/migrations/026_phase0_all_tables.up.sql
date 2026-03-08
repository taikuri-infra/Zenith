-- Phase 0: Migrate all memory-only repositories to PostgreSQL
-- Tables: app_auth, backup, billing, apikey, session, mfa, webhook, role, ip, branding, sso, preview, autoscale

-- ============================================================
-- 1. App Auth (per-app built-in authentication)
-- ============================================================
CREATE TABLE IF NOT EXISTS app_auth_configs (
    app_id       TEXT PRIMARY KEY,
    enabled      BOOLEAN NOT NULL DEFAULT false,
    max_users    INTEGER NOT NULL DEFAULT 100,
    jwt_secret   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app_auth_users (
    id            TEXT PRIMARY KEY,
    app_id        TEXT NOT NULL,
    email         TEXT NOT NULL,
    name          TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    verified      BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_app_auth_users_app_email ON app_auth_users(app_id, email);
CREATE INDEX IF NOT EXISTS idx_app_auth_users_app ON app_auth_users(app_id);

-- ============================================================
-- 2. Database Backups
-- ============================================================
CREATE TABLE IF NOT EXISTS database_backups (
    id          TEXT PRIMARY KEY,
    database_id TEXT NOT NULL,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL DEFAULT 'manual',
    status      TEXT NOT NULL DEFAULT 'pending',
    size_mb     INTEGER NOT NULL DEFAULT 0,
    storage_key TEXT NOT NULL DEFAULT '',
    error       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_database_backups_database ON database_backups(database_id);
CREATE INDEX IF NOT EXISTS idx_database_backups_user ON database_backups(user_id);

-- ============================================================
-- 3. Billing (Stripe subscriptions, customer mapping, invoices)
-- ============================================================
CREATE TABLE IF NOT EXISTS subscriptions (
    id                     TEXT PRIMARY KEY,
    user_id                TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stripe_subscription_id TEXT NOT NULL DEFAULT '',
    stripe_customer_id     TEXT NOT NULL DEFAULT '',
    stripe_price_id        TEXT NOT NULL DEFAULT '',
    tier                   TEXT NOT NULL DEFAULT 'free',
    status                 TEXT NOT NULL DEFAULT 'active',
    current_period_start   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cancel_at_period_end   BOOLEAN NOT NULL DEFAULT false,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_stripe ON subscriptions(stripe_subscription_id) WHERE stripe_subscription_id != '';

CREATE TABLE IF NOT EXISTS stripe_customers (
    user_id     TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    customer_id TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS invoices (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stripe_invoice_id TEXT NOT NULL DEFAULT '',
    amount_cents     INTEGER NOT NULL DEFAULT 0,
    currency         TEXT NOT NULL DEFAULT 'eur',
    status           TEXT NOT NULL DEFAULT 'open',
    invoice_url      TEXT NOT NULL DEFAULT '',
    invoice_pdf      TEXT NOT NULL DEFAULT '',
    period_start     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period_end       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_invoices_user ON invoices(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoices_stripe ON invoices(stripe_invoice_id) WHERE stripe_invoice_id != '';

-- ============================================================
-- 4. API Keys
-- ============================================================
CREATE TABLE IF NOT EXISTS api_keys (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id  TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    key_prefix  TEXT NOT NULL DEFAULT '',
    key_hash    TEXT NOT NULL,
    scopes      TEXT[] NOT NULL DEFAULT '{}',
    type        TEXT NOT NULL DEFAULT 'personal',
    last_used_at TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);

-- ============================================================
-- 5. Sessions
-- ============================================================
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address  TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    device      TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

-- ============================================================
-- 6. MFA Enrollments
-- ============================================================
CREATE TABLE IF NOT EXISTS mfa_enrollments (
    user_id      TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'disabled',
    secret       TEXT NOT NULL DEFAULT '',
    backup_codes TEXT[] NOT NULL DEFAULT '{}',
    used_codes   TEXT[] NOT NULL DEFAULT '{}',
    enabled_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 7. User Webhooks & Deliveries
-- ============================================================
CREATE TABLE IF NOT EXISTS user_webhooks (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    events     TEXT[] NOT NULL DEFAULT '{}',
    secret     TEXT NOT NULL DEFAULT '',
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_webhooks_user ON user_webhooks(user_id);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id          TEXT PRIMARY KEY,
    webhook_id  TEXT NOT NULL REFERENCES user_webhooks(id) ON DELETE CASCADE,
    event       TEXT NOT NULL,
    payload     TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'pending',
    status_code INTEGER NOT NULL DEFAULT 0,
    error       TEXT NOT NULL DEFAULT '',
    attempts    INTEGER NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook ON webhook_deliveries(webhook_id);

-- ============================================================
-- 8. Custom Roles & Assignments (RBAC)
-- ============================================================
CREATE TABLE IF NOT EXISTS custom_roles (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_custom_roles_user ON custom_roles(user_id);

CREATE TABLE IF NOT EXISTS role_assignments (
    id          TEXT PRIMARY KEY,
    role_id     TEXT NOT NULL REFERENCES custom_roles(id) ON DELETE CASCADE,
    member_id   TEXT NOT NULL,
    assigned_by TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_role_assignments_role ON role_assignments(role_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_role_assignments_unique ON role_assignments(role_id, member_id);

-- ============================================================
-- 9. IP Whitelist
-- ============================================================
CREATE TABLE IF NOT EXISTS ip_whitelist (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cidr        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ip_whitelist_user ON ip_whitelist(user_id);

-- ============================================================
-- 10. DPA Records & Branding Config
-- ============================================================
CREATE TABLE IF NOT EXISTS dpa_records (
    user_id    TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status     TEXT NOT NULL DEFAULT 'unsigned',
    signed_by  TEXT NOT NULL DEFAULT '',
    signed_at  TIMESTAMPTZ,
    ip_address TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS branding_configs (
    user_id          TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    company_name     TEXT NOT NULL DEFAULT '',
    logo_url         TEXT NOT NULL DEFAULT '',
    primary_color    TEXT NOT NULL DEFAULT '',
    dashboard_domain TEXT NOT NULL DEFAULT '',
    domain_verified  BOOLEAN NOT NULL DEFAULT false,
    hide_branding    BOOLEAN NOT NULL DEFAULT false,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 11. SSO Configs
-- ============================================================
CREATE TABLE IF NOT EXISTS sso_configs (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL,
    entity_id     TEXT NOT NULL DEFAULT '',
    sso_url       TEXT NOT NULL DEFAULT '',
    certificate   TEXT NOT NULL DEFAULT '',
    client_id     TEXT NOT NULL DEFAULT '',
    client_secret TEXT NOT NULL DEFAULT '',
    discovery_url TEXT NOT NULL DEFAULT '',
    enabled       BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sso_configs_user ON sso_configs(user_id);

-- ============================================================
-- 12. Preview Deployments
-- ============================================================
CREATE TABLE IF NOT EXISTS preview_deployments (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL,
    pr_number  INTEGER NOT NULL DEFAULT 0,
    branch     TEXT NOT NULL DEFAULT '',
    url        TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'building',
    git_sha    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_preview_deployments_app ON preview_deployments(app_id);

-- ============================================================
-- 13. Autoscaler (nodes, events, status)
-- ============================================================
CREATE TABLE IF NOT EXISTS autoscaler_nodes (
    server_id    BIGINT PRIMARY KEY,
    name         TEXT NOT NULL DEFAULT '',
    ip           TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'provisioning',
    server_type  TEXT NOT NULL DEFAULT '',
    cpu_cores    INTEGER NOT NULL DEFAULT 0,
    ram_mb       INTEGER NOT NULL DEFAULT 0,
    monthly_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS autoscale_events (
    id          TEXT PRIMARY KEY,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    action      TEXT NOT NULL,
    old_count   INTEGER NOT NULL DEFAULT 0,
    new_count   INTEGER NOT NULL DEFAULT 0,
    reason      TEXT NOT NULL DEFAULT '',
    server_name TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_autoscale_events_time ON autoscale_events(timestamp DESC);

CREATE TABLE IF NOT EXISTS autoscaler_status (
    id             INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled        BOOLEAN NOT NULL DEFAULT false,
    node_count     INTEGER NOT NULL DEFAULT 0,
    min_nodes      INTEGER NOT NULL DEFAULT 0,
    max_nodes      INTEGER NOT NULL DEFAULT 0,
    cpu_percent    DOUBLE PRECISION NOT NULL DEFAULT 0,
    ram_percent    DOUBLE PRECISION NOT NULL DEFAULT 0,
    budget_cap_eur DOUBLE PRECISION NOT NULL DEFAULT 0,
    budget_used_eur DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_scale_up   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_scale_down TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_check_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
