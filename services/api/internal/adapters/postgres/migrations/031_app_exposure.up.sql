-- Add exposure and auto-gateway columns to apps table.
-- exposure: "public" (frontend) or "protected" (API behind gateway auth)
-- auto_gateway_id: references the auto-created per-project gateway
ALTER TABLE apps ADD COLUMN IF NOT EXISTS exposure TEXT NOT NULL DEFAULT 'public';
ALTER TABLE apps ADD COLUMN IF NOT EXISTS auto_gateway_id TEXT REFERENCES gateways(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_apps_auto_gateway ON apps(auto_gateway_id);

-- CHECK constraint: exposure can only be 'public' or 'protected'
DO $$ BEGIN
  ALTER TABLE apps ADD CONSTRAINT chk_apps_exposure CHECK (exposure IN ('public', 'protected'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Unique constraint on subdomain to prevent race-condition slug collisions
CREATE UNIQUE INDEX IF NOT EXISTS idx_apps_subdomain ON apps(subdomain);

-- Unique constraint: one auto-gateway per project (prevents TOCTOU in EnsureProjectGateway)
-- Only applies to gateways created via auto-gateway flow (name ends with " Gateway")
-- Note: the gateways table likely already has a unique on slug, but we also need per-project uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idx_gateways_project_id ON gateways(project_id) WHERE project_id IS NOT NULL AND project_id != '';
