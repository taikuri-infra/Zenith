CREATE TABLE referral_rewards (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_id     TEXT NOT NULL REFERENCES users(id),
    referred_id     TEXT NOT NULL REFERENCES users(id),
    status          TEXT NOT NULL DEFAULT 'pending',
    reward_type     TEXT NOT NULL DEFAULT 'pro_month',
    reward_amount   INT DEFAULT 0,
    credited_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(referrer_id, referred_id)
);
CREATE INDEX idx_referral_rewards_referrer ON referral_rewards(referrer_id);
