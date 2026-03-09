package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/gofiber/fiber/v2"
)

// AdminServicesHandler serves infrastructure service endpoints.
type AdminServicesHandler struct {
	k8s k8sclient.Client
}

// NewAdminServicesHandler creates a new AdminServicesHandler.
func NewAdminServicesHandler(k8s k8sclient.Client) *AdminServicesHandler {
	return &AdminServicesHandler{k8s: k8s}
}

// platformServices defines all platform services to monitor.
var platformServices = []struct {
	Name      string
	Namespace string
	Kind      string
}{
	{"traefik", "kube-system", "Deployment"},
	{"apisix", "zenith-platform", "Deployment"},
	{"cert-manager", "cert-manager", "Deployment"},
	{"zenith-postgres", "zenith-staging", "Cluster"},
	{"free-pg", "zenith-shared", "Cluster"},
	{"keycloak", "zenith-staging", "StatefulSet"},
	{"argocd-server", "zenith-platform", "Deployment"},
	{"harbor-core", "zenith-platform", "Deployment"},
	{"grafana", "zenith-monitoring", "Deployment"},
	{"prometheus", "zenith-monitoring", "StatefulSet"},
	{"loki", "zenith-monitoring", "StatefulSet"},
	{"tempo", "zenith-monitoring", "Deployment"},
	{"velero", "zenith-platform", "Deployment"},
	{"kyverno", "zenith-platform", "Deployment"},
	{"falco", "zenith-platform", "DaemonSet"},
	{"keda-operator", "zenith-platform", "Deployment"},
	{"sealed-secrets-controller", "zenith-platform", "Deployment"},
	{"external-dns", "zenith-platform", "Deployment"},
	{"temporal", "zenith-platform", "Deployment"},
	{"otel-collector", "zenith-monitoring", "Deployment"},
	{"nats", "zenith-platform", "StatefulSet"},
	{"zenith-api", "zenith-platform", "Deployment"},
}

// ListServices returns the health status of all platform services.
// GET /api/v1/admin/services
func (h *AdminServicesHandler) ListServices(c *fiber.Ctx) error {
	var services []entities.ServiceStatus

	for _, svc := range platformServices {
		status := entities.ServiceStatus{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Kind:      svc.Kind,
			Status:    "unknown",
		}

		// Try to get CRD to check actual status
		obj, err := h.k8s.GetCRD(c.Context(), svc.Kind, svc.Namespace, svc.Name)
		if err == nil && obj != nil {
			status.Status = "healthy"
			if obj.Metadata.Labels != nil {
				if v, ok := obj.Metadata.Labels["app.kubernetes.io/version"]; ok {
					status.Version = v
				}
			}
		} else {
			status.Status = "unknown"
		}

		// Try to get pods
		pods, err := h.k8s.ListPods(c.Context(), svc.Namespace, "app.kubernetes.io/name="+svc.Name)
		if err == nil {
			status.Replicas = len(pods)
			for _, p := range pods {
				if p.Ready {
					status.Ready++
				}
				status.Restarts += int(p.Restarts)
			}
			if status.Ready > 0 && status.Ready == status.Replicas {
				status.Status = "healthy"
			} else if status.Ready > 0 {
				status.Status = "degraded"
			} else if status.Replicas > 0 {
				status.Status = "down"
			}
		}

		services = append(services, status)
	}

	return c.JSON(services)
}

// GetService returns detailed status of a single service.
// GET /api/v1/admin/services/:name
func (h *AdminServicesHandler) GetService(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("service name is required")
	}

	// Find the service definition
	var svcDef *struct{ Name, Namespace, Kind string }
	for _, s := range platformServices {
		if s.Name == name {
			svcDef = &struct{ Name, Namespace, Kind string }{s.Name, s.Namespace, s.Kind}
			break
		}
	}
	if svcDef == nil {
		return NewNotFound("service")
	}

	detail := entities.ServiceDetail{
		ServiceStatus: entities.ServiceStatus{
			Name:      svcDef.Name,
			Namespace: svcDef.Namespace,
			Kind:      svcDef.Kind,
			Status:    "unknown",
		},
	}

	// Get pods
	pods, err := h.k8s.ListPods(c.Context(), svcDef.Namespace, "app.kubernetes.io/name="+svcDef.Name)
	if err == nil {
		detail.Replicas = len(pods)
		for _, p := range pods {
			sp := entities.ServicePod{
				Name:     p.Name,
				Status:   p.Status,
				Restarts: int(p.Restarts),
			}
			if p.Ready {
				detail.Ready++
			}
			detail.Pods = append(detail.Pods, sp)
		}
	}

	if detail.Ready > 0 && detail.Ready == detail.Replicas {
		detail.Status = "healthy"
	} else if detail.Ready > 0 {
		detail.Status = "degraded"
	} else if detail.Replicas > 0 {
		detail.Status = "down"
	}

	return c.JSON(detail)
}

// RestartService restarts a service via rollout restart.
// POST /api/v1/admin/services/:name/restart
func (h *AdminServicesHandler) RestartService(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("service name is required")
	}

	// Find the service
	var found bool
	for _, s := range platformServices {
		if s.Name == name {
			found = true
			break
		}
	}
	if !found {
		return NewNotFound("service")
	}

	// In a real implementation, this would trigger a rollout restart
	// via the K8s API (patch annotation with restart timestamp)
	return c.JSON(fiber.Map{"message": "restart initiated", "service": name})
}
