# =============================================================================
# Harbor — Container & Helm Chart Registry (5.11)
# =============================================================================

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
