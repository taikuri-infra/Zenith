# =============================================================================
# KEDA — Scale-to-zero (existing, kept)
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
