-- Phase 6: Stripe Billing

ALTER TABLE users ADD COLUMN IF NOT EXISTS stripe_customer_id TEXT;
CREATE INDEX IF NOT EXISTS idx_users_stripe_customer_id ON users(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS subscriptions (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL REFERENCES users(id),
    stripe_subscription_id TEXT UNIQUE NOT NULL,
    stripe_customer_id     TEXT NOT NULL,
    stripe_price_id        TEXT NOT NULL,
    tier            TEXT NOT NULL DEFAULT 'free',
    status          TEXT NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMPTZ,
    current_period_end   TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);

CREATE TABLE IF NOT EXISTS invoices (
    id                TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL REFERENCES users(id),
    stripe_invoice_id TEXT UNIQUE NOT NULL,
    amount_cents      INTEGER NOT NULL DEFAULT 0,
    currency          TEXT NOT NULL DEFAULT 'eur',
    status            TEXT NOT NULL DEFAULT 'draft',
    invoice_url       TEXT,
    invoice_pdf       TEXT,
    period_start      TIMESTAMPTZ,
    period_end        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invoices_user_id ON invoices(user_id);
