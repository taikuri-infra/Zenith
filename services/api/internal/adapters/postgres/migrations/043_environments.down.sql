ALTER TABLE managed_services DROP COLUMN IF EXISTS environment_id;
ALTER TABLE apps DROP COLUMN IF EXISTS environment_id;
DROP TABLE IF EXISTS environments;
