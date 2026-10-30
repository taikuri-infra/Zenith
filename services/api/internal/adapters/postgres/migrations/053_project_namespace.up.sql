-- Each project runs in its own namespace. Existing projects stay in the legacy
-- shared namespace so running workloads are not moved.
ALTER TABLE projects ADD COLUMN IF NOT EXISTS namespace TEXT NOT NULL DEFAULT 'zenith-apps';
