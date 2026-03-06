-- Add s3_bucket_name for per-customer real S3 buckets.
-- When non-empty, objects are stored in this real S3 bucket instead of the shared platform bucket.
ALTER TABLE user_buckets ADD COLUMN IF NOT EXISTS s3_bucket_name TEXT NOT NULL DEFAULT '';
