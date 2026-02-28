# =============================================================================
# Harbor — Pro-tier Customer Registry (5.11)
# =============================================================================
# This Harbor instance is for pro/paid customers only.
# Free-tier users do NOT get a registry.
# Platform images & Helm charts are stored in the separate internal Harbor
# (registry.stage.freezenith.com), managed outside this cluster.

resource "helm_release" "harbor" {
  count = var.enable_harbor ? 1 : 0

  name             = "harbor"
  repository       = "https://helm.goharbor.io"
  chart            = "harbor"
  version          = var.harbor_version
  namespace        = "harbor"
  create_namespace = true
  wait             = true
  timeout          = 600

  set {
    name  = "expose.type"
    value = "clusterIP"
  }

  set {
    name  = "expose.tls.enabled"
    value = "false"
  }

  set {
    name  = "externalURL"
    value = "https://${var.customer_registry_host}"
  }

  # S3 storage backend
  set {
    name  = "persistence.imageChartStorage.type"
    value = "s3"
  }

  set {
    name  = "persistence.imageChartStorage.s3.region"
    value = "fsn1"
  }

  set {
    name  = "persistence.imageChartStorage.s3.bucket"
    value = "zenith-harbor"
  }

  set_sensitive {
    name  = "persistence.imageChartStorage.s3.accesskey"
    value = var.s3_access_key
  }

  set_sensitive {
    name  = "persistence.imageChartStorage.s3.secretkey"
    value = var.s3_secret_key
  }

  set {
    name  = "persistence.imageChartStorage.s3.regionendpoint"
    value = var.s3_endpoint
  }

  set {
    name  = "trivy.enabled"
    value = "true"
  }

  set_sensitive {
    name  = "harborAdminPassword"
    value = var.admin_password
  }

  set {
    name  = "core.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "core.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "priorityClassName"
    value = "platform"
  }

  depends_on = [
    kubernetes_manifest.cluster_issuer,
    kubernetes_priority_class.platform,
  ]
}

# =============================================================================
# Harbor TLS — Certificate + IngressRoute for customer registry (5.11b)
# =============================================================================

resource "kubernetes_manifest" "harbor_certificate" {
  count = var.enable_harbor ? 1 : 0

  manifest = {
    apiVersion = "cert-manager.io/v1"
    kind       = "Certificate"
    metadata = {
      name      = "harbor-tls"
      namespace = "harbor"
    }
    spec = {
      secretName = "harbor-tls"
      issuerRef = {
        name = "letsencrypt-prod"
        kind = "ClusterIssuer"
      }
      dnsNames = [
        var.customer_registry_host,
      ]
    }
  }

  depends_on = [helm_release.harbor]
}

resource "kubernetes_manifest" "harbor_ingressroute" {
  count = var.enable_harbor ? 1 : 0

  manifest = {
    apiVersion = "traefik.io/v1alpha1"
    kind       = "IngressRoute"
    metadata = {
      name      = "harbor"
      namespace = "harbor"
    }
    spec = {
      entryPoints = ["websecure"]
      routes = [{
        match = "Host(`${var.customer_registry_host}`)"
        kind  = "Rule"
        services = [{
          name = "harbor-core"
          port = 80
        }]
      }]
      tls = {
        secretName = "harbor-tls"
      }
    }
  }

  depends_on = [helm_release.harbor]
}
