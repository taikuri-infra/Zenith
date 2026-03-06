DROP INDEX IF EXISTS idx_user_databases_user_name;
DROP INDEX IF EXISTS idx_user_databases_app_engine;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_databases_app_engine ON user_databases(app_id, engine);
