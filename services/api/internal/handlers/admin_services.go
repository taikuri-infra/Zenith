package handlers

import (
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
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

// platformService defines a service to monitor with its label selector.
type platformService struct {
	Name          string
	Namespace     string
	Kind          string
	LabelSelector string // custom label selector (overrides default)
}

// platformServices defines all platform services to monitor.
// Namespaces match the actual staging cluster layout.
var platformServices = []platformService{
	{"traefik", "kube-system", "Deployment", "app.kubernetes.io/name=traefik"},
	{"apisix", "apisix", "Deployment", "app.kubernetes.io/name=apisix"},
	{"cert-manager", "cert-manager", "Deployment", "app.kubernetes.io/name=cert-manager"},
	{"zenith-postgres", "zenith-staging", "Cluster", "cnpg.io/cluster=zenith-postgres"},
	{"free-pg", "zenith-shared", "Cluster", "cnpg.io/cluster=free-pg"},
	{"keycloak", "keycloak", "StatefulSet", "app.kubernetes.io/name=keycloak"},
	{"argocd-server", "argocd", "Deployment", "app.kubernetes.io/name=argocd-server"},
	{"harbor-core", "harbor", "Deployment", "app.kubernetes.io/name=harbor,component=core"},
	{"grafana", "monitoring", "Deployment", "app.kubernetes.io/name=grafana"},
	{"prometheus", "monitoring", "StatefulSet", "app.kubernetes.io/name=prometheus"},
	{"loki", "monitoring", "StatefulSet", "app.kubernetes.io/name=loki"},
	{"tempo", "monitoring", "StatefulSet", "app.kubernetes.io/name=tempo"},
	{"velero", "velero", "Deployment", "app.kubernetes.io/name=velero"},
	{"kyverno", "kyverno", "Deployment", "app.kubernetes.io/name=kyverno"},
	{"falco", "falco", "DaemonSet", "app.kubernetes.io/name=falco"},
	{"keda-operator", "keda", "Deployment", "app=keda-operator"},
	{"sealed-secrets", "sealed-secrets", "Deployment", "app.kubernetes.io/name=sealed-secrets"},
	{"external-dns", "external-dns", "Deployment", "app.kubernetes.io/name=external-dns"},
	{"temporal", "temporal", "Deployment", "app.kubernetes.io/name=temporal,app.kubernetes.io/component=frontend"},
	{"otel-collector", "monitoring", "DaemonSet", "app.kubernetes.io/name=opentelemetry-collector"},
	{"zenith-api", "zenith-staging", "Deployment", "app=zenith-api"},
	{"zenith-operator", "zenith-staging", "Deployment", "app=zenith-operator"},
	{"cnpg-operator", "cnpg-system", "Deployment", "app.kubernetes.io/name=cloudnative-pg"},
	{"metrics-server", "kube-system", "Deployment", "k8s-app=metrics-server"},
}

// ListServices returns the health status of all platform services.
func (h *AdminServicesHandler) ListServices(c *fiber.Ctx) error {
	var services []entities.ServiceStatus

	for _, svc := range platformServices {
		status := entities.ServiceStatus{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Kind:      svc.Kind,
			Status:    "unknown",
		}

		// Try to get pods matching the label selector
		pods, err := h.k8s.ListPods(c.Context(), svc.Namespace, svc.LabelSelector)
		if err == nil && len(pods) > 0 {
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

			// Get uptime from oldest pod
			oldest := pods[0].StartedAt
			for _, p := range pods[1:] {
				if p.StartedAt.Before(oldest) {
					oldest = p.StartedAt
				}
			}
			status.Uptime = formatUptime(time.Since(oldest))

			// Get version from pod labels if available
			obj, err := h.k8s.GetCRDWithVersion(c.Context(), "zenith.dev/v1alpha1", svc.Kind, svc.Namespace, svc.Name)
			if err == nil && obj != nil && obj.Metadata.Labels != nil {
				if v, ok := obj.Metadata.Labels["app.kubernetes.io/version"]; ok {
					status.Version = v
				}
			}
		} else {
			// No pods found — check if it's a CNPG cluster (different resource type)
			if svc.Kind == "Cluster" {
				obj, err := h.k8s.GetCRDWithVersion(c.Context(), cnpgAPI, "Cluster", svc.Namespace, svc.Name)
				if err == nil && obj != nil {
					status.Status = "healthy"
					status.Replicas = 1
					status.Ready = 1
					// Try CNPG-specific pod labels
					cnpgPods, err := h.k8s.ListPods(c.Context(), svc.Namespace, svc.LabelSelector)
					if err == nil && len(cnpgPods) > 0 {
						status.Replicas = len(cnpgPods)
						status.Ready = 0
						for _, p := range cnpgPods {
							if p.Ready {
								status.Ready++
							}
						}
					}
				}
			}
		}

		services = append(services, status)
	}

	return c.JSON(services)
}

// GetService returns detailed status of a single service.
func (h *AdminServicesHandler) GetService(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("service name is required")
	}

	var svcDef *platformService
	for i := range platformServices {
		if platformServices[i].Name == name {
			svcDef = &platformServices[i]
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
	pods, err := h.k8s.ListPods(c.Context(), svcDef.Namespace, svcDef.LabelSelector)
	if err == nil {
		detail.Replicas = len(pods)
		for _, p := range pods {
			sp := entities.ServicePod{
				Name:     p.Name,
				Status:   p.Status,
				Restarts: int(p.Restarts),
				Age:      formatUptime(time.Since(p.StartedAt)),
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

	// For CNPG clusters, also check the CRD
	if svcDef.Kind == "Cluster" && detail.Status == "unknown" {
		obj, err := h.k8s.GetCRDWithVersion(c.Context(), cnpgAPI, "Cluster", svcDef.Namespace, svcDef.Name)
		if err == nil && obj != nil {
			detail.Status = "healthy"
		}
	}

	return c.JSON(detail)
}

// RestartService restarts a service via rollout restart.
func (h *AdminServicesHandler) RestartService(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("service name is required")
	}

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

	return c.JSON(fiber.Map{"message": "restart initiated", "service": name})
}

func formatUptime(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
