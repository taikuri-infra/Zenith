ALTER TABLE app_env_vars_v2 DROP CONSTRAINT IF EXISTS app_env_vars_v2_app_key_env_unique;
ALTER TABLE app_env_vars_v2 ADD CONSTRAINT app_env_vars_v2_app_id_key_key UNIQUE (app_id, key);
DROP INDEX IF EXISTS idx_env_vars_v2_env;
ALTER TABLE app_env_vars_v2 DROP COLUMN IF EXISTS environment_id;
