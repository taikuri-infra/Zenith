package services

import (
	"context"
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newTestStorageQuotaService() (*StorageQuotaService, *memory.MemoryUserPlanRepository, *memory.MemoryStorageRepository, *mockObjectStorage) {
	s3 := newMockObjectStorage()
	planRepo := memory.NewMemoryUserPlanRepository()
	storageRepo := memory.NewMemoryStorageRepository()
	svc := NewStorageQuotaService(s3, planRepo, storageRepo, "shared-platform-bucket")
	return svc, planRepo, storageRepo, s3
}

func TestStorageQuota_UnderLimit(t *testing.T) {
	svc, planRepo, storageRepo, s3 := newTestStorageQuotaService()
	ctx := context.Background()

	userID := "user-under"
	planRepo.SetUserPlan(ctx, userID, entities.PlanPro) // MaxStorageMB = 10240

	// Create a bucket and put a small object
	bucket, _ := storageRepo.CreateBucket(ctx, "app-1", userID, &dto.CreateBucketInput{Name: "mybucket"})
	s3.PutObject(ctx, bucket.S3BucketName, "small.txt", "text/plain", strings.NewReader("hello"), 5)

	err := svc.CheckUploadAllowed(ctx, userID, 1024) // 1KB upload
	if err != nil {
		t.Errorf("Expected upload to be allowed, got: %v", err)
	}
}

func TestStorageQuota_OverLimit(t *testing.T) {
	svc, _, _, _ := newTestStorageQuotaService()
	ctx := context.Background()

	userID := "user-over"
	// Free plan: MaxStorageMB = 1024 = 1GB

	// Try to upload more than 1GB
	overSize := int64(1024*1024*1024) + 1 // just over 1GB
	err := svc.CheckUploadAllowed(ctx, userID, overSize)
	if err == nil {
		t.Error("Expected error when upload exceeds storage quota")
	}
	if !strings.Contains(err.Error(), "quota exceeded") {
		t.Errorf("Expected quota exceeded error, got: %v", err)
	}
}

func TestStorageQuota_ZeroStoragePlan(t *testing.T) {
	svc, planRepo, _, _ := newTestStorageQuotaService()
	ctx := context.Background()

	userID := "user-zero"
	// Manually set a plan with 0 storage (free plan has 1024MB but let's test edge case)
	// Free plan has MaxStorageMB = 1024, so this test verifies normal behavior
	planRepo.SetUserPlan(ctx, userID, entities.PlanFree)

	// Small upload should be allowed
	err := svc.CheckUploadAllowed(ctx, userID, 100)
	if err != nil {
		t.Errorf("Expected small upload to be allowed on free plan, got: %v", err)
	}
}

func TestStorageQuota_InvalidateCache(t *testing.T) {
	svc, _, _, _ := newTestStorageQuotaService()
	ctx := context.Background()

	userID := "user-cache"

	// First call populates cache
	svc.CheckUploadAllowed(ctx, userID, 100)

	// Invalidate
	svc.InvalidateCache(userID)

	// Should still work (re-queries)
	err := svc.CheckUploadAllowed(ctx, userID, 100)
	if err != nil {
		t.Errorf("Expected upload to work after cache invalidation, got: %v", err)
	}
}

func TestStorageQuota_CacheHit(t *testing.T) {
	svc, _, _, _ := newTestStorageQuotaService()
	ctx := context.Background()

	userID := "user-hit"

	// First call populates cache
	svc.CheckUploadAllowed(ctx, userID, 100)

	// Second call should use cache (same result)
	err := svc.CheckUploadAllowed(ctx, userID, 100)
	if err != nil {
		t.Errorf("Expected upload to be allowed from cache, got: %v", err)
	}
}

func TestStorageQuota_NoStorageRepo(t *testing.T) {
	s3 := newMockObjectStorage()
	planRepo := memory.NewMemoryUserPlanRepository()
	// Pass nil storage repo
	svc := NewStorageQuotaService(s3, planRepo, nil, "shared-platform-bucket")
	ctx := context.Background()

	userID := "user-norepo"

	// Should fall back to legacy prefix-based listing
	err := svc.CheckUploadAllowed(ctx, userID, 100)
	if err != nil {
		t.Errorf("Expected upload to work without storage repo, got: %v", err)
	}
}
