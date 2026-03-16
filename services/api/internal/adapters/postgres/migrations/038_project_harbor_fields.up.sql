-- Add Harbor integration and status fields to projects table
ALTER TABLE projects ADD COLUMN IF NOT EXISTS harbor_project_name TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS harbor_robot_user TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS harbor_robot_pass TEXT;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
