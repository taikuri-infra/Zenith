package services

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// TenantStorage wraps an ObjectStorage client and routes all operations through
// either a real per-customer S3 bucket (when S3BucketName is set) or the shared
// platform bucket using prefix-based isolation.
// Users never get direct S3 credentials — all access is API-proxied.
type TenantStorage struct {
	s3             ports.ObjectStorage
	platformBucket string
}

// NewTenantStorage creates a new TenantStorage.
func NewTenantStorage(s3 ports.ObjectStorage, platformBucket string) *TenantStorage {
	return &TenantStorage{s3: s3, platformBucket: platformBucket}
}

// validateKey rejects path-traversal attempts and absolute paths in object keys.
func validateKey(key string) error {
	if key == "" {
		return fmt.Errorf("object key is required")
	}
	if strings.Contains(key, "..") {
		return fmt.Errorf("invalid object key: path traversal not allowed")
	}
	if strings.HasPrefix(key, "/") {
		return fmt.Errorf("invalid object key: must not start with /")
	}
	return nil
}

// bucketAndKey returns the real S3 bucket name and object key for a given user bucket.
// If the bucket has a real S3BucketName, use it directly with the raw key.
// Otherwise, fall back to the shared platform bucket with prefix isolation.
func (ts *TenantStorage) bucketAndKey(bucket *entities.UserBucket, key string) (string, string) {
	if bucket.S3BucketName != "" {
		return bucket.S3BucketName, key
	}
	return ts.platformBucket, bucket.S3Prefix + key
}

// ListObjects lists objects in the bucket, stripping prefixes from returned keys
// so callers see user-relative paths.
func (ts *TenantStorage) ListObjects(ctx context.Context, bucket *entities.UserBucket, prefix, delimiter string, maxKeys int) (*ports.ObjectListResult, error) {
	if bucket.S3BucketName != "" {
		// Real bucket: list directly with user prefix
		result, err := ts.s3.ListObjects(ctx, bucket.S3BucketName, prefix, delimiter, maxKeys)
		if err != nil {
			return nil, err
		}
		result.Prefix = prefix
		return result, nil
	}

	// Legacy: shared bucket with prefix isolation
	fullPrefix := bucket.S3Prefix + prefix

	result, err := ts.s3.ListObjects(ctx, ts.platformBucket, fullPrefix, delimiter, maxKeys)
	if err != nil {
		return nil, err
	}

	// Strip the bucket's S3 prefix from returned keys and common prefixes.
	for i := range result.Objects {
		result.Objects[i].Key = strings.TrimPrefix(result.Objects[i].Key, bucket.S3Prefix)
	}
	for i := range result.CommonPrefixes {
		result.CommonPrefixes[i] = strings.TrimPrefix(result.CommonPrefixes[i], bucket.S3Prefix)
	}
	result.Prefix = prefix

	return result, nil
}

// PutObject uploads an object.
func (ts *TenantStorage) PutObject(ctx context.Context, bucket *entities.UserBucket, key, contentType string, body io.Reader, size int64) error {
	if err := validateKey(key); err != nil {
		return err
	}
	s3Bucket, s3Key := ts.bucketAndKey(bucket, key)
	return ts.s3.PutObject(ctx, s3Bucket, s3Key, contentType, body, size)
}

// GetObject downloads an object.
func (ts *TenantStorage) GetObject(ctx context.Context, bucket *entities.UserBucket, key string) (io.ReadCloser, string, int64, error) {
	if err := validateKey(key); err != nil {
		return nil, "", 0, err
	}
	s3Bucket, s3Key := ts.bucketAndKey(bucket, key)
	return ts.s3.GetObject(ctx, s3Bucket, s3Key)
}

// DeleteObject removes an object.
func (ts *TenantStorage) DeleteObject(ctx context.Context, bucket *entities.UserBucket, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	s3Bucket, s3Key := ts.bucketAndKey(bucket, key)
	return ts.s3.DeleteObject(ctx, s3Bucket, s3Key)
}

// CreateFolder creates a zero-byte folder marker.
func (ts *TenantStorage) CreateFolder(ctx context.Context, bucket *entities.UserBucket, prefix string) error {
	if err := validateKey(prefix); err != nil {
		return err
	}
	s3Bucket, s3Key := ts.bucketAndKey(bucket, prefix)
	return ts.s3.CreateFolder(ctx, s3Bucket, s3Key)
}

// GeneratePresignedUploadURL creates a presigned PUT URL.
func (ts *TenantStorage) GeneratePresignedUploadURL(ctx context.Context, bucket *entities.UserBucket, key, contentType string, expiry time.Duration) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	s3Bucket, s3Key := ts.bucketAndKey(bucket, key)
	return ts.s3.GeneratePresignedUploadURL(ctx, s3Bucket, s3Key, contentType, expiry)
}

// GeneratePresignedDownloadURL creates a presigned GET URL.
func (ts *TenantStorage) GeneratePresignedDownloadURL(ctx context.Context, bucket *entities.UserBucket, key string, expiry time.Duration) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	s3Bucket, s3Key := ts.bucketAndKey(bucket, key)
	return ts.s3.GeneratePresignedDownloadURL(ctx, s3Bucket, s3Key, expiry)
}

// DeleteAllForBucket removes every object stored for this bucket.
// Used when a user deletes a bucket.
func (ts *TenantStorage) DeleteAllForBucket(ctx context.Context, bucket *entities.UserBucket) error {
	var s3Bucket, listPrefix string
	if bucket.S3BucketName != "" {
		s3Bucket = bucket.S3BucketName
		listPrefix = "" // list all objects in the real bucket
	} else {
		s3Bucket = ts.platformBucket
		listPrefix = bucket.S3Prefix
	}

	result, err := ts.s3.ListObjects(ctx, s3Bucket, listPrefix, "", 1000)
	if err != nil {
		return fmt.Errorf("list objects for cleanup: %w", err)
	}
	for _, obj := range result.Objects {
		if err := ts.s3.DeleteObject(ctx, s3Bucket, obj.Key); err != nil {
			return fmt.Errorf("delete object %s: %w", obj.Key, err)
		}
	}
	return nil
}
