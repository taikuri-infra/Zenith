# =============================================================================
# k8s-platform module
# =============================================================================
#
# Installs the full Zenith platform stack on a k3s cluster.
# Designed for parity between staging and production (same components,
# different resource limits).
#
# Components (modular Helm charts — independently deployable):
#
#   Infrastructure (Terraform-managed):
#     1. cert-manager        — TLS certificate automation via Let's Encrypt
#     2. ClusterIssuer       — ACME HTTP-01 solver (uses Traefik)
#     3. CloudNativePG       — PostgreSQL operator (optional)
#     4. Kong                — API Gateway (DB-less, ClusterIP) (optional)
#     5. KEDA                — Event-driven autoscaling (optional)
#     6. Monitoring          — Prometheus + Grafana + Loki (optional)
#
#   Application Charts:
#     7. zenith-platform     — Shared resources (secrets, postgres CR, certs, middleware, kong routes)
#     8. zenith-api          — Go API server
#     9. zenith-landing      — Next.js landing page
#    10. zenith-demo         — MC + Web demo instances (optional)
#    11. zenith-tenant       — Per-customer MC + Web deployments (optional)
#
# =============================================================================

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.17"
    }
  }
}

# =============================================================================
# cert-manager — TLS certificate automation
# =============================================================================

resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  version          = var.cert_manager_version
  namespace        = "cert-manager"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "crds.enabled"
    value = "true"
  }

  set {
    name  = "replicaCount"
    value = "1"
  }

  set {
    name  = "resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "resources.requests.memory"
    value = "64Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "100m"
  }

  set {
    name  = "resources.limits.memory"
    value = "128Mi"
  }
}

resource "kubernetes_manifest" "cluster_issuer" {
  manifest = {
    apiVersion = "cert-manager.io/v1"
    kind       = "ClusterIssuer"
    metadata = {
      name = "letsencrypt-prod"
    }
    spec = {
      acme = {
        server = "https://acme-v02.api.letsencrypt.org/directory"
        email  = var.cert_issuer_email
        privateKeySecretRef = {
          name = "letsencrypt-prod-key"
        }
        solvers = [{
          http01 = {
            ingress = {
              ingressClassName = "traefik"
            }
          }
        }]
      }
    }
  }

  depends_on = [helm_release.cert_manager]
}

# =============================================================================
# CloudNativePG — PostgreSQL Operator
# =============================================================================

resource "helm_release" "cnpg" {
  count = var.enable_cnpg ? 1 : 0

  name             = "cnpg"
  repository       = "https://cloudnative-pg.github.io/charts"
  chart            = "cloudnative-pg"
  version          = var.cnpg_version
  namespace        = "cnpg-system"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "resources.requests.memory"
    value = "64Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "200m"
  }

  set {
    name  = "resources.limits.memory"
    value = "256Mi"
  }

  depends_on = [helm_release.cert_manager]
}

# =============================================================================
# Zenith Platform — shared infrastructure (secrets, postgres, certs, middleware)
# =============================================================================

resource "helm_release" "zenith_platform" {
  name             = "zenith-platform"
  chart            = var.chart_repository != "" ? "zenith-platform" : var.platform_chart_path
  version          = var.chart_repository != "" ? var.chart_version : null
  namespace        = var.platform_namespace
  create_namespace = true
  wait             = true
  timeout          = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.platform_values_file)]

  # App secrets
  set_sensitive {
    name  = "secrets.jwtSecret"
    value = var.jwt_secret
  }

  set_sensitive {
    name  = "secrets.adminEmail"
    value = var.admin_email
  }

  set_sensitive {
    name  = "secrets.adminPassword"
    value = var.admin_password
  }

  # Registry credentials
  set_sensitive {
    name  = "registry.host"
    value = var.registry_host
  }

  set_sensitive {
    name  = "registry.username"
    value = var.registry_username
  }

  set_sensitive {
    name  = "registry.password"
    value = var.registry_password
  }

  depends_on = [kubernetes_manifest.cluster_issuer, helm_release.cnpg]
}

# =============================================================================
# Zenith API — Go API server
# =============================================================================

resource "helm_release" "zenith_api" {
  name      = "zenith-api"
  chart     = var.chart_repository != "" ? "zenith-api" : var.api_chart_path
  version   = var.chart_repository != "" ? var.chart_version : null
  namespace = var.platform_namespace
  wait      = false
  timeout   = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.api_values_file)]

  # Registry image pull secret
  set {
    name  = "imagePullSecret"
    value = var.registry_host != "" ? "harbor-registry" : ""
  }

  set {
    name  = "imageRegistry"
    value = var.registry_host != "" ? "${var.registry_host}/zenith-stage" : ""
  }

  depends_on = [helm_release.zenith_platform]
}

# =============================================================================
# Zenith Landing — Next.js landing page
# =============================================================================

resource "helm_release" "zenith_landing" {
  name      = "zenith-landing"
  chart     = var.chart_repository != "" ? "zenith-landing" : var.landing_chart_path
  version   = var.chart_repository != "" ? var.chart_version : null
  namespace = var.platform_namespace
  wait      = false
  timeout   = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.landing_values_file)]

  set {
    name  = "imagePullSecret"
    value = var.registry_host != "" ? "harbor-registry" : ""
  }

  set {
    name  = "imageRegistry"
    value = var.registry_host != "" ? "${var.registry_host}/zenith-stage" : ""
  }

  depends_on = [helm_release.zenith_platform]
}

# =============================================================================
# Zenith Demo — MC + Web demo instances (optional)
# =============================================================================

resource "helm_release" "zenith_demo" {
  count = var.enable_demo ? 1 : 0

  name      = "zenith-demo"
  chart     = var.chart_repository != "" ? "zenith-demo" : var.demo_chart_path
  version   = var.chart_repository != "" ? var.chart_version : null
  namespace = var.platform_namespace
  wait      = false
  timeout   = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.demo_values_file)]

  set {
    name  = "imagePullSecret"
    value = var.registry_host != "" ? "harbor-registry" : ""
  }

  set {
    name  = "imageRegistry"
    value = var.registry_host != "" ? "${var.registry_host}/zenith-stage" : ""
  }

  depends_on = [helm_release.zenith_platform]
}

# =============================================================================
# Zenith Tenant — per-customer MC + Web deployments (optional)
# =============================================================================

resource "helm_release" "zenith_tenant" {
  count = var.enable_tenants ? 1 : 0

  name      = "zenith-tenant"
  chart     = var.chart_repository != "" ? "zenith-tenant" : var.tenant_chart_path
  version   = var.chart_repository != "" ? var.chart_version : null
  namespace = var.platform_namespace
  wait      = false
  timeout   = 300

  repository          = var.chart_repository != "" ? var.chart_repository : null
  repository_username = var.chart_repository != "" ? var.registry_username : null
  repository_password = var.chart_repository != "" ? var.registry_password : null

  values = [file(var.tenant_values_file)]

  # Registry credentials for tenant namespaces
  set_sensitive {
    name  = "registry.host"
    value = var.registry_host
  }

  set_sensitive {
    name  = "registry.username"
    value = var.registry_username
  }

  set_sensitive {
    name  = "registry.password"
    value = var.registry_password
  }

  set {
    name  = "imageRegistry"
    value = var.registry_host != "" ? "${var.registry_host}/zenith-stage" : ""
  }

  depends_on = [helm_release.zenith_platform]
}

# =============================================================================
# KEDA — Scale-to-zero (optional)
# =============================================================================

resource "helm_release" "keda" {
  count = var.enable_keda ? 1 : 0

  name             = "keda"
  repository       = "https://kedacore.github.io/charts"
  chart            = "keda"
  version          = var.keda_version
  namespace        = "keda"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "operator.replicaCount"
    value = "1"
  }

  set {
    name  = "operator.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "operator.resources.requests.memory"
    value = "128Mi"
  }

  set {
    name  = "metricsServer.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "metricsServer.resources.requests.memory"
    value = "64Mi"
  }
}

resource "helm_release" "keda_http_addon" {
  count = var.enable_keda ? 1 : 0

  name       = "keda-http-add-on"
  repository = "https://kedacore.github.io/charts"
  chart      = "keda-add-ons-http"
  version    = var.keda_http_addon_version
  namespace  = "keda"
  wait       = true
  timeout    = 300

  depends_on = [helm_release.keda]
}

# =============================================================================
# Kong — API Gateway
# =============================================================================

resource "helm_release" "kong" {
  count = var.enable_kong ? 1 : 0

  name             = "kong"
  repository       = "https://charts.konghq.com"
  chart            = "ingress"
  version          = var.kong_version
  namespace        = "kong"
  create_namespace = true
  wait             = true
  timeout          = 300

  # DB-less mode — declarative config via CRDs
  set {
    name  = "gateway.env.database"
    value = "off"
  }

  # ClusterIP — not exposed externally, Traefik routes to it
  set {
    name  = "gateway.proxy.type"
    value = "ClusterIP"
  }

  # Enable Kong Ingress Controller CRDs
  set {
    name  = "controller.ingressController.installCRDs"
    value = "false"
  }

  # Kong Manager — admin UI (OSS, port 8002)
  set {
    name  = "gateway.admin.http.enabled"
    value = "true"
  }

  set {
    name  = "gateway.manager.enabled"
    value = "true"
  }

  set {
    name  = "gateway.manager.type"
    value = "ClusterIP"
  }

  # Resources (staging-friendly)
  set {
    name  = "gateway.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "gateway.resources.requests.memory"
    value = "128Mi"
  }

  set {
    name  = "gateway.resources.limits.cpu"
    value = "500m"
  }

  set {
    name  = "gateway.resources.limits.memory"
    value = "512Mi"
  }

  set {
    name  = "controller.resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "controller.resources.requests.memory"
    value = "64Mi"
  }

  set {
    name  = "controller.resources.limits.cpu"
    value = "200m"
  }

  set {
    name  = "controller.resources.limits.memory"
    value = "256Mi"
  }

  depends_on = [kubernetes_manifest.cluster_issuer]
}

# =============================================================================
# Monitoring — Prometheus + Grafana + Loki (optional)
# =============================================================================

resource "helm_release" "monitoring" {
  count = var.enable_monitoring ? 1 : 0

  name             = "zenith-monitoring"
  chart            = var.monitoring_chart_path
  namespace        = "monitoring"
  create_namespace = true
  wait             = true
  timeout          = 600

  values = [file(var.monitoring_values_file)]

  set_sensitive {
    name  = "kube-prometheus-stack.grafana.adminPassword"
    value = var.admin_password
  }

  # Monitoring domain for IngressRoutes (grafana.<domain>, prometheus.<domain>)
  set {
    name  = "global.zenith.domain"
    value = var.monitoring_domain
  }

  set {
    name  = "global.zenith.platformNamespace"
    value = var.platform_namespace
  }
}
