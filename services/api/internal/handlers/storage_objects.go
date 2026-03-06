package handlers

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// StorageObjectHandler manages object-level operations within buckets.
type StorageObjectHandler struct {
	storageRepo ports.StorageRepository
	objStorage  ports.ObjectStorage
}

// NewStorageObjectHandler creates a new StorageObjectHandler.
func NewStorageObjectHandler(storageRepo ports.StorageRepository, objStorage ports.ObjectStorage) *StorageObjectHandler {
	return &StorageObjectHandler{storageRepo: storageRepo, objStorage: objStorage}
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

	result, err := h.objStorage.ListObjects(c.Context(), bucket.Name, prefix, delimiter, 1000)
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

	expiry := 15 * time.Minute
	url, err := h.objStorage.GeneratePresignedUploadURL(c.Context(), bucket.Name, input.Key, input.ContentType, expiry)
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
	url, err := h.objStorage.GeneratePresignedDownloadURL(c.Context(), bucket.Name, key, expiry)
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

	if err := h.objStorage.DeleteObject(c.Context(), bucket.Name, key); err != nil {
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

	if err := h.objStorage.PutObject(c.Context(), bucket.Name, key, contentType, bytes.NewReader(body), size); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
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

	body, contentType, size, err := h.objStorage.GetObject(c.Context(), bucket.Name, key)
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

	if err := h.objStorage.CreateFolder(c.Context(), bucket.Name, input.Prefix); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "folder created", "prefix": input.Prefix})
}
