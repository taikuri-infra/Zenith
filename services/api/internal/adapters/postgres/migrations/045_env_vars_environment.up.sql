-- Add environment_id to app_env_vars_v2 for per-environment env vars.
-- NULL = production / default (backward compatible with existing rows).
ALTER TABLE app_env_vars_v2
    ADD COLUMN IF NOT EXISTS environment_id TEXT REFERENCES environments(id) ON DELETE CASCADE;

-- Drop old unique constraint (app_id, key) — same key can exist per environment
ALTER TABLE app_env_vars_v2 DROP CONSTRAINT IF EXISTS app_env_vars_v2_app_id_key_key;

-- New unique constraint: one value per (app, key, environment)
-- NULL environment_id treated as distinct from non-NULL via NULLS NOT DISTINCT
ALTER TABLE app_env_vars_v2
    ADD CONSTRAINT app_env_vars_v2_app_key_env_unique
    UNIQUE NULLS NOT DISTINCT (app_id, key, environment_id);

CREATE INDEX IF NOT EXISTS idx_env_vars_v2_env ON app_env_vars_v2(app_id, environment_id);
