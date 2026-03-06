-- Add s3_prefix column for prefix-based tenant isolation in shared S3 bucket.
-- Each user bucket maps to a unique prefix: u/{userID}/{bucketName}/
ALTER TABLE user_buckets ADD COLUMN IF NOT EXISTS s3_prefix TEXT NOT NULL DEFAULT '';

-- Backfill existing rows
UPDATE user_buckets SET s3_prefix = 'u/' || user_id || '/' || name || '/' WHERE s3_prefix = '';
