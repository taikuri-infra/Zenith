-- Remove project_id from gateways
DROP INDEX IF EXISTS idx_gateways_project;
ALTER TABLE gateways DROP CONSTRAINT IF EXISTS fk_gateways_project;
ALTER TABLE gateways DROP COLUMN IF EXISTS project_id;

-- Remove project_id from user_buckets
DROP INDEX IF EXISTS idx_user_buckets_project;
DROP INDEX IF EXISTS idx_user_buckets_project_name;
ALTER TABLE user_buckets DROP CONSTRAINT IF EXISTS fk_user_buckets_project;
ALTER TABLE user_buckets DROP COLUMN IF EXISTS project_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_buckets_user_name ON user_buckets(user_id, name);

-- Remove project_id from user_databases
DROP INDEX IF EXISTS idx_user_databases_project;
DROP INDEX IF EXISTS idx_user_databases_project_name;
ALTER TABLE user_databases DROP CONSTRAINT IF EXISTS fk_user_databases_project;
ALTER TABLE user_databases DROP COLUMN IF EXISTS project_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_databases_user_name ON user_databases(user_id, name);

-- Remove project_id from apps
DROP INDEX IF EXISTS idx_apps_project;
DROP INDEX IF EXISTS idx_apps_project_name;
ALTER TABLE apps DROP CONSTRAINT IF EXISTS fk_apps_project;
ALTER TABLE apps DROP COLUMN IF EXISTS project_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_apps_user_name ON apps(user_id, name);

-- Drop projects table
DROP INDEX IF EXISTS idx_projects_user;
DROP INDEX IF EXISTS idx_projects_user_slug;
DROP TABLE IF EXISTS projects;
