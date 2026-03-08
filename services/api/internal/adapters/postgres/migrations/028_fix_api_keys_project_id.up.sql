-- Add project_id column to api_keys if missing (was in 026 CREATE TABLE IF NOT EXISTS
-- which was a no-op since 024 already created the table without this column)
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS project_id TEXT NOT NULL DEFAULT '';
