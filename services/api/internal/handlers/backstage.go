package handlers

import (
	"encoding/json"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/gofiber/fiber/v2"
)

// BackstageHandler serves Backstage catalog entities generated from Zenith CRDs.
type BackstageHandler struct {
	k8sClient k8sclient.Client
}

// NewBackstageHandler creates a new BackstageHandler.
func NewBackstageHandler(client k8sclient.Client) *BackstageHandler {
	return &BackstageHandler{k8sClient: client}
}

// BackstageEntity represents a Backstage catalog entity.
type BackstageEntity struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   BackstageMetadata      `json:"metadata"`
	Spec       map[string]interface{} `json:"spec"`
}

// BackstageMetadata holds metadata for a Backstage entity.
type BackstageMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Description string            `json:"description,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

// BackstageCatalogResponse wraps the list of Backstage entities.
type BackstageCatalogResponse struct {
	Items []BackstageEntity `json:"items"`
	Total int               `json:"total"`
}

// GetCatalog returns all Zenith resources as Backstage catalog entities.
// GET /api/v1/backstage/catalog
func (h *BackstageHandler) GetCatalog(c *fiber.Ctx) error {
	entities, err := h.collectAllEntities(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to collect catalog entities")
	}

	return c.JSON(BackstageCatalogResponse{
		Items: entities,
		Total: len(entities),
	})
}

// GetCatalogByKind returns Backstage catalog entities filtered by kind.
// GET /api/v1/backstage/catalog/:kind
// Supported kinds: Component, Resource, API, System
func (h *BackstageHandler) GetCatalogByKind(c *fiber.Ctx) error {
	kind := c.Params("kind")
	validKinds := map[string]bool{
		"Component": true,
		"Resource":  true,
		"API":       true,
		"System":    true,
	}

	if !validKinds[kind] {
		return NewBadRequest("invalid kind: must be one of Component, Resource, API, System")
	}

	entities, err := h.collectAllEntities(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to collect catalog entities")
	}

	var filtered []BackstageEntity
	for _, e := range entities {
		if e.Kind == kind {
			filtered = append(filtered, e)
		}
	}

	if filtered == nil {
		filtered = []BackstageEntity{}
	}

	return c.JSON(BackstageCatalogResponse{
		Items: filtered,
		Total: len(filtered),
	})
}

// collectAllEntities gathers all Zenith CRDs and converts them to Backstage entities.
func (h *BackstageHandler) collectAllEntities(c *fiber.Ctx) ([]BackstageEntity, error) {
	var entities []BackstageEntity

	// List all projects first to enumerate namespaces
	projects, err := h.k8sClient.ListCRDs(c.Context(), "Project", "")
	if err != nil {
		// If we cannot list projects, return empty (the CRDs may not exist yet)
		return entities, nil
	}

	// Convert each project to a System entity
	for _, p := range projects {
		entity := projectToBackstageSystem(p)
		entities = append(entities, entity)

		namespace := "zenith-" + p.Metadata.Name
		projectName := p.Metadata.Name
		owner := p.Metadata.Labels["zenith.dev/owner"]
		if owner == "" {
			owner = "zenith"
		}

		// Collect App entities for this project
		apps, _ := h.k8sClient.ListCRDs(c.Context(), "App", namespace)
		for _, a := range apps {
			entities = append(entities, appToBackstageComponent(a, projectName, owner))
		}

		// Collect Database entities for this project
		dbs, _ := h.k8sClient.ListCRDs(c.Context(), "Database", namespace)
		for _, d := range dbs {
			entities = append(entities, databaseToBackstageResource(d, projectName, owner))
		}

		// Collect StorageBucket entities for this project
		buckets, _ := h.k8sClient.ListCRDs(c.Context(), "StorageBucket", namespace)
		for _, b := range buckets {
			entities = append(entities, storageBucketToBackstageResource(b, projectName, owner))
		}

		// Collect Domain entities for this project
		domains, _ := h.k8sClient.ListCRDs(c.Context(), "Domain", namespace)
		for _, d := range domains {
			entities = append(entities, domainToBackstageAPI(d, projectName, owner))
		}
	}

	if entities == nil {
		entities = []BackstageEntity{}
	}

	return entities, nil
}

// projectToBackstageSystem converts a Zenith Project CRD to a Backstage System entity.
func projectToBackstageSystem(p *k8sclient.CRDObject) BackstageEntity {
	var spec map[string]interface{}
	_ = json.Unmarshal(p.Spec, &spec)

	displayName, _ := spec["displayName"].(string)
	owner, _ := spec["owner"].(string)
	if owner == "" {
		owner = "zenith"
	}

	return BackstageEntity{
		APIVersion: "backstage.io/v1alpha1",
		Kind:       "System",
		Metadata: BackstageMetadata{
			Name:        p.Metadata.Name,
			Description: displayName,
			Annotations: map[string]string{
				"zenith.dev/project": p.Metadata.Name,
				"zenith.dev/type":    "project",
			},
			Labels: p.Metadata.Labels,
		},
		Spec: map[string]interface{}{
			"owner": owner,
			"type":  "project",
		},
	}
}

// appToBackstageComponent converts a Zenith App CRD to a Backstage Component entity.
func appToBackstageComponent(a *k8sclient.CRDObject, project, owner string) BackstageEntity {
	var spec map[string]interface{}
	_ = json.Unmarshal(a.Spec, &spec)

	image, _ := spec["image"].(string)
	appName := a.Metadata.Labels["zenith.dev/app-name"]
	if appName == "" {
		appName = a.Metadata.Name
	}

	tags := []string{"zenith", "app"}
	if image != "" {
		parts := strings.SplitN(image, ":", 2)
		if len(parts) > 0 {
			tags = append(tags, parts[0])
		}
	}

	providesApis := []string{}
	domain, _ := spec["domain"].(string)
	if domain != "" {
		providesApis = append(providesApis, appName+"-api")
	}

	componentSpec := map[string]interface{}{
		"type":      "service",
		"lifecycle": "production",
		"owner":     owner,
		"system":    project,
	}
	if len(providesApis) > 0 {
		componentSpec["providesApis"] = providesApis
	}

	return BackstageEntity{
		APIVersion: "backstage.io/v1alpha1",
		Kind:       "Component",
		Metadata: BackstageMetadata{
			Name: appName,
			Annotations: map[string]string{
				"zenith.dev/project": project,
				"zenith.dev/type":    "app",
				"zenith.dev/image":   image,
			},
			Labels: a.Metadata.Labels,
			Tags:   tags,
		},
		Spec: componentSpec,
	}
}

// databaseToBackstageResource converts a Zenith Database CRD to a Backstage Resource entity.
func databaseToBackstageResource(d *k8sclient.CRDObject, project, owner string) BackstageEntity {
	var spec map[string]interface{}
	_ = json.Unmarshal(d.Spec, &spec)

	engine, _ := spec["engine"].(string)
	version, _ := spec["version"].(string)

	return BackstageEntity{
		APIVersion: "backstage.io/v1alpha1",
		Kind:       "Resource",
		Metadata: BackstageMetadata{
			Name:        d.Metadata.Name,
			Description: engine + " " + version + " database",
			Annotations: map[string]string{
				"zenith.dev/project": project,
				"zenith.dev/type":    "database",
				"zenith.dev/engine":  engine,
			},
			Labels: d.Metadata.Labels,
			Tags:   []string{"zenith", "database", engine},
		},
		Spec: map[string]interface{}{
			"type":      "database",
			"owner":     owner,
			"system":    project,
			"lifecycle": "production",
		},
	}
}

// storageBucketToBackstageResource converts a Zenith StorageBucket CRD to a Backstage Resource entity.
func storageBucketToBackstageResource(b *k8sclient.CRDObject, project, owner string) BackstageEntity {
	var spec map[string]interface{}
	_ = json.Unmarshal(b.Spec, &spec)

	access, _ := spec["access"].(string)
	region, _ := spec["region"].(string)

	bucketName := b.Metadata.Labels["zenith.dev/bucket-name"]
	if bucketName == "" {
		bucketName = b.Metadata.Name
	}

	return BackstageEntity{
		APIVersion: "backstage.io/v1alpha1",
		Kind:       "Resource",
		Metadata: BackstageMetadata{
			Name:        bucketName,
			Description: "S3-compatible object storage bucket",
			Annotations: map[string]string{
				"zenith.dev/project": project,
				"zenith.dev/type":    "storage",
				"zenith.dev/access":  access,
				"zenith.dev/region":  region,
			},
			Labels: b.Metadata.Labels,
			Tags:   []string{"zenith", "storage", "s3"},
		},
		Spec: map[string]interface{}{
			"type":      "storage",
			"owner":     owner,
			"system":    project,
			"lifecycle": "production",
		},
	}
}

// domainToBackstageAPI converts a Zenith Domain CRD to a Backstage API entity.
func domainToBackstageAPI(d *k8sclient.CRDObject, project, owner string) BackstageEntity {
	var spec map[string]interface{}
	_ = json.Unmarshal(d.Spec, &spec)

	domain, _ := spec["domain"].(string)
	appRef, _ := spec["appRef"].(string)

	return BackstageEntity{
		APIVersion: "backstage.io/v1alpha1",
		Kind:       "API",
		Metadata: BackstageMetadata{
			Name:        d.Metadata.Name,
			Description: "Domain " + domain + " -> " + appRef,
			Annotations: map[string]string{
				"zenith.dev/project": project,
				"zenith.dev/type":    "domain",
				"zenith.dev/domain":  domain,
				"zenith.dev/app-ref": appRef,
			},
			Labels: d.Metadata.Labels,
			Tags:   []string{"zenith", "domain"},
		},
		Spec: map[string]interface{}{
			"type":       "openapi",
			"lifecycle":  "production",
			"owner":      owner,
			"system":     project,
			"definition": "# API served at " + domain,
		},
	}
}
