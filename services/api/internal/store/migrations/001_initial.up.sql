-- Users table
CREATE TABLE IF NOT EXISTS users (
    id         TEXT PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL DEFAULT '',
    role       TEXT NOT NULL DEFAULT 'developer',
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- Platform settings (single-row table)
CREATE TABLE IF NOT EXISTS platform_settings (
    id              INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    platform_name   TEXT NOT NULL DEFAULT 'Zenith',
    base_domain     TEXT NOT NULL DEFAULT 'freezenith.com',
    provider        TEXT NOT NULL DEFAULT 'Hetzner Cloud',
    default_region  TEXT NOT NULL DEFAULT 'fsn1',
    region_label    TEXT NOT NULL DEFAULT 'Falkenstein',
    auto_backups    BOOLEAN NOT NULL DEFAULT true,
    retention_days  INTEGER NOT NULL DEFAULT 30
);

-- Modules table
CREATE TABLE IF NOT EXISTS modules (
    name        TEXT PRIMARY KEY,
    installed   TEXT NOT NULL,
    latest      TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'up_to_date',
    description TEXT NOT NULL DEFAULT ''
);

-- Audit log
CREATE TABLE IF NOT EXISTS audit_log (
    id         SERIAL PRIMARY KEY,
    time       TEXT NOT NULL,
    actor      TEXT NOT NULL DEFAULT 'system',
    action     TEXT NOT NULL,
    cluster    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log (created_at DESC);

-- Update history
CREATE TABLE IF NOT EXISTS update_history (
    version TEXT PRIMARY KEY,
    date    TEXT NOT NULL,
    status  TEXT NOT NULL DEFAULT 'installed'
);
