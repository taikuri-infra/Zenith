package deploy

import (
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// K8sResources holds the generated Kubernetes manifests for an app deployment.
type K8sResources struct {
	Deployment       map[string]interface{}
	Service          map[string]interface{}
	IngressRoute     map[string]interface{}
	HTTPScaledObject map[string]interface{} // nil when always-on (paid tiers)
}

// GenerateK8sResources creates the Kubernetes manifests needed to deploy an app.
// When planLimits is non-nil and the plan uses scale-to-zero, the deployment
// starts at 0 replicas and an HTTPScaledObject CRD is included for KEDA.
func GenerateK8sResources(app *entities.App, imageTag, baseDomain string, envVars []entities.EnvVar, planLimits *entities.PlanLimits) *K8sResources {
	namespace := "zenith-apps"
	labels := map[string]string{
		"app":                   app.Subdomain,
		"zenith.dev/app-id":     app.ID,
		"zenith.dev/managed-by": "zenith",
	}

	scaleToZero := planLimits != nil && ShouldScaleToZero(planLimits)

	res := &K8sResources{
		Deployment: generateDeployment(app, imageTag, namespace, labels, envVars),
		Service:    generateService(app, namespace, labels),
	}

	if scaleToZero {
		// Set deployment replicas to 0 — KEDA manages scaling
		spec := res.Deployment["spec"].(map[string]interface{})
		spec["replicas"] = int32(0)

		res.HTTPScaledObject = GenerateHTTPScaledObject(app, baseDomain, planLimits.SleepAfterMins)
		res.IngressRoute = generateIngressRouteWithColdStart(app, namespace, labels, baseDomain)
	} else {
		res.IngressRoute = generateIngressRoute(app, namespace, labels, baseDomain)
	}

	return res
}

func generateDeployment(app *entities.App, imageTag, namespace string, labels map[string]string, envVars []entities.EnvVar) map[string]interface{} {
	replicas := int32(1)
	port := app.Port
	if port == 0 {
		port = 8080
	}

	// Convert env vars to K8s env spec
	k8sEnv := make([]map[string]interface{}, 0, len(envVars)+1)
	k8sEnv = append(k8sEnv, map[string]interface{}{
		"name":  "PORT",
		"value": fmt.Sprintf("%d", port),
	})
	for _, ev := range envVars {
		k8sEnv = append(k8sEnv, map[string]interface{}{
			"name":  ev.Key,
			"value": ev.Value,
		})
	}

	return map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      app.Subdomain,
			"namespace": namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"replicas": replicas,
			"selector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app": app.Subdomain,
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": labels,
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "app",
							"image": imageTag,
							"ports": []map[string]interface{}{
								{
									"containerPort": port,
									"protocol":      "TCP",
								},
							},
							"env": k8sEnv,
							"resources": map[string]interface{}{
								"limits": map[string]string{
									"cpu":    "500m",
									"memory": "512Mi",
								},
								"requests": map[string]string{
									"cpu":    "100m",
									"memory": "128Mi",
								},
							},
							"readinessProbe": map[string]interface{}{
								"httpGet": map[string]interface{}{
									"path": "/",
									"port": port,
								},
								"initialDelaySeconds": 5,
								"periodSeconds":       10,
							},
							"livenessProbe": map[string]interface{}{
								"httpGet": map[string]interface{}{
									"path": "/",
									"port": port,
								},
								"initialDelaySeconds": 15,
								"periodSeconds":       20,
							},
						},
					},
				},
			},
		},
	}
}

func generateService(app *entities.App, namespace string, labels map[string]string) map[string]interface{} {
	port := app.Port
	if port == 0 {
		port = 8080
	}

	return map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      app.Subdomain,
			"namespace": namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"selector": map[string]string{
				"app": app.Subdomain,
			},
			"ports": []map[string]interface{}{
				{
					"port":       80,
					"targetPort": port,
					"protocol":   "TCP",
				},
			},
		},
	}
}

func generateIngressRoute(app *entities.App, namespace string, labels map[string]string, baseDomain string) map[string]interface{} {
	host := app.Subdomain + "." + baseDomain

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
					"match": fmt.Sprintf("Host(`%s`)", host),
					"kind":  "Rule",
					"services": []map[string]interface{}{
						{
							"name": app.Subdomain,
							"port": 80,
						},
					},
				},
			},
			"tls": map[string]interface{}{
				"certResolver": "letsencrypt",
			},
		},
	}
}
