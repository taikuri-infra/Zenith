-- Fix unique index: app-scoped DBs are one-per-engine-per-app,
-- standalone DBs are one-per-name-per-user.
DROP INDEX IF EXISTS idx_user_databases_app_engine;

-- For app-scoped databases: one engine type per app (only when app_id is non-empty)
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_databases_app_engine
    ON user_databases(app_id, engine) WHERE app_id != '';

-- For standalone databases: unique name per user
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_databases_user_name
    ON user_databases(user_id, name) WHERE app_id = '';
