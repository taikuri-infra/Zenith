# =============================================================================
# Keycloak — Identity Provider (5.7)
# =============================================================================

resource "helm_release" "keycloak" {
  count = var.enable_keycloak ? 1 : 0

  name             = "keycloak"
  repository       = "oci://registry-1.docker.io/bitnamicharts"
  chart            = "keycloak"
  version          = var.keycloak_version
  namespace        = "keycloak"
  create_namespace = false
  wait             = true
  timeout          = 600

  # Bitnami purged docker.io/bitnami — use legacy registry
  set {
    name  = "image.registry"
    value = "docker.io"
  }

  set {
    name  = "image.repository"
    value = "bitnamilegacy/keycloak"
  }

  # Use external CNPG database
  set {
    name  = "postgresql.enabled"
    value = "false"
  }

  set {
    name  = "externalDatabase.host"
    value = "${kubernetes_manifest.cnpg_keycloak[0].manifest.metadata.name}-rw.${kubernetes_manifest.cnpg_keycloak[0].manifest.metadata.namespace}.svc.cluster.local"
  }

  set {
    name  = "externalDatabase.port"
    value = "5432"
  }

  set {
    name  = "externalDatabase.database"
    value = "keycloak"
  }

  set {
    name  = "externalDatabase.user"
    value = "keycloak"
  }

  set_sensitive {
    name  = "externalDatabase.password"
    value = var.keycloak_db_password
  }

  set_sensitive {
    name  = "auth.adminUser"
    value = "admin"
  }

  set_sensitive {
    name  = "auth.adminPassword"
    value = var.keycloak_admin_password
  }

  set {
    name  = "resources.requests.cpu"
    value = "200m"
  }

  set {
    name  = "resources.requests.memory"
    value = "512Mi"
  }

  set {
    name  = "resources.limits.cpu"
    value = "1"
  }

  set {
    name  = "resources.limits.memory"
    value = "1Gi"
  }

  set {
    name  = "httpRelativePath"
    value = "/"
  }

  set {
    name  = "proxyHeaders"
    value = "xforwarded"
  }

  set {
    name  = "priorityClassName"
    value = "infra-critical"
  }

  depends_on = [
    kubernetes_manifest.cnpg_keycloak,
    kubernetes_priority_class.infra_critical,
  ]
}
