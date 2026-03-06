package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// BucketService manages real S3 bucket lifecycle operations.
type BucketService struct {
	s3 ports.ObjectStorage
}

// NewBucketService creates a new BucketService.
func NewBucketService(s3 ports.ObjectStorage) *BucketService {
	return &BucketService{s3: s3}
}

// GenerateS3BucketName returns a deterministic real S3 bucket name for a user.
// Format: zenith-{userID[:12]}-{bucketName}  (max 63 chars per S3 rules).
func GenerateS3BucketName(userID, name string) string {
	uid := userID
	if len(uid) > 12 {
		uid = uid[:12]
	}
	// Remove non-alphanumeric/hyphen characters from userID segment
	uid = sanitizeBucketSegment(uid)
	result := "zenith-" + uid + "-" + name
	if len(result) > 63 {
		result = result[:63]
	}
	return strings.ToLower(result)
}

// sanitizeBucketSegment keeps only lowercase alphanumeric and hyphens.
func sanitizeBucketSegment(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// CreateRealBucket creates a real S3 bucket.
func (bs *BucketService) CreateRealBucket(ctx context.Context, bucketName string) error {
	return bs.s3.CreateBucket(ctx, bucketName)
}

// DeleteRealBucket empties and deletes a real S3 bucket.
func (bs *BucketService) DeleteRealBucket(ctx context.Context, bucketName string) error {
	// List and delete all objects first
	result, err := bs.s3.ListObjects(ctx, bucketName, "", "", 10000)
	if err != nil {
		return fmt.Errorf("list objects for cleanup: %w", err)
	}
	for _, obj := range result.Objects {
		if err := bs.s3.DeleteObject(ctx, bucketName, obj.Key); err != nil {
			return fmt.Errorf("delete object %s: %w", obj.Key, err)
		}
	}

	return bs.s3.DeleteBucket(ctx, bucketName)
}
