# =============================================================================
# Monitoring — Prometheus + Grafana + Alertmanager (5.16)
# =============================================================================

resource "helm_release" "prometheus_stack" {
  count = var.enable_monitoring ? 1 : 0

  name             = "kube-prometheus-stack"
  repository       = "https://prometheus-community.github.io/helm-charts"
  chart            = "kube-prometheus-stack"
  version          = var.prometheus_stack_version
  namespace        = "monitoring"
  create_namespace = true
  wait             = true
  timeout          = 600

  set_sensitive {
    name  = "grafana.adminPassword"
    value = var.admin_password
  }

  set {
    name  = "grafana.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "grafana.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "prometheus.prometheusSpec.retention"
    value = var.environment == "production" ? "90d" : "15d"
  }

  set {
    name  = "prometheus.prometheusSpec.resources.requests.cpu"
    value = "200m"
  }

  set {
    name  = "prometheus.prometheusSpec.resources.requests.memory"
    value = "512Mi"
  }

  set {
    name  = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName"
    value = "hcloud-volumes"
  }

  set {
    name  = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage"
    value = var.environment == "production" ? "50Gi" : "20Gi"
  }

  set {
    name  = "alertmanager.alertmanagerSpec.resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "alertmanager.alertmanagerSpec.resources.requests.memory"
    value = "64Mi"
  }

  # Scrape all ServiceMonitors/PodMonitors regardless of labels
  set {
    name  = "prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues"
    value = "false"
  }

  set {
    name  = "prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues"
    value = "false"
  }

  set {
    name  = "prometheus.prometheusSpec.priorityClassName"
    value = "platform"
  }

  depends_on = [kubernetes_priority_class.platform]
}

# =============================================================================
# Loki — Log Aggregation (5.17)
# =============================================================================

resource "helm_release" "loki" {
  count = var.enable_monitoring ? 1 : 0

  name       = "loki"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki"
  version    = var.loki_version
  namespace  = "monitoring"
  wait       = true
  timeout    = 600

  set {
    name  = "deploymentMode"
    value = "SingleBinary"
  }

  set {
    name  = "singleBinary.replicas"
    value = "1"
  }

  set {
    name  = "singleBinary.resources.requests.cpu"
    value = "100m"
  }

  set {
    name  = "singleBinary.resources.requests.memory"
    value = "256Mi"
  }

  set {
    name  = "loki.storage.type"
    value = "filesystem"
  }

  set {
    name  = "singleBinary.persistence.enabled"
    value = "true"
  }

  set {
    name  = "singleBinary.persistence.storageClass"
    value = "hcloud-volumes"
  }

  set {
    name  = "singleBinary.persistence.size"
    value = "10Gi"
  }

  set {
    name  = "loki.auth_enabled"
    value = "false"
  }

  # Schema config required by Loki 6.x+
  set {
    name  = "loki.schemaConfig.configs[0].from"
    value = "2024-04-01"
  }

  set {
    name  = "loki.schemaConfig.configs[0].store"
    value = "tsdb"
  }

  set {
    name  = "loki.schemaConfig.configs[0].object_store"
    value = "filesystem"
  }

  set {
    name  = "loki.schemaConfig.configs[0].schema"
    value = "v13"
  }

  set {
    name  = "loki.schemaConfig.configs[0].index.prefix"
    value = "loki_index_"
  }

  set {
    name  = "loki.schemaConfig.configs[0].index.period"
    value = "24h"
  }

  # Zero-out SimpleScalable replicas to avoid conflict with SingleBinary mode
  set {
    name  = "read.replicas"
    value = "0"
  }

  set {
    name  = "write.replicas"
    value = "0"
  }

  set {
    name  = "backend.replicas"
    value = "0"
  }

  # Disable chunksCache and resultsCache for staging (saves ~2Gi memory)
  set {
    name  = "chunksCache.enabled"
    value = "false"
  }

  set {
    name  = "resultsCache.enabled"
    value = "false"
  }

  depends_on = [helm_release.prometheus_stack]
}

# =============================================================================
# Tempo — Distributed Trace Storage (5.18)
# =============================================================================

resource "helm_release" "tempo" {
  count = var.enable_monitoring ? 1 : 0

  name       = "tempo"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "tempo"
  version    = var.tempo_version
  namespace  = "monitoring"
  wait       = true
  timeout    = 600

  set {
    name  = "tempo.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "tempo.resources.requests.memory"
    value = "128Mi"
  }

  set {
    name  = "persistence.enabled"
    value = "true"
  }

  set {
    name  = "persistence.storageClassName"
    value = "hcloud-volumes"
  }

  set {
    name  = "persistence.size"
    value = "10Gi"
  }

  # Fix permission denied on hcloud-volumes (fsGroup must be pod-level, not container-level)
  set {
    name  = "securityContext.fsGroup"
    value = "10001"
  }

  set {
    name  = "securityContext.runAsUser"
    value = "10001"
  }

  set {
    name  = "securityContext.runAsGroup"
    value = "10001"
  }

  depends_on = [helm_release.prometheus_stack]
}

# =============================================================================
# OpenTelemetry Collector — Trace & Metric Pipeline (5.19)
# =============================================================================

resource "helm_release" "otel_collector" {
  count = var.enable_monitoring ? 1 : 0

  name       = "otel-collector"
  repository = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart      = "opentelemetry-collector"
  version    = var.otel_collector_version
  namespace  = "monitoring"
  wait       = true
  timeout    = 300

  set {
    name  = "image.repository"
    value = "otel/opentelemetry-collector-contrib"
  }

  set {
    name  = "mode"
    value = "daemonset"
  }

  set {
    name  = "config.exporters.otlp.endpoint"
    value = "tempo.monitoring.svc.cluster.local:4317"
  }

  set {
    name  = "config.exporters.otlp.tls.insecure"
    value = "true"
  }

  set {
    name  = "config.receivers.otlp.protocols.grpc.endpoint"
    value = "0.0.0.0:4317"
  }

  set {
    name  = "config.receivers.otlp.protocols.http.endpoint"
    value = "0.0.0.0:4318"
  }

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }

  depends_on = [helm_release.tempo]
}

# =============================================================================
# Hubble UI IngressRoute (5.20)
# REMOVED (Phase 7: API-as-Proxy) — Hubble UI public access removed.
# Access via kubectl port-forward or API proxy if needed.
# =============================================================================
