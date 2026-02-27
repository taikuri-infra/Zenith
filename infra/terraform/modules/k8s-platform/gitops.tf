# =============================================================================
# ArgoCD — GitOps Engine (5.10)
# =============================================================================

resource "helm_release" "argocd" {
  count = var.enable_argocd ? 1 : 0

  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = var.argocd_version
  namespace        = "argocd"
  create_namespace = true
  wait             = true
  timeout          = 600

  set {
    name  = "server.replicas"
    value = "1"
  }

  set {
    name  = "server.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "server.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "repoServer.replicas"
    value = "1"
  }

  set {
    name  = "repoServer.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "repoServer.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "controller.resources.requests.cpu"
    value = "200m"
  }

  set {
    name  = "controller.resources.requests.memory"
    value = "512Mi"
  }

  set {
    name  = "server.extensions.enabled"
    value = "true"
  }

  set {
    name  = "configs.params.server\\.insecure"
    value = "true"
  }

  depends_on = [kubernetes_manifest.cluster_issuer]
}

resource "helm_release" "argocd_image_updater" {
  count = var.enable_argocd ? 1 : 0

  name       = "argocd-image-updater"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argocd-image-updater"
  version    = var.argocd_image_updater_version
  namespace  = "argocd"
  wait       = true
  timeout    = 300

  set {
    name  = "config.registries[0].name"
    value = "harbor"
  }

  set {
    name  = "config.registries[0].prefix"
    value = var.customer_registry_host
  }

  set {
    name  = "config.registries[0].api_url"
    value = "https://${var.customer_registry_host}"
  }

  set {
    name  = "config.registries[0].default"
    value = "true"
  }

  depends_on = [helm_release.argocd]
}
