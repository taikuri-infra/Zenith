CREATE TABLE email_sends (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    template_key TEXT NOT NULL,
    sent_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    opened_at    TIMESTAMPTZ,
    clicked_at   TIMESTAMPTZ,
    UNIQUE(user_id, template_key)
);
CREATE INDEX idx_email_sends_user ON email_sends(user_id);
CREATE INDEX idx_email_sends_template ON email_sends(template_key);
