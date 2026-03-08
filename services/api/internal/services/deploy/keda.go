package deploy

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// ShouldScaleToZero returns true when the user's plan uses sleep mode (scale-to-zero).
func ShouldScaleToZero(limits *entities.PlanLimits) bool {
	return !limits.AlwaysOn
}

// GenerateHTTPScaledObject produces the KEDA HTTPScaledObject CRD manifest
// that makes Traefik → KEDA interceptor → Pod routing work with scale-to-zero.
func GenerateHTTPScaledObject(app *entities.App, baseDomain string, sleepAfterMins int) map[string]interface{} {
	host := app.Subdomain + "." + baseDomain
	namespace := "zenith-apps"

	scaledownPeriod := sleepAfterMins * 60
	if scaledownPeriod <= 0 {
		scaledownPeriod = 900 // default 15 min
	}

	return map[string]interface{}{
		"apiVersion": "http.keda.sh/v1alpha1",
		"kind":       "HTTPScaledObject",
		"metadata": map[string]interface{}{
			"name":      app.Subdomain,
			"namespace": namespace,
			"labels": map[string]string{
				"app":                   app.Subdomain,
				"zenith.dev/app-id":     app.ID,
				"zenith.dev/managed-by": "zenith",
			},
		},
		"spec": map[string]interface{}{
			"hosts": []string{host},
			"scaleTargetRef": map[string]interface{}{
				"name":       app.Subdomain,
				"kind":       "Deployment",
				"apiVersion": "apps/v1",
				"service":    app.Subdomain,
				"port":       80,
			},
			"replicas": map[string]interface{}{
				"min": 0,
				"max": 1,
			},
			"scaledownPeriod":      scaledownPeriod,
			"targetPendingRequests": 1,
		},
	}
}

// generateColdStartMiddleware creates a Traefik errors middleware that serves
// the cold-start splash page when the KEDA interceptor returns 502/503.
func generateColdStartMiddleware(namespace string) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "traefik.io/v1alpha1",
		"kind":       "Middleware",
		"metadata": map[string]interface{}{
			"name":      "cold-start-errors",
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"errors": map[string]interface{}{
				"status": []string{"502-503"},
				"service": map[string]interface{}{
					"name": "cold-start-page",
					"port": 80,
				},
				"query": "/",
			},
		},
	}
}

// generateIngressRouteWithColdStart generates an IngressRoute with the cold-start
// error page middleware attached (for free-tier apps that scale to zero).
func generateIngressRouteWithColdStart(app *entities.App, namespace string, labels map[string]string, baseDomain string, customDomains []string) map[string]interface{} {
	matchRule := buildHostMatchRule(app.Subdomain+"."+baseDomain, customDomains)

	tls := map[string]interface{}{}
	if len(customDomains) > 0 {
		tls["secretName"] = app.Subdomain + "-custom-tls"
	}

	return map[string]interface{}{
		"apiVersion": "traefik.io/v1alpha1",
		"kind":       "IngressRoute",
		"metadata": map[string]interface{}{
			"name":      app.Subdomain,
			"namespace": namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"entryPoints": []string{"websecure"},
			"routes": []map[string]interface{}{
				{
					"match": matchRule,
					"kind":  "Rule",
					"middlewares": []map[string]interface{}{
						{
							"name":      "cold-start-errors",
							"namespace": namespace,
						},
					},
					"services": []map[string]interface{}{
						{
							"name": app.Subdomain,
							"port": 80,
						},
					},
				},
			},
			"tls": tls,
		},
	}
}
