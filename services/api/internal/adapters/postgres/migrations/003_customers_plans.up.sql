-- Plans table
CREATE TABLE IF NOT EXISTS plans (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL UNIQUE,
    cpu_cores      INTEGER NOT NULL,
    ram_gb         INTEGER NOT NULL,
    s3_tb          INTEGER NOT NULL DEFAULT 0,
    db_storage_gb  INTEGER NOT NULL DEFAULT 0,
    volume_gb      INTEGER NOT NULL DEFAULT 0,
    lb_count       INTEGER NOT NULL DEFAULT 0,
    price_cents    INTEGER NOT NULL,
    currency       TEXT NOT NULL DEFAULT 'EUR',
    billing_cycle  TEXT NOT NULL DEFAULT 'monthly',
    active         BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Customers table
CREATE TABLE IF NOT EXISTS customers (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    domain         TEXT NOT NULL UNIQUE,
    plan_id        TEXT NOT NULL REFERENCES plans(id),
    contact_email  TEXT NOT NULL,
    contact_name   TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'active',
    cluster_status TEXT NOT NULL DEFAULT 'pending',
    notes          TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_customers_status ON customers(status);
CREATE INDEX IF NOT EXISTS idx_customers_plan_id ON customers(plan_id);
