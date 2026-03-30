-- Add replicas column to apps for configurable replica count per app.
-- Defaults to 1 to preserve existing behaviour.
ALTER TABLE apps ADD COLUMN IF NOT EXISTS replicas INT NOT NULL DEFAULT 1;
