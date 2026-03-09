# =============================================================================
# APISIX + etcd — API Gateway (replaces Kong) (5.8)
# =============================================================================

resource "helm_release" "apisix" {
  count = var.enable_apisix ? 1 : 0

  name             = "apisix"
  repository       = "https://charts.apiseven.com"
  chart            = "apisix"
  version          = var.apisix_version
  namespace        = "apisix"
  create_namespace = true
  wait             = true
  timeout          = 600

  # etcd configuration
  set {
    name  = "etcd.replicaCount"
    value = var.environment == "production" ? "3" : "1"
  }

  set {
    name  = "etcd.persistence.enabled"
    value = "true"
  }

  set {
    name  = "etcd.persistence.storageClass"
    value = "hcloud-volumes"
  }

  set {
    name  = "etcd.persistence.size"
    value = "5Gi"
  }

  # Gateway
  set {
    name  = "gateway.type"
    value = "ClusterIP"
  }

  set {
    name  = "replicaCount"
    value = var.environment == "production" ? "2" : "1"
  }

  # Allow ingress controller pod network to reach admin API
  set {
    name  = "admin.allow.list[0]"
    value = "127.0.0.1/24"
  }

  set {
    name  = "admin.allow.list[1]"
    value = "10.42.0.0/16"
  }

  # Plugins (must be a YAML array, not object)
  values = [yamlencode({
    plugins = ["jwt-auth", "cors", "limit-count", "openid-connect", "opentelemetry", "prometheus"]
  })]

  # Resources
  set {
    name  = "resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "500m"
  }

  set {
    name  = "resources.limits.memory"
    value = "512Mi"
  }

  set {
    name  = "priorityClassName"
    value = "infra-critical"
  }

  depends_on = [
    kubernetes_manifest.cluster_issuer,
    kubernetes_priority_class.infra_critical,
  ]
}

resource "helm_release" "apisix_ingress" {
  count = var.enable_apisix ? 1 : 0

  name       = "apisix-ingress-controller"
  repository = "https://charts.apiseven.com"
  chart      = "apisix-ingress-controller"
  version    = var.apisix_ingress_version
  namespace  = "apisix"
  wait       = true
  timeout    = 300

  set {
    name  = "config.apisix.adminAPIVersion"
    value = "v3"
  }

  set {
    name  = "config.apisix.serviceNamespace"
    value = "apisix"
  }

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }

  depends_on = [helm_release.apisix]
}

# =============================================================================
# external-dns — Automatic Cloudflare DNS (5.9)
# =============================================================================

resource "helm_release" "external_dns" {
  count = var.enable_external_dns ? 1 : 0

  name             = "external-dns"
  repository       = "oci://registry-1.docker.io/bitnamicharts"
  chart            = "external-dns"
  version          = var.external_dns_version
  namespace        = "external-dns"
  create_namespace = true
  wait             = true
  timeout          = 600

  # Bitnami purged docker.io/bitnami — use legacy registry
  set {
    name  = "image.registry"
    value = "docker.io"
  }

  set {
    name  = "image.repository"
    value = "bitnamilegacy/external-dns"
  }

  set {
    name  = "provider"
    value = "cloudflare"
  }

  set_sensitive {
    name  = "cloudflare.apiToken"
    value = var.cloudflare_api_token
  }

  set {
    name  = "domainFilters[0]"
    value = var.domain
  }

  # Sources: service + ingress (default) + traefik-proxy (IngressRoute CRDs)
  set {
    name  = "sources[0]"
    value = "service"
  }

  set {
    name  = "sources[1]"
    value = "ingress"
  }

  set {
    name  = "sources[2]"
    value = "traefik-proxy"
  }

  set {
    name  = "policy"
    value = "sync"
  }

  set {
    name  = "txtOwnerId"
    value = "zenith-${var.environment}"
  }

  # Traefik 3.x only ships traefik.io CRDs — disable legacy containo.us watch
  set {
    name  = "extraArgs.traefik-disable-legacy"
    value = ""
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

  depends_on = [kubernetes_manifest.cluster_issuer]
}
