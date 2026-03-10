DROP INDEX IF EXISTS idx_gateways_project_id;
DROP INDEX IF EXISTS idx_apps_subdomain;
ALTER TABLE apps DROP CONSTRAINT IF EXISTS chk_apps_exposure;
DROP INDEX IF EXISTS idx_apps_auto_gateway;
ALTER TABLE apps DROP COLUMN IF EXISTS auto_gateway_id;
ALTER TABLE apps DROP COLUMN IF EXISTS exposure;
