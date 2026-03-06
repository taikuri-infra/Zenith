package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/ports"
)

type cachedUsage struct {
	bytes  int64
	expiry time.Time
}

// StorageQuotaService enforces per-user storage quotas by checking current
// usage against plan limits before allowing uploads.
type StorageQuotaService struct {
	s3             ports.ObjectStorage
	planRepo       ports.UserPlanRepository
	storageRepo    ports.StorageRepository
	platformBucket string
	cache          sync.Map // userID → *cachedUsage
	cacheTTL       time.Duration
}

// NewStorageQuotaService creates a new StorageQuotaService.
func NewStorageQuotaService(s3 ports.ObjectStorage, planRepo ports.UserPlanRepository, storageRepo ports.StorageRepository, platformBucket string) *StorageQuotaService {
	return &StorageQuotaService{
		s3:             s3,
		planRepo:       planRepo,
		storageRepo:    storageRepo,
		platformBucket: platformBucket,
		cacheTTL:       60 * time.Second,
	}
}

// CheckUploadAllowed verifies user has enough remaining quota for the upload.
func (sq *StorageQuotaService) CheckUploadAllowed(ctx context.Context, userID string, uploadSizeBytes int64) error {
	// Get plan limits (fail-closed)
	plan, err := sq.planRepo.GetUserPlan(ctx, userID)
	if err != nil {
		return fmt.Errorf("unable to verify storage quota")
	}

	maxBytes := int64(plan.Limits.MaxStorageMB) * 1024 * 1024
	if maxBytes <= 0 {
		return fmt.Errorf("storage not available on your plan")
	}

	usedBytes, err := sq.getUserUsageBytes(ctx, userID)
	if err != nil {
		return fmt.Errorf("unable to verify storage usage")
	}

	if usedBytes+uploadSizeBytes > maxBytes {
		usedMB := usedBytes / (1024 * 1024)
		maxMB := plan.Limits.MaxStorageMB
		return fmt.Errorf("storage quota exceeded: %dMB used of %dMB limit", usedMB, maxMB)
	}

	return nil
}

// InvalidateCache clears cached usage for a user.
func (sq *StorageQuotaService) InvalidateCache(userID string) {
	sq.cache.Delete(userID)
}

// getUserUsageBytes returns the total bytes used by a user, using cache when fresh.
// Queries each user bucket individually (real S3 buckets) for accurate usage.
func (sq *StorageQuotaService) getUserUsageBytes(ctx context.Context, userID string) (int64, error) {
	if cached, ok := sq.cache.Load(userID); ok {
		cu := cached.(*cachedUsage)
		if time.Now().Before(cu.expiry) {
			return cu.bytes, nil
		}
	}

	var totalBytes int64

	// If we have a storage repo, query real buckets for each user bucket
	if sq.storageRepo != nil {
		buckets, err := sq.storageRepo.ListBucketsByUser(ctx, userID)
		if err == nil {
			for _, b := range buckets {
				var s3Bucket, prefix string
				if b.S3BucketName != "" {
					s3Bucket = b.S3BucketName
					prefix = ""
				} else {
					s3Bucket = sq.platformBucket
					prefix = b.S3Prefix
				}

				result, err := sq.s3.ListObjects(ctx, s3Bucket, prefix, "", 10000)
				if err != nil {
					continue
				}
				for _, obj := range result.Objects {
					totalBytes += obj.Size
				}
			}
		}
	} else {
		// Fallback: list all objects under legacy prefix
		prefix := "u/" + userID + "/"
		result, err := sq.s3.ListObjects(ctx, sq.platformBucket, prefix, "", 10000)
		if err != nil {
			return 0, err
		}
		for _, obj := range result.Objects {
			totalBytes += obj.Size
		}
	}

	sq.cache.Store(userID, &cachedUsage{
		bytes:  totalBytes,
		expiry: time.Now().Add(sq.cacheTTL),
	})

	return totalBytes, nil
}
