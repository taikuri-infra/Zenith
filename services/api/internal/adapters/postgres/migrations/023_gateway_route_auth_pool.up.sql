ALTER TABLE gateway_routes ADD COLUMN IF NOT EXISTS auth_pool_id TEXT REFERENCES auth_pools(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_gw_routes_auth_pool ON gateway_routes(auth_pool_id) WHERE auth_pool_id IS NOT NULL;
