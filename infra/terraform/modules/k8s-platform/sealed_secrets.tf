# =============================================================================
# Sealed Secrets — Encrypted secrets for GitOps (5.3)
# =============================================================================

resource "helm_release" "sealed_secrets" {
  count = var.enable_sealed_secrets ? 1 : 0

  name             = "sealed-secrets"
  repository       = "https://bitnami-labs.github.io/sealed-secrets"
  chart            = "sealed-secrets"
  version          = var.sealed_secrets_version
  namespace        = "sealed-secrets"
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
    value = "100m"
  }

  set {
    name  = "resources.limits.memory"
    value = "128Mi"
  }

  depends_on = [helm_release.cert_manager]
}
