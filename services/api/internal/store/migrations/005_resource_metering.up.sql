CREATE TABLE IF NOT EXISTS resource_usage (
    id            TEXT PRIMARY KEY,
    customer_id   TEXT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    cpu_cores     REAL NOT NULL DEFAULT 0,
    ram_gb        REAL NOT NULL DEFAULT 0,
    s3_tb         REAL NOT NULL DEFAULT 0,
    db_storage_gb REAL NOT NULL DEFAULT 0,
    volume_gb     REAL NOT NULL DEFAULT 0,
    lb_count      INTEGER NOT NULL DEFAULT 0,
    recorded_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_resource_usage_customer_recorded ON resource_usage(customer_id, recorded_at DESC);
