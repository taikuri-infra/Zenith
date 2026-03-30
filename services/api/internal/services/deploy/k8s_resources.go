package deploy

import (
	"fmt"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// K8sResources holds the generated Kubernetes manifests for an app deployment.
type K8sResources struct {
	Deployment       map[string]interface{}
	Service          map[string]interface{}
	IngressRoute     map[string]interface{}
	HTTPScaledObject map[string]interface{} // nil when always-on (paid tiers)
	NetworkPolicy    map[string]interface{} // per-app tenant isolation
	Certificate      map[string]interface{} // nil when no custom domains
}

// StagingPerAppResources returns minimal CPU/RAM limits for staging environment apps.
func StagingPerAppResources() (cpuLimit, memLimit, cpuReq, memReq string) {
	return "250m", "256Mi", "50m", "64Mi"
}

// AppHostname returns the full hostname for an app based on its environment.
// Production: subdomain.apps.{baseDomain}
// Staging:    subdomain.dev.apps.{baseDomain}
func AppHostname(subdomain, baseDomain string, isStaging bool) string {
	if isStaging {
		return subdomain + ".dev." + baseDomain
	}
	return subdomain + "." + baseDomain
}

// PerAppResources returns CPU/RAM limits and requests per tier for a single app container.
func PerAppResources(tier entities.PlanTier) (cpuLimit, memLimit, cpuReq, memReq string) {
	switch tier {
	case entities.PlanPro:
		return "500m", "512Mi", "100m", "128Mi"
	case entities.PlanTeam:
		return "1000m", "1Gi", "200m", "256Mi"
	case entities.PlanBusiness:
		return "2000m", "2Gi", "500m", "512Mi"
	case entities.PlanEnterprise:
		return "4000m", "4Gi", "1000m", "1Gi"
	default: // Free — low resources, always-on
		return "100m", "128Mi", "50m", "64Mi"
	}
}

// GenerateK8sResources creates the Kubernetes manifests needed to deploy an app.
// When planLimits is non-nil and the plan uses scale-to-zero, the deployment
// starts at 0 replicas and an HTTPScaledObject CRD is included for KEDA.
// customDomains is a list of verified custom domain hostnames to add to the IngressRoute.
func GenerateK8sResources(app *entities.App, imageTag, baseDomain string, envVars []entities.EnvVar, planLimits *entities.PlanLimits, tier entities.PlanTier, customDomains []string, isStaging ...bool) *K8sResources {
	namespace := "zenith-apps"
	labels := map[string]string{
		"app":                   app.Subdomain,
		"zenith.dev/app-id":     app.ID,
		"zenith.dev/managed-by": "zenith",
	}

	staging := len(isStaging) > 0 && isStaging[0]
	scaleToZero := planLimits != nil && ShouldScaleToZero(planLimits)

	// Staging environments use minimal resources regardless of plan tier
	effectiveTier := tier
	if staging {
		effectiveTier = entities.PlanFree // minimal resources for staging
	}

	// APISIX is used for the platform API gateway (api.stage.freezenith.com),
	// NOT for routing customer app traffic. Customer apps route directly to
	// their own service. The auto-gateway creates APISIX routes on *.gw.* domains.
	useApisix := false

	// Build the effective hostname for this environment
	effectiveDomain := baseDomain
	if staging {
		// Staging apps get dev.apps.{baseDomain} subdomain
		parts := strings.SplitN(baseDomain, ".", 2)
		if len(parts) == 2 {
			effectiveDomain = "dev." + baseDomain
		}
	}

	res := &K8sResources{
		Deployment:    generateDeployment(app, imageTag, namespace, labels, envVars, effectiveTier),
		Service:       generateService(app, namespace, labels),
		NetworkPolicy: generateNetworkPolicy(app, namespace, labels, useApisix),
	}

	if scaleToZero && !staging {
		// Set deployment replicas to 0 — KEDA manages scaling
		spec := res.Deployment["spec"].(map[string]interface{})
		spec["replicas"] = int32(0)

		res.HTTPScaledObject = GenerateHTTPScaledObject(app, effectiveDomain, planLimits.SleepAfterMins)
		if useApisix {
			res.IngressRoute = generateIngressRouteViaApisixWithColdStart(app, namespace, labels, effectiveDomain, customDomains)
		} else {
			res.IngressRoute = generateIngressRouteWithColdStart(app, namespace, labels, effectiveDomain, customDomains)
		}
	} else {
		if useApisix {
			res.IngressRoute = generateIngressRouteViaApisix(app, namespace, labels, effectiveDomain, customDomains)
		} else {
			res.IngressRoute = generateIngressRoute(app, namespace, labels, effectiveDomain, customDomains, effectiveTier, staging)
		}
	}

	// Generate Certificate CRD when custom domains are present
	if len(customDomains) > 0 {
		res.Certificate = generateCertificate(app, namespace, labels, baseDomain, customDomains)
	}

	return res
}

func generateDeployment(app *entities.App, imageTag, namespace string, labels map[string]string, envVars []entities.EnvVar, tier entities.PlanTier) map[string]interface{} {
	replicas := int32(1)
	if app.Replicas > 1 {
		replicas = int32(app.Replicas)
	}
	port := app.Port
	if port == 0 {
		port = 8080
	}

	// Build init containers for depends_on: wait until each dependency is reachable
	// via TCP before starting the main container. Uses busybox nc (netcat).
	var initContainers []map[string]interface{}
	for _, dep := range app.DependsOn {
		depName := dep
		initContainers = append(initContainers, map[string]interface{}{
			"name":  "wait-for-" + depName,
			"image": "busybox:1.36",
			"command": []string{
				"sh", "-c",
				// All K8s Services expose port 80 (see generateService).
				// nc -z polls TCP connectivity without sending data.
				// 300s timeout prevents infinite hang if dependency never starts.
				fmt.Sprintf(
					"timeout=300; elapsed=0; until nc -z %s.%s.svc 80 2>/dev/null; do if [ $elapsed -ge $timeout ]; then echo '%s not ready after ${timeout}s, giving up'; exit 1; fi; echo 'waiting for %s... (${elapsed}s/${timeout}s)'; sleep 2; elapsed=$((elapsed+2)); done; echo '%s is ready'",
					depName, namespace, depName, depName, depName,
				),
			},
			"securityContext": map[string]interface{}{
				"allowPrivilegeEscalation": false,
				"capabilities":             map[string]interface{}{"drop": []string{"ALL"}},
			},
		})
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

	cpuLimit, memLimit, cpuReq, memReq := PerAppResources(tier)

	container := map[string]interface{}{
		"name":  "app",
		"image": imageTag,
		"env":   k8sEnv,
		"securityContext": map[string]interface{}{
			"allowPrivilegeEscalation": false,
			"capabilities": map[string]interface{}{
				"drop": []string{"ALL"},
				"add":  []string{"CHOWN", "SETUID", "SETGID", "NET_BIND_SERVICE"},
			},
		},
		"resources": map[string]interface{}{
			"limits": map[string]string{
				"cpu":    cpuLimit,
				"memory": memLimit,
			},
			"requests": map[string]string{
				"cpu":    cpuReq,
				"memory": memReq,
			},
		},
	}

	// Propagate compose `command:` override → K8s container args.
	// In compose, `command:` overrides the image CMD (not ENTRYPOINT).
	// In K8s, `args` maps to CMD. Split on spaces for simple cases.
	if app.Command != "" {
		args := strings.Fields(app.Command)
		if len(args) > 0 {
			container["args"] = args
		}
	}

	// Conditional probes and ports by app type
	switch app.AppType {
	case entities.AppTypeWorker:
		// Workers don't serve HTTP — use TCP socket probe on their port if set,
		// otherwise a permissive exec probe. This is more meaningful than `/bin/sh -c true`.
		if port > 0 {
			container["ports"] = []map[string]interface{}{
				{"containerPort": port, "protocol": "TCP"},
			}
			container["readinessProbe"] = map[string]interface{}{
				"tcpSocket":           map[string]interface{}{"port": port},
				"initialDelaySeconds": 5,
				"periodSeconds":       15,
				"failureThreshold":    5,
			}
			container["livenessProbe"] = map[string]interface{}{
				"tcpSocket":           map[string]interface{}{"port": port},
				"initialDelaySeconds": 20,
				"periodSeconds":       30,
				"failureThreshold":    3,
			}
		} else {
			container["readinessProbe"] = map[string]interface{}{
				"exec":                map[string]interface{}{"command": []string{"/bin/sh", "-c", "true"}},
				"initialDelaySeconds": 5,
				"periodSeconds":       30,
			}
			container["livenessProbe"] = map[string]interface{}{
				"exec":                map[string]interface{}{"command": []string{"/bin/sh", "-c", "true"}},
				"initialDelaySeconds": 15,
				"periodSeconds":       60,
			}
		}
	case entities.AppTypeCron:
		// Cron jobs run and exit — no probes needed
	default:
		// Web apps: HTTP probes on the app port.
		// Use a generous failureThreshold so apps that are slow to start (e.g. JVM, Next.js)
		// don't get killed before they're ready. 10 failures × 10s = 100s of grace.
		healthPath := app.HealthCheckPath
		if healthPath == "" {
			healthPath = "/"
		}
		container["ports"] = []map[string]interface{}{
			{
				"containerPort": port,
				"protocol":      "TCP",
			},
		}
		container["readinessProbe"] = map[string]interface{}{
			"httpGet": map[string]interface{}{
				"path": healthPath,
				"port": port,
			},
			"initialDelaySeconds": 5,
			"periodSeconds":       10,
			// 10 failures × 10s = 100s grace period for slow-starting apps (JVM, Next.js, etc.)
			"failureThreshold": 10,
		}
		container["livenessProbe"] = map[string]interface{}{
			"httpGet": map[string]interface{}{
				"path": healthPath,
				"port": port,
			},
			"initialDelaySeconds": 15,
			"periodSeconds":       20,
			"failureThreshold":    5,
		}
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
				"spec": func() map[string]interface{} {
					pullSecrets := []map[string]interface{}{
						{"name": "app-registry-auth"},
					}
					// Per-app custom registry secret (created by deployer when RegistryUser is set)
					if app.RegistryUser != "" {
						pullSecrets = append(pullSecrets, map[string]interface{}{
							"name": "regcred-" + app.Subdomain,
						})
					}
					podSpec := map[string]interface{}{
						"imagePullSecrets": pullSecrets,
						"containers": []map[string]interface{}{
							container,
						},
					}
					if len(initContainers) > 0 {
						podSpec["initContainers"] = initContainers
					}
					return podSpec
				}(),
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

func generateIngressRoute(app *entities.App, namespace string, labels map[string]string, baseDomain string, customDomains []string, tier entities.PlanTier, staging ...bool) map[string]interface{} {
	matchRule := buildHostMatchRule(app.Subdomain+"."+baseDomain, customDomains)

	isStaging := len(staging) > 0 && staging[0]
	tls := map[string]interface{}{}
	if len(customDomains) > 0 {
		tls["secretName"] = app.Subdomain + "-custom-tls"
	} else if isStaging {
		// Staging apps use a separate wildcard cert for *.dev.apps.{baseDomain}
		tls["secretName"] = "dev-apps-wildcard-tls"
	}

	route := map[string]interface{}{
		"match": matchRule,
		"kind":  "Rule",
		"services": []map[string]interface{}{
			{
				"name": app.Subdomain,
				"port": 80,
			},
		},
	}

	// Free-tier apps get the "Powered by Zenith" banner injection middleware.
	// The Middleware CRD is deployed as a static resource via Helm.
	if tier == entities.PlanFree {
		route["middlewares"] = []map[string]interface{}{
			{
				"name":      "powered-by-zenith",
				"namespace": namespace,
			},
		}
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
			"routes":      []map[string]interface{}{route},
			"tls":         tls,
		},
	}
}

// generateIngressRouteViaApisix creates a Traefik IngressRoute that routes through the
// APISIX gateway bridge (ExternalName svc) instead of directly to the app service.
// Used for apps with auto-gateway (exposure=public or exposure=protected).
func generateIngressRouteViaApisix(app *entities.App, namespace string, labels map[string]string, baseDomain string, customDomains []string) map[string]interface{} {
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
					"services": []map[string]interface{}{
						{
							"name": "apisix-gateway-bridge",
							"port": 80,
						},
					},
				},
			},
			"tls": tls,
		},
	}
}

// generateIngressRouteViaApisixWithColdStart creates an IngressRoute that routes through APISIX
// with the cold-start error page middleware (for free-tier apps that scale to zero).
func generateIngressRouteViaApisixWithColdStart(app *entities.App, namespace string, labels map[string]string, baseDomain string, customDomains []string) map[string]interface{} {
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
							"name": "apisix-gateway-bridge",
							"port": 80,
						},
					},
				},
			},
			"tls": tls,
		},
	}
}

// buildHostMatchRule creates a Traefik Host() match rule with one or more hosts.
func buildHostMatchRule(primaryHost string, customDomains []string) string {
	hosts := []string{fmt.Sprintf("Host(`%s`)", primaryHost)}
	for _, d := range customDomains {
		hosts = append(hosts, fmt.Sprintf("Host(`%s`)", d))
	}
	return strings.Join(hosts, " || ")
}

// generateCertificate creates a cert-manager Certificate CRD for custom domains.
func generateCertificate(app *entities.App, namespace string, labels map[string]string, baseDomain string, customDomains []string) map[string]interface{} {
	dnsNames := make([]string, 0, len(customDomains)+1)
	dnsNames = append(dnsNames, app.Subdomain+"."+baseDomain)
	dnsNames = append(dnsNames, customDomains...)

	return map[string]interface{}{
		"apiVersion": "cert-manager.io/v1",
		"kind":       "Certificate",
		"metadata": map[string]interface{}{
			"name":      app.Subdomain + "-custom-tls",
			"namespace": namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"secretName": app.Subdomain + "-custom-tls",
			"issuerRef": map[string]interface{}{
				"name": "letsencrypt-prod",
				"kind": "ClusterIssuer",
			},
			"dnsNames": dnsNames,
		},
	}
}

// generateNetworkPolicy creates a NetworkPolicy that isolates user app pods:
// - Ingress: only from Traefik (kube-system namespace) and optionally APISIX (apisix namespace)
// - Egress: DNS (kube-dns) + internet (blocks 10.0.0.0/8, 172.16.0.0/12 to prevent pod-to-pod)
func generateNetworkPolicy(app *entities.App, namespace string, labels map[string]string, allowApisix bool) map[string]interface{} {
	ingressFrom := []map[string]interface{}{
		{
			"namespaceSelector": map[string]interface{}{
				"matchLabels": map[string]string{
					"kubernetes.io/metadata.name": "kube-system",
				},
			},
			"podSelector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app.kubernetes.io/name": "traefik",
				},
			},
		},
	}

	if allowApisix {
		ingressFrom = append(ingressFrom, map[string]interface{}{
			"namespaceSelector": map[string]interface{}{
				"matchLabels": map[string]string{
					"kubernetes.io/metadata.name": "apisix",
				},
			},
			"podSelector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app.kubernetes.io/name": "apisix",
				},
			},
		})
	}

	return map[string]interface{}{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "NetworkPolicy",
		"metadata": map[string]interface{}{
			"name":      app.Subdomain + "-netpol",
			"namespace": namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"podSelector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app": app.Subdomain,
				},
			},
			"policyTypes": []string{"Ingress", "Egress"},
			"ingress": []map[string]interface{}{
				{
					"from": ingressFrom,
				},
			},
			"egress": []map[string]interface{}{
				{
					// DNS
					"to": []map[string]interface{}{
						{
							"namespaceSelector": map[string]interface{}{
								"matchLabels": map[string]string{
									"kubernetes.io/metadata.name": "kube-system",
								},
							},
						},
					},
					"ports": []map[string]interface{}{
						{"protocol": "UDP", "port": 53},
						{"protocol": "TCP", "port": 53},
					},
				},
				{
					// Internet (block private ranges to prevent pod-to-pod and internal svc access)
					"to": []map[string]interface{}{
						{
							"ipBlock": map[string]interface{}{
								"cidr": "0.0.0.0/0",
								"except": []string{
									"10.0.0.0/8",
									"172.16.0.0/12",
									"192.168.0.0/16",
								},
							},
						},
					},
				},
			},
		},
	}
}
