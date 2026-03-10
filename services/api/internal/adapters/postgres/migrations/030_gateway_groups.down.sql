ALTER TABLE gateway_routes ALTER COLUMN app_subdomain SET NOT NULL;
ALTER TABLE gateway_routes ALTER COLUMN app_id SET NOT NULL;
DROP INDEX IF EXISTS idx_gw_routes_group;
ALTER TABLE gateway_routes DROP COLUMN IF EXISTS group_id;
DROP TABLE IF EXISTS gateway_groups;
