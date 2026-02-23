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
# Zenith Platform — all app resources via Helm chart
# =============================================================================

resource "helm_release" "zenith" {
  name      = "zenith"
  chart     = var.zenith_chart_path
  namespace = var.platform_namespace
  wait      = true
  timeout   = 300

  # Base values file
  values = [file(var.zenith_values_file)]

  # Secrets (passed via Terraform, never in values files)
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

  set_sensitive {
    name  = "secrets.dbPassword"
    value = var.db_password
  }

  depends_on = [kubernetes_manifest.cluster_issuer]
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
  repository = "https://kedacore.github.io/http-add-on"
  chart      = "keda-add-ons-http"
  version    = var.keda_http_addon_version
  namespace  = "keda"
  wait       = true
  timeout    = 300

  depends_on = [helm_release.keda]
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
}
