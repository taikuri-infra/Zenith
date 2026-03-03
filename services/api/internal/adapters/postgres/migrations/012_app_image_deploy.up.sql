-- Add image-based deployment support to apps table.
-- deploy_source: "git" (default, backward-compatible) or "image"
-- image_url: container image reference (e.g. docker.io/user/app:latest)
-- registry_username/registry_password: for private registries

ALTER TABLE apps ADD COLUMN IF NOT EXISTS deploy_source TEXT NOT NULL DEFAULT 'git';
ALTER TABLE apps ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN IF NOT EXISTS registry_username TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN IF NOT EXISTS registry_password TEXT NOT NULL DEFAULT '';

-- Make repo_url optional (image deploys don't have one)
ALTER TABLE apps ALTER COLUMN repo_url SET DEFAULT '';
