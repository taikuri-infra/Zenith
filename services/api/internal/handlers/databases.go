package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type DatabaseHandler struct {
	k8sClient k8sclient.Client
}

func NewDatabaseHandler(client k8sclient.Client) *DatabaseHandler {
	return &DatabaseHandler{k8sClient: client}
}

type CreateDatabaseRequest struct {
	Name     string `json:"name"`
	Engine   string `json:"engine"`
	Version  string `json:"version"`
	Storage  string `json:"storage"`
	Replicas int32  `json:"replicas,omitempty"`
}

type UpdateDatabaseRequest struct {
	Storage  string `json:"storage,omitempty"`
	Replicas int32  `json:"replicas,omitempty"`
}

type DatabaseResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ProjectID        string    `json:"project_id"`
	Engine           string    `json:"engine"`
	Version          string    `json:"version"`
	Storage          string    `json:"storage"`
	Replicas         int32     `json:"replicas"`
	Phase            string    `json:"phase"`
	Host             string    `json:"host,omitempty"`
	Port             int32     `json:"port,omitempty"`
	ConnectionString string    `json:"connection_string,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type BackupResponse struct {
	ID        string    `json:"id"`
	Database  string    `json:"database"`
	Status    string    `json:"status"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

var validEngines = map[string]map[string]bool{
	"postgresql": {"14": true, "15": true, "16": true},
	"mysql":      {"5.7": true, "8.0": true, "8.4": true},
	"mongodb":    {"6.0": true, "7.0": true},
	"redis":      {"7.0": true, "7.2": true},
}

var defaultPorts = map[string]int32{
	"postgresql": 5432,
	"mysql":      3306,
	"mongodb":    27017,
	"redis":      6379,
}

func (h *DatabaseHandler) Create(c *fiber.Ctx) error {
	projectID := c.Params("id")
	if projectID == "" {
		return NewBadRequest("project id is required")
	}

	var req CreateDatabaseRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}

	if req.Name == "" {
		return NewBadRequest("name is required")
	}
	if req.Engine == "" {
		return NewBadRequest("engine is required")
	}
	if req.Version == "" {
		return NewBadRequest("version is required")
	}
	if req.Storage == "" {
		return NewBadRequest("storage is required")
	}

	versions, ok := validEngines[req.Engine]
	if !ok {
		return NewBadRequest("invalid engine: must be postgresql, mysql, mongodb, or redis")
	}
	if !versions[req.Version] {
		return NewBadRequest(fmt.Sprintf("invalid version for %s", req.Engine))
	}

	replicas := int32(1)
	if req.Replicas > 0 {
		replicas = req.Replicas
	}

	dbID := "db-" + uuid.New().String()[:8]
	namespace := "zenith-" + projectID
	port := defaultPorts[req.Engine]
	host := fmt.Sprintf("%s.%s.svc.cluster.local", dbID, namespace)

	spec, _ := json.Marshal(map[string]interface{}{
		"engine":   req.Engine,
		"version":  req.Version,
		"storage":  req.Storage,
		"replicas": replicas,
		"name":     req.Name,
	})

	crd := &k8sclient.CRDObject{
		APIVersion: "zenith.dev/v1alpha1",
		Kind:       "Database",
		Metadata: k8sclient.ObjectMeta{
			Name:      dbID,
			Namespace: namespace,
			Labels: map[string]string{
				"zenith.dev/project": projectID,
				"zenith.dev/db-name": req.Name,
				"zenith.dev/engine":  req.Engine,
			},
		},
		Spec: spec,
	}

	if err := h.k8sClient.CreateCRD(c.Context(), crd); err != nil {
		return NewConflict("database already exists")
	}

	connStr := fmt.Sprintf("%s://user:pass@%s:%d/%s", req.Engine, host, port, req.Name)
	if req.Engine == "redis" {
		connStr = fmt.Sprintf("redis://%s:%d", host, port)
	}

	return c.Status(fiber.StatusCreated).JSON(DatabaseResponse{
		ID:               dbID,
		Name:             req.Name,
		ProjectID:        projectID,
		Engine:           req.Engine,
		Version:          req.Version,
		Storage:          req.Storage,
		Replicas:         replicas,
		Phase:            "Provisioning",
		Host:             host,
		Port:             port,
		ConnectionString: connStr,
		CreatedAt:        time.Now(),
	})
}

func (h *DatabaseHandler) List(c *fiber.Ctx) error {
	projectID := c.Params("id")
	namespace := "zenith-" + projectID

	dbs, err := h.k8sClient.ListCRDs(c.Context(), "Database", namespace)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list databases")
	}

	var result []DatabaseResponse
	for _, d := range dbs {
		result = append(result, dbCRDToResponse(d, projectID))
	}

	if result == nil {
		result = []DatabaseResponse{}
	}

	return c.JSON(fiber.Map{
		"items": result,
		"total": len(result),
	})
}

func (h *DatabaseHandler) Get(c *fiber.Ctx) error {
	projectID := c.Params("id")
	dbName := c.Params("name")
	namespace := "zenith-" + projectID

	db, err := h.k8sClient.GetCRD(c.Context(), "Database", namespace, dbName)
	if err != nil {
		return NewNotFound("database")
	}

	return c.JSON(dbCRDToResponse(db, projectID))
}

func (h *DatabaseHandler) Delete(c *fiber.Ctx) error {
	projectID := c.Params("id")
	dbName := c.Params("name")
	namespace := "zenith-" + projectID

	if _, err := h.k8sClient.GetCRD(c.Context(), "Database", namespace, dbName); err != nil {
		return NewNotFound("database")
	}

	if err := h.k8sClient.DeleteCRD(c.Context(), "Database", namespace, dbName); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete database")
	}

	return c.JSON(fiber.Map{"message": "database scheduled for deletion"})
}

func (h *DatabaseHandler) ListBackups(c *fiber.Ctx) error {
	// TODO: Implement real backup listing from CRD/S3
	return c.JSON(fiber.Map{
		"items": []BackupResponse{},
		"total": 0,
	})
}

func (h *DatabaseHandler) CreateBackup(c *fiber.Ctx) error {
	projectID := c.Params("id")
	dbName := c.Params("name")
	namespace := "zenith-" + projectID

	if _, err := h.k8sClient.GetCRD(c.Context(), "Database", namespace, dbName); err != nil {
		return NewNotFound("database")
	}

	return c.Status(fiber.StatusCreated).JSON(BackupResponse{
		ID:        "bkp-" + uuid.New().String()[:8],
		Database:  dbName,
		Status:    "in_progress",
		CreatedAt: time.Now(),
	})
}

func dbCRDToResponse(crd *k8sclient.CRDObject, projectID string) DatabaseResponse {
	var spec map[string]interface{}
	_ = json.Unmarshal(crd.Spec, &spec)

	engine, _ := spec["engine"].(string)
	version, _ := spec["version"].(string)
	storage, _ := spec["storage"].(string)
	name, _ := spec["name"].(string)
	replicas := int32(1)
	if r, ok := spec["replicas"].(float64); ok {
		replicas = int32(r)
	}

	port := defaultPorts[engine]
	host := fmt.Sprintf("%s.%s.svc.cluster.local", crd.Metadata.Name, crd.Metadata.Namespace)

	return DatabaseResponse{
		ID:        crd.Metadata.Name,
		Name:      name,
		ProjectID: projectID,
		Engine:    engine,
		Version:   version,
		Storage:   storage,
		Replicas:  replicas,
		Phase:     "Ready",
		Host:      host,
		Port:      port,
	}
}
