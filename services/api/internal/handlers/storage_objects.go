package handlers

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// StorageObjectHandler manages object-level operations within buckets.
type StorageObjectHandler struct {
	storageRepo    ports.StorageRepository
	tenantStorage  *services.TenantStorage
	quotaSvc       *services.StorageQuotaService
}

// NewStorageObjectHandler creates a new StorageObjectHandler.
func NewStorageObjectHandler(storageRepo ports.StorageRepository, tenantStorage *services.TenantStorage, quotaSvc *services.StorageQuotaService) *StorageObjectHandler {
	return &StorageObjectHandler{storageRepo: storageRepo, tenantStorage: tenantStorage, quotaSvc: quotaSvc}
}

// ListObjects returns objects in a bucket with optional prefix/delimiter filtering.
// GET /api/v1/storage-buckets/:bucketId/objects?prefix=&delimiter=/
func (h *StorageObjectHandler) ListObjects(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	prefix := c.Query("prefix", "")
	delimiter := c.Query("delimiter", "/")

	result, err := h.tenantStorage.ListObjects(c.Context(), bucket, prefix, delimiter, 1000)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Convert to DTO
	objects := make([]dto.ObjectEntry, 0, len(result.Objects))
	for _, obj := range result.Objects {
		objects = append(objects, dto.ObjectEntry{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified.Format(time.RFC3339),
			ETag:         obj.ETag,
			IsFolder:     false,
		})
	}

	// Add common prefixes as folders
	for _, cp := range result.CommonPrefixes {
		objects = append(objects, dto.ObjectEntry{
			Key:      cp,
			IsFolder: true,
		})
	}

	return c.JSON(dto.ListObjectsResponse{
		Objects:        objects,
		CommonPrefixes: result.CommonPrefixes,
		Prefix:         result.Prefix,
		IsTruncated:    result.IsTruncated,
	})
}

// GetUploadURL generates a presigned PUT URL for uploading an object.
// POST /api/v1/storage-buckets/:bucketId/objects/upload
func (h *StorageObjectHandler) GetUploadURL(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	var input dto.UploadURLInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "key is required")
	}
	if input.Size <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "size is required and must be positive")
	}
	const maxFileSize = 100 * 1024 * 1024 // 100 MB
	if input.Size > maxFileSize {
		return fiber.NewError(fiber.StatusBadRequest, "file size exceeds 100MB limit")
	}

	// Enforce storage quota
	if h.quotaSvc != nil {
		if err := h.quotaSvc.CheckUploadAllowed(c.Context(), userID, input.Size); err != nil {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
	}

	expiry := 15 * time.Minute
	url, err := h.tenantStorage.GeneratePresignedUploadURL(c.Context(), bucket, input.Key, input.ContentType, expiry)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(dto.PresignedURLResponse{
		URL:       url,
		Method:    "PUT",
		ExpiresIn: int(expiry.Seconds()),
	})
}

// GetDownloadURL generates a presigned GET URL for downloading an object.
// GET /api/v1/storage-buckets/:bucketId/objects/download?key=
func (h *StorageObjectHandler) GetDownloadURL(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	key := c.Query("key")
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "key query parameter is required")
	}

	expiry := 15 * time.Minute
	url, err := h.tenantStorage.GeneratePresignedDownloadURL(c.Context(), bucket, key, expiry)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(dto.PresignedURLResponse{
		URL:       url,
		Method:    "GET",
		ExpiresIn: int(expiry.Seconds()),
	})
}

// DeleteObject deletes an object from a bucket.
// DELETE /api/v1/storage-buckets/:bucketId/objects?key=
func (h *StorageObjectHandler) DeleteObject(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	key := c.Query("key")
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "key query parameter is required")
	}

	if err := h.tenantStorage.DeleteObject(c.Context(), bucket, key); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "object deleted"})
}

// UploadObject accepts a raw file body and proxies it to S3.
// PUT /api/v1/storage-buckets/:bucketId/objects/content?key=
func (h *StorageObjectHandler) UploadObject(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	key := c.Query("key")
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "key query parameter is required")
	}

	contentType := c.Get("Content-Type", "application/octet-stream")
	body := c.Body()
	size := int64(len(body))

	// Enforce storage quota
	if h.quotaSvc != nil {
		if err := h.quotaSvc.CheckUploadAllowed(c.Context(), userID, size); err != nil {
			return fiber.NewError(fiber.StatusForbidden, err.Error())
		}
	}

	if err := h.tenantStorage.PutObject(c.Context(), bucket, key, contentType, bytes.NewReader(body), size); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if h.quotaSvc != nil {
		h.quotaSvc.InvalidateCache(userID)
	}

	return c.JSON(fiber.Map{"message": "object uploaded", "key": key})
}

// DownloadObject streams an object from S3 through the API.
// GET /api/v1/storage-buckets/:bucketId/objects/content?key=
func (h *StorageObjectHandler) DownloadObject(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	key := c.Query("key")
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "key query parameter is required")
	}

	body, contentType, size, err := h.tenantStorage.GetObject(c.Context(), bucket, key)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer body.Close()

	if contentType != "" {
		c.Set("Content-Type", contentType)
	}
	if size > 0 {
		c.Set("Content-Length", fmt.Sprintf("%d", size))
	}
	parts := strings.Split(key, "/")
	filename := parts[len(parts)-1]
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.SendStream(body, int(size))
}

// CreateFolder creates a folder (zero-byte object with trailing /) in a bucket.
// POST /api/v1/storage-buckets/:bucketId/objects/folder
func (h *StorageObjectHandler) CreateFolder(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	bucketID := c.Params("bucketId")

	bucket, err := h.storageRepo.GetBucket(c.Context(), bucketID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "bucket not found")
	}
	if bucket.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your bucket")
	}

	var input dto.CreateFolderInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Prefix == "" {
		return fiber.NewError(fiber.StatusBadRequest, "prefix is required")
	}

	// Ensure prefix ends with /
	if !strings.HasSuffix(input.Prefix, "/") {
		input.Prefix += "/"
	}

	if err := h.tenantStorage.CreateFolder(c.Context(), bucket, input.Prefix); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "folder created", "prefix": input.Prefix})
}
