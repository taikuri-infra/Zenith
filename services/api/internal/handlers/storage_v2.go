package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// StorageHandlerV2 manages per-app storage buckets (Phase 3).
type StorageHandlerV2 struct {
	storageRepo ports.StorageRepository
	appRepo     ports.AppRepository
}

// NewStorageHandlerV2 creates a new StorageHandlerV2.
func NewStorageHandlerV2(storageRepo ports.StorageRepository, appRepo ports.AppRepository) *StorageHandlerV2 {
	return &StorageHandlerV2{storageRepo: storageRepo, appRepo: appRepo}
}

// Create provisions a new storage bucket for an app.
// POST /api/v1/apps/:appId/storage
func (h *StorageHandlerV2) Create(c *fiber.Ctx) error {
	appID := c.Params("appId")
	userID, _ := c.Locals("user_id").(string)

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}
	if app.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your app")
	}

	var input dto.CreateBucketInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	bucket, err := h.storageRepo.CreateBucket(c.Context(), appID, userID, &input)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	// Auto-inject S3 endpoint as env var
	h.appRepo.SetEnvVars(c.Context(), appID, map[string]string{
		"S3_ENDPOINT": bucket.Endpoint,
		"S3_BUCKET":   bucket.Name,
	})

	return c.Status(fiber.StatusCreated).JSON(toBucketInfo(bucket))
}

// List returns all storage buckets for an app.
// GET /api/v1/apps/:appId/storage
func (h *StorageHandlerV2) List(c *fiber.Ctx) error {
	appID := c.Params("appId")

	buckets, err := h.storageRepo.ListBucketsByApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]dto.BucketInfo, len(buckets))
	for i, b := range buckets {
		result[i] = toBucketInfo(&b)
	}
	return c.JSON(result)
}

// Get returns a single storage bucket.
// GET /api/v1/apps/:appId/storage/:bucketId
func (h *StorageHandlerV2) Get(c *fiber.Ctx) error {
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}

	return c.JSON(toBucketInfo(bucket))
}

// Delete removes a storage bucket.
// DELETE /api/v1/apps/:appId/storage/:bucketId
func (h *StorageHandlerV2) Delete(c *fiber.Ctx) error {
	bucketID := c.Params("bucketId")
	userID, _ := c.Locals("user_id").(string)

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	if err := h.storageRepo.DeleteBucket(c.Context(), bucketID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Remove auto-injected env vars
	h.appRepo.DeleteEnvVar(c.Context(), bucket.AppID, "S3_ENDPOINT")
	h.appRepo.DeleteEnvVar(c.Context(), bucket.AppID, "S3_BUCKET")

	return c.JSON(fiber.Map{"message": "bucket deleted"})
}

// ListByUser returns all storage buckets for the authenticated user.
// GET /api/v1/storage
func (h *StorageHandlerV2) ListByUser(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	buckets, err := h.storageRepo.ListBucketsByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]dto.BucketInfo, len(buckets))
	for i, b := range buckets {
		result[i] = toBucketInfo(&b)
	}
	return c.JSON(result)
}

func toBucketInfo(b *entities.UserBucket) dto.BucketInfo {
	return dto.BucketInfo{
		ID:        b.ID,
		AppID:     b.AppID,
		Name:      b.Name,
		Access:    b.Access,
		Region:    b.Region,
		SizeMB:    b.SizeMB,
		MaxSizeMB: b.MaxSizeMB,
		Objects:   b.Objects,
		Status:    b.Status,
		Endpoint:  b.Endpoint,
		CreatedAt: b.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
