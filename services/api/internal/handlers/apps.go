package handlers

import (
	"encoding/json"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AppHandler struct {
	k8sClient k8sclient.Client
}

func NewAppHandler(client k8sclient.Client) *AppHandler {
	return &AppHandler{k8sClient: client}
}

type CreateAppRequest struct {
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	Replicas *int32            `json:"replicas,omitempty"`
	Port     int32             `json:"port,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Domain   string            `json:"domain,omitempty"`
}

type UpdateAppRequest struct {
	Image    string            `json:"image,omitempty"`
	Replicas *int32            `json:"replicas,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Domain   string            `json:"domain,omitempty"`
}

type AppResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	ProjectID     string    `json:"project_id"`
	Image         string    `json:"image"`
	Replicas      int32     `json:"replicas"`
	Port          int32     `json:"port"`
	Domain        string    `json:"domain,omitempty"`
	Phase         string    `json:"phase"`
	URL           string    `json:"url,omitempty"`
	InternalURL   string    `json:"internal_url,omitempty"`
	ReadyReplicas int32     `json:"ready_replicas"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *AppHandler) Create(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return NewBadRequest("project id is required")
	}

	var req CreateAppRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	if req.Name == "" {
		return NewBadRequest("name is required")
	}
	if req.Image == "" {
		return NewBadRequest("image is required")
	}

	replicas := int32(1)
	if req.Replicas != nil {
		replicas = *req.Replicas
	}
	port := int32(8080)
	if req.Port > 0 {
		port = req.Port
	}

	appID := req.Name + "-" + uuid.New().String()[:6]
	namespace := "zenith-" + projectID

	spec, _ := json.Marshal(map[string]interface{}{
		"image":    req.Image,
		"replicas": replicas,
		"port":     port,
		"env":      req.Env,
		"domain":   req.Domain,
	})

	crd := &k8sclient.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "App",
		Metadata: k8sclient.ObjectMeta{
			Name:      appID,
			Namespace: namespace,
			Labels: map[string]string{
				"zenith.dev/project":  projectID,
				"zenith.dev/app-name": req.Name,
			},
		},
		Spec: spec,
	}

	if err := h.k8sClient.CreateCRD(c.Context(), crd); err != nil {
		return NewConflict("app already exists")
	}

	return c.Status(fiber.StatusCreated).JSON(AppResponse{
		ID:        appID,
		Name:      req.Name,
		ProjectID: projectID,
		Image:     req.Image,
		Replicas:  replicas,
		Port:      port,
		Domain:    req.Domain,
		Phase:     "Pending",
		CreatedAt: time.Now(),
	})
}

func (h *AppHandler) List(c *fiber.Ctx) error {
	projectID := c.Params("id")
	namespace := "zenith-" + projectID

	apps, err := h.k8sClient.ListCRDs(c.Context(), "App", namespace)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list apps")
	}

	var result []AppResponse
	for _, a := range apps {
		result = append(result, appCRDToResponse(a, projectID))
	}

	if result == nil {
		result = []AppResponse{}
	}

	return c.JSON(fiber.Map{
		"items": result,
		"total": len(result),
	})
}

func (h *AppHandler) Get(c *fiber.Ctx) error {
	projectID := c.Params("id")
	appName := c.Params("name")
	namespace := "zenith-" + projectID

	app, err := h.k8sClient.GetCRD(c.Context(), "App", namespace, appName)
	if err != nil {
		return NewNotFound("app")
	}

	return c.JSON(appCRDToResponse(app, projectID))
}

func (h *AppHandler) Update(c *fiber.Ctx) error {
	projectID := c.Params("id")
	appName := c.Params("name")
	namespace := "zenith-" + projectID

	var req UpdateAppRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	app, err := h.k8sClient.GetCRD(c.Context(), "App", namespace, appName)
	if err != nil {
		return NewNotFound("app")
	}

	var spec map[string]interface{}
	_ = json.Unmarshal(app.Spec, &spec)

	if req.Image != "" {
		spec["image"] = req.Image
	}
	if req.Replicas != nil {
		spec["replicas"] = *req.Replicas
	}
	if req.Env != nil {
		spec["env"] = req.Env
	}
	if req.Domain != "" {
		spec["domain"] = req.Domain
	}

	app.Spec, _ = json.Marshal(spec)

	if err := h.k8sClient.UpdateCRD(c.Context(), app); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update app")
	}

	return c.JSON(appCRDToResponse(app, projectID))
}

func (h *AppHandler) Delete(c *fiber.Ctx) error {
	projectID := c.Params("id")
	appName := c.Params("name")
	namespace := "zenith-" + projectID

	if _, err := h.k8sClient.GetCRD(c.Context(), "App", namespace, appName); err != nil {
		return NewNotFound("app")
	}

	if err := h.k8sClient.DeleteCRD(c.Context(), "App", namespace, appName); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete app")
	}

	return c.JSON(fiber.Map{"message": "app scheduled for deletion"})
}

func (h *AppHandler) Redeploy(c *fiber.Ctx) error {
	projectID := c.Params("id")
	appName := c.Params("name")
	namespace := "zenith-" + projectID

	app, err := h.k8sClient.GetCRD(c.Context(), "App", namespace, appName)
	if err != nil {
		return NewNotFound("app")
	}

	// Trigger redeploy by adding an annotation with timestamp
	if app.Metadata.Annotations == nil {
		app.Metadata.Annotations = make(map[string]string)
	}
	app.Metadata.Annotations["zenith.dev/redeploy-at"] = time.Now().Format(time.RFC3339)

	if err := h.k8sClient.UpdateCRD(c.Context(), app); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to trigger redeploy")
	}

	return c.JSON(fiber.Map{"message": "redeploy triggered"})
}

func appCRDToResponse(crd *k8sclient.CRDObject, projectID string) AppResponse {
	var spec map[string]interface{}
	_ = json.Unmarshal(crd.Spec, &spec)

	image, _ := spec["image"].(string)
	replicas := int32(1)
	if r, ok := spec["replicas"].(float64); ok {
		replicas = int32(r)
	}
	port := int32(8080)
	if p, ok := spec["port"].(float64); ok {
		port = int32(p)
	}
	domain, _ := spec["domain"].(string)

	return AppResponse{
		ID:        crd.Metadata.Name,
		Name:      crd.Metadata.Labels["zenith.dev/app-name"],
		ProjectID: projectID,
		Image:     image,
		Replicas:  replicas,
		Port:      port,
		Domain:    domain,
		Phase:     "Running",
	}
}
