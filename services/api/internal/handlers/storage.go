package handlers

import (
	"encoding/json"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/k8s"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type StorageHandler struct {
	k8sClient k8s.Client
}

func NewStorageHandler(client k8s.Client) *StorageHandler {
	return &StorageHandler{k8sClient: client}
}

type CreateStorageRequest struct {
	Name       string `json:"name"`
	Access     string `json:"access,omitempty"`
	Versioning bool   `json:"versioning,omitempty"`
	Region     string `json:"region,omitempty"`
}

type StorageResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ProjectID   string    `json:"project_id"`
	Access      string    `json:"access"`
	Versioning  bool      `json:"versioning"`
	Region      string    `json:"region"`
	Phase       string    `json:"phase"`
	Endpoint    string    `json:"endpoint,omitempty"`
	SizeBytes   int64     `json:"size_bytes"`
	ObjectCount int64     `json:"object_count"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *StorageHandler) Create(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return NewBadRequest("project id is required")
	}

	var req CreateStorageRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	if req.Name == "" {
		return NewBadRequest("name is required")
	}

	if req.Access == "" {
		req.Access = "private"
	}
	if req.Access != "private" && req.Access != "public-read" {
		return NewBadRequest("access must be 'private' or 'public-read'")
	}
	if req.Region == "" {
		req.Region = "fsn1"
	}

	bucketID := "sb-" + uuid.New().String()[:8]
	namespace := "zenith-" + projectID

	spec, _ := json.Marshal(map[string]interface{}{
		"access":     req.Access,
		"versioning": req.Versioning,
		"region":     req.Region,
		"name":       req.Name,
	})

	crd := &k8s.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "StorageBucket",
		Metadata: k8s.ObjectMeta{
			Name:      bucketID,
			Namespace: namespace,
			Labels: map[string]string{
				"zenith.dev/project":     projectID,
				"zenith.dev/bucket-name": req.Name,
			},
		},
		Spec: spec,
	}

	if err := h.k8sClient.CreateCRD(c.Context(), crd); err != nil {
		return NewConflict("storage bucket already exists")
	}

	return c.Status(fiber.StatusCreated).JSON(StorageResponse{
		ID:         bucketID,
		Name:       req.Name,
		ProjectID:  projectID,
		Access:     req.Access,
		Versioning: req.Versioning,
		Region:     req.Region,
		Phase:      "Creating",
		CreatedAt:  time.Now(),
	})
}

func (h *StorageHandler) List(c *fiber.Ctx) error {
	projectID := c.Params("id")
	namespace := "zenith-" + projectID

	buckets, err := h.k8sClient.ListCRDs(c.Context(), "StorageBucket", namespace)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list storage buckets")
	}

	var result []StorageResponse
	for _, b := range buckets {
		result = append(result, storageCRDToResponse(b, projectID))
	}

	if result == nil {
		result = []StorageResponse{}
	}

	return c.JSON(fiber.Map{
		"items": result,
		"total": len(result),
	})
}

func (h *StorageHandler) Get(c *fiber.Ctx) error {
	projectID := c.Params("id")
	bucketName := c.Params("name")
	namespace := "zenith-" + projectID

	bucket, err := h.k8sClient.GetCRD(c.Context(), "StorageBucket", namespace, bucketName)
	if err != nil {
		return NewNotFound("storage bucket")
	}

	return c.JSON(storageCRDToResponse(bucket, projectID))
}

func (h *StorageHandler) Delete(c *fiber.Ctx) error {
	projectID := c.Params("id")
	bucketName := c.Params("name")
	namespace := "zenith-" + projectID

	if _, err := h.k8sClient.GetCRD(c.Context(), "StorageBucket", namespace, bucketName); err != nil {
		return NewNotFound("storage bucket")
	}

	if err := h.k8sClient.DeleteCRD(c.Context(), "StorageBucket", namespace, bucketName); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete storage bucket")
	}

	return c.JSON(fiber.Map{"message": "storage bucket scheduled for deletion"})
}

func storageCRDToResponse(crd *k8s.CRDObject, projectID string) StorageResponse {
	var spec map[string]interface{}
	_ = json.Unmarshal(crd.Spec, &spec)

	name, _ := spec["name"].(string)
	access, _ := spec["access"].(string)
	versioning, _ := spec["versioning"].(bool)
	region, _ := spec["region"].(string)

	return StorageResponse{
		ID:         crd.Metadata.Name,
		Name:       name,
		ProjectID:  projectID,
		Access:     access,
		Versioning: versioning,
		Region:     region,
		Phase:      "Ready",
	}
}
