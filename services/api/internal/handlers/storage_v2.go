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
	objStorage  ports.ObjectStorage
}

// NewStorageHandlerV2 creates a new StorageHandlerV2.
func NewStorageHandlerV2(storageRepo ports.StorageRepository, appRepo ports.AppRepository) *StorageHandlerV2 {
	return &StorageHandlerV2{storageRepo: storageRepo, appRepo: appRepo}
}

// SetObjectStorage wires the S3 client for real bucket operations.
func (h *StorageHandlerV2) SetObjectStorage(s ports.ObjectStorage) {
	h.objStorage = s
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

// CreateStandalone provisions a standalone bucket (not tied to an app).
// POST /api/v1/storage-buckets
func (h *StorageHandlerV2) CreateStandalone(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var input dto.CreateBucketInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	bucket, err := h.storageRepo.CreateBucket(c.Context(), "", userID, &input)
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	// Create real S3 bucket if S3 client is available
	if h.objStorage != nil {
		if err := h.objStorage.CreateBucket(c.Context(), bucket.Name); err != nil {
			// Don't fail the API call, bucket is stored but S3 may lag
			bucket.Status = entities.BucketStatusError
		}
	}

	return c.Status(fiber.StatusCreated).JSON(toBucketInfo(bucket))
}

// GetStandalone returns a single standalone bucket.
// GET /api/v1/storage-buckets/:bucketId
func (h *StorageHandlerV2) GetStandalone(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	return c.JSON(toBucketInfo(bucket))
}

// DeleteStandalone removes a standalone bucket.
// DELETE /api/v1/storage-buckets/:bucketId
func (h *StorageHandlerV2) DeleteStandalone(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

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

	// Delete real S3 bucket if S3 client is available
	if h.objStorage != nil {
		h.objStorage.DeleteBucket(c.Context(), bucket.Name)
	}

	return c.JSON(fiber.Map{"message": "bucket deleted"})
}

// UpdateBucket updates a bucket's access setting.
// PUT /api/v1/storage-buckets/:bucketId
func (h *StorageHandlerV2) UpdateBucket(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	var input dto.UpdateBucketInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	updated, err := h.storageRepo.UpdateBucket(c.Context(), bucketID, input.Access)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(toBucketInfo(updated))
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
