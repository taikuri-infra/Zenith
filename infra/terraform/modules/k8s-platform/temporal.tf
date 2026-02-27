# =============================================================================
# Temporal — Provisioning Workflow Engine (5.12)
# =============================================================================

resource "helm_release" "temporal" {
  count = var.enable_temporal ? 1 : 0

  name             = "temporal"
  repository       = "https://go.temporal.io/helm-charts"
  chart            = "temporal"
  version          = var.temporal_version
  namespace        = "temporal"
  create_namespace = true
  wait             = true
  timeout          = 600

  # External PostgreSQL (CNPG free cluster)
  set {
    name  = "server.config.persistence.default.sql.driver"
    value = "postgres12"
  }

  set {
    name  = "server.config.persistence.default.sql.host"
    value = "free-pg-rw.zenith-shared.svc.cluster.local"
  }

  set {
    name  = "server.config.persistence.default.sql.port"
    value = "5432"
  }

  set {
    name  = "server.config.persistence.default.sql.database"
    value = "temporal"
  }

  set_sensitive {
    name  = "server.config.persistence.default.sql.user"
    value = var.temporal_db_user
  }

  set_sensitive {
    name  = "server.config.persistence.default.sql.password"
    value = var.temporal_db_password
  }

  # Visibility store
  set {
    name  = "server.config.persistence.visibility.sql.driver"
    value = "postgres12"
  }

  set {
    name  = "server.config.persistence.visibility.sql.host"
    value = "free-pg-rw.zenith-shared.svc.cluster.local"
  }

  set {
    name  = "server.config.persistence.visibility.sql.port"
    value = "5432"
  }

  set {
    name  = "server.config.persistence.visibility.sql.database"
    value = "temporal_visibility"
  }

  set_sensitive {
    name  = "server.config.persistence.visibility.sql.user"
    value = var.temporal_db_user
  }

  set_sensitive {
    name  = "server.config.persistence.visibility.sql.password"
    value = var.temporal_db_password
  }

  # Disable built-in dependencies
  set {
    name  = "cassandra.enabled"
    value = "false"
  }

  set {
    name  = "mysql.enabled"
    value = "false"
  }

  set {
    name  = "postgresql.enabled"
    value = "false"
  }

  set {
    name  = "elasticsearch.enabled"
    value = "false"
  }

  set {
    name  = "web.enabled"
    value = "true"
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
    name  = "server.priorityClassName"
    value = "platform"
  }

  depends_on = [
    kubernetes_manifest.cnpg_free,
    kubernetes_priority_class.platform,
  ]
}
