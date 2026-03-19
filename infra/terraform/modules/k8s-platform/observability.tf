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
  timeout          = 900

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

  # --- Auth Proxy: trust Cloudflare Access header for SSO ---
  # After passing Cloudflare Zero Trust, the Cf-Access-Authenticated-User-Email
  # header contains the verified email. Grafana reads it and auto-logs in.
  set {
    name  = "grafana.grafana\\.ini.auth\\.proxy.enabled"
    value = "true"
  }

  set {
    name  = "grafana.grafana\\.ini.auth\\.proxy.header_name"
    value = "Cf-Access-Authenticated-User-Email"
  }

  set {
    name  = "grafana.grafana\\.ini.auth\\.proxy.header_property"
    value = "email"
  }

  set {
    name  = "grafana.grafana\\.ini.auth\\.proxy.auto_sign_up"
    value = "true"
  }

  set {
    name  = "grafana.grafana\\.ini.auth\\.proxy.headers"
    value = "Email:Cf-Access-Authenticated-User-Email"
  }

  set {
    name  = "grafana.grafana\\.ini.users.auto_assign_org_role"
    value = "Admin"
  }

  # --- Disable k3s false-positive alerts and duplicate-series rules ---
  # k3s bundles controller-manager, proxy, scheduler into a single process
  set {
    name  = "kubeControllerManager.enabled"
    value = "false"
  }

  set {
    name  = "kubeProxy.enabled"
    value = "false"
  }

  set {
    name  = "kubeScheduler.enabled"
    value = "false"
  }

  # k3s exposes kubelet metrics on multiple endpoints (/metrics, /metrics/cadvisor,
  # /metrics/probes), each returning kubelet_node_name with different labels.
  # This causes "duplicate series" errors in recording rules that join on it.
  # Fix: drop kubelet_node_name from non-primary endpoints via metricRelabelings.
  set {
    name  = "kubelet.serviceMonitor.cAdvisorMetricRelabelings[0].sourceLabels[0]"
    value = "__name__"
  }

  set {
    name  = "kubelet.serviceMonitor.cAdvisorMetricRelabelings[0].regex"
    value = "kubelet_node_name"
  }

  set {
    name  = "kubelet.serviceMonitor.cAdvisorMetricRelabelings[0].action"
    value = "drop"
  }

  set {
    name  = "kubelet.serviceMonitor.probesMetricRelabelings[0].sourceLabels[0]"
    value = "__name__"
  }

  set {
    name  = "kubelet.serviceMonitor.probesMetricRelabelings[0].regex"
    value = "kubelet_node_name"
  }

  set {
    name  = "kubelet.serviceMonitor.probesMetricRelabelings[0].action"
    value = "drop"
  }

  # --- Grafana plugins ---
  # Infinity: query any REST API (Hetzner Cloud, etc.)
  set {
    name  = "grafana.plugins[0]"
    value = "yesoreyeram-infinity-datasource"
  }

  # Cloudflare analytics
  set {
    name  = "grafana.plugins[1]"
    value = "cloudflare-app"
  }

  # --- Loki datasource ---
  set {
    name  = "grafana.additionalDataSources[0].name"
    value = "Loki"
  }

  set {
    name  = "grafana.additionalDataSources[0].type"
    value = "loki"
  }

  set {
    name  = "grafana.additionalDataSources[0].url"
    value = "http://loki.monitoring.svc.cluster.local:3100"
  }

  set {
    name  = "grafana.additionalDataSources[0].access"
    value = "proxy"
  }

  set {
    name  = "grafana.additionalDataSources[0].uid"
    value = "loki"
  }

  set {
    name  = "grafana.additionalDataSources[0].isDefault"
    value = "false"
  }

  # --- Tempo datasource ---
  set {
    name  = "grafana.additionalDataSources[1].name"
    value = "Tempo"
  }

  set {
    name  = "grafana.additionalDataSources[1].type"
    value = "tempo"
  }

  set {
    name  = "grafana.additionalDataSources[1].url"
    value = "http://tempo.monitoring.svc.cluster.local:3100"
  }

  set {
    name  = "grafana.additionalDataSources[1].access"
    value = "proxy"
  }

  set {
    name  = "grafana.additionalDataSources[1].uid"
    value = "tempo"
  }

  set {
    name  = "grafana.additionalDataSources[1].isDefault"
    value = "false"
  }

  # Tempo → Loki: click a trace span → see matching logs
  set {
    name  = "grafana.additionalDataSources[1].jsonData.tracesToLogsV2.datasourceUid"
    value = "loki"
  }

  set {
    name  = "grafana.additionalDataSources[1].jsonData.tracesToLogsV2.filterByTraceID"
    value = "true"
  }

  set {
    name  = "grafana.additionalDataSources[1].jsonData.tracesToLogsV2.filterBySpanID"
    value = "true"
  }

  # Service map (node graph) powered by Prometheus metrics
  set {
    name  = "grafana.additionalDataSources[1].jsonData.serviceMap.datasourceUid"
    value = "prometheus"
  }

  set {
    name  = "grafana.additionalDataSources[1].jsonData.nodeGraph.enabled"
    value = "true"
  }

  depends_on = [kubernetes_priority_class.platform]
}

# =============================================================================
# Grafana Dashboards — Auto-provisioned via sidecar (label: grafana_dashboard=1)
# =============================================================================

locals {
  dashboard_files = fileset("${path.module}/../../../helm/monitoring/dashboards", "*.json")
}

resource "kubernetes_config_map_v1" "grafana_dashboards" {
  for_each = var.enable_monitoring ? { for f in local.dashboard_files : trimsuffix(f, ".json") => f } : {}

  metadata {
    name      = "zenith-dashboard-${each.key}"
    namespace = "monitoring"
    labels = {
      grafana_dashboard                = "1"
      "app.kubernetes.io/part-of"      = "zenith"
      "app.kubernetes.io/component"    = "monitoring"
    }
    annotations = {
      grafana_folder = "Zenith"
    }
  }

  data = {
    "${each.value}" = file("${path.module}/../../../helm/monitoring/dashboards/${each.value}")
  }

  depends_on = [helm_release.prometheus_stack]
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

  # Wire traces pipeline to Tempo via otlp exporter (not just debug)
  set {
    name  = "config.service.pipelines.traces.exporters[0]"
    value = "otlp"
  }

  set {
    name  = "config.service.pipelines.traces.receivers[0]"
    value = "otlp"
  }

  set {
    name  = "config.service.pipelines.traces.receivers[1]"
    value = "jaeger"
  }

  set {
    name  = "config.service.pipelines.traces.receivers[2]"
    value = "zipkin"
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

# =============================================================================
# ServiceMonitors — Enable Prometheus scraping for all platform services
# =============================================================================

# ArgoCD metrics — ServiceMonitors created by ArgoCD Helm chart
# (server.metrics.serviceMonitor.enabled, controller.metrics.serviceMonitor.enabled, etc.)

# Cert-Manager metrics
resource "kubernetes_manifest" "servicemonitor_cert_manager" {
  count = var.enable_monitoring ? 1 : 0

  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "ServiceMonitor"
    metadata = {
      name      = "cert-manager"
      namespace = "cert-manager"
      labels = {
        release = "kube-prometheus-stack"
      }
    }
    spec = {
      endpoints = [
        { port = "tcp-prometheus-servicemonitor", interval = "60s", path = "/metrics" },
      ]
      selector = {
        matchLabels = {
          "app.kubernetes.io/name" = "cert-manager"
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_stack, helm_release.cert_manager]
}

# Kyverno metrics
resource "kubernetes_manifest" "servicemonitor_kyverno" {
  count = var.enable_kyverno && var.enable_monitoring ? 1 : 0

  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "ServiceMonitor"
    metadata = {
      name      = "kyverno"
      namespace = "kyverno"
      labels = {
        release = "kube-prometheus-stack"
      }
    }
    spec = {
      endpoints = [
        { port = "metrics-port", interval = "30s" },
      ]
      selector = {
        matchExpressions = [{
          key      = "app.kubernetes.io/component"
          operator = "In"
          values   = ["admission-controller", "background-controller", "cleanup-controller", "reports-controller"]
        }]
      }
    }
  }

  depends_on = [helm_release.prometheus_stack, helm_release.kyverno]
}

# Temporal metrics (all headless services expose port 9090)
resource "kubernetes_manifest" "servicemonitor_temporal" {
  count = var.enable_temporal && var.enable_monitoring ? 1 : 0

  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "ServiceMonitor"
    metadata = {
      name      = "temporal"
      namespace = "temporal"
      labels = {
        release = "kube-prometheus-stack"
      }
    }
    spec = {
      endpoints = [
        { port = "metrics", interval = "30s" },
      ]
      namespaceSelector = {
        matchNames = ["temporal"]
      }
      selector = {
        matchLabels = {
          "app.kubernetes.io/instance" = "temporal"
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_stack, helm_release.temporal]
}

# Loki metrics
resource "kubernetes_manifest" "servicemonitor_loki" {
  count = var.enable_monitoring ? 1 : 0

  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "ServiceMonitor"
    metadata = {
      name      = "loki"
      namespace = "monitoring"
      labels = {
        release = "kube-prometheus-stack"
      }
    }
    spec = {
      endpoints = [
        { port = "http-metrics", interval = "30s" },
      ]
      namespaceSelector = {
        matchNames = ["monitoring"]
      }
      selector = {
        matchLabels = {
          "app.kubernetes.io/name" = "loki"
        }
        matchExpressions = [
          {
            key      = "app.kubernetes.io/component"
            operator = "DoesNotExist"
          },
          {
            key      = "variant"
            operator = "DoesNotExist"
          },
        ]
      }
    }
  }

  depends_on = [helm_release.prometheus_stack, helm_release.loki]
}

# Tempo metrics
resource "kubernetes_manifest" "servicemonitor_tempo" {
  count = var.enable_monitoring ? 1 : 0

  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "ServiceMonitor"
    metadata = {
      name      = "tempo"
      namespace = "monitoring"
      labels = {
        release = "kube-prometheus-stack"
      }
    }
    spec = {
      endpoints = [
        { port = "tempo-prom-metrics", interval = "30s" },
      ]
      selector = {
        matchLabels = {
          "app.kubernetes.io/name" = "tempo"
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_stack, helm_release.tempo]
}

# Hubble metrics (Cilium network observability)
resource "kubernetes_manifest" "servicemonitor_hubble" {
  count = var.enable_monitoring ? 1 : 0

  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "ServiceMonitor"
    metadata = {
      name      = "hubble"
      namespace = "kube-system"
      labels = {
        release = "kube-prometheus-stack"
      }
    }
    spec = {
      endpoints = [
        { port = "hubble-metrics", interval = "30s" },
      ]
      namespaceSelector = {
        matchNames = ["kube-system"]
      }
      selector = {
        matchLabels = {
          "k8s-app" = "hubble"
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_stack]
}

# Velero metrics
resource "kubernetes_manifest" "servicemonitor_velero" {
  count = var.enable_velero && var.enable_monitoring ? 1 : 0

  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "ServiceMonitor"
    metadata = {
      name      = "velero"
      namespace = "velero"
      labels = {
        release = "kube-prometheus-stack"
      }
    }
    spec = {
      endpoints = [
        { port = "http-monitoring", interval = "60s" },
      ]
      namespaceSelector = {
        matchNames = ["velero"]
      }
      selector = {
        matchLabels = {
          "app.kubernetes.io/name" = "velero"
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_stack, helm_release.velero]
}

# APISIX metrics — ServiceMonitor created by APISIX Helm chart (metrics.serviceMonitor.enabled)

# Falcosidekick metrics (prometheus on port 2810)
resource "kubernetes_manifest" "servicemonitor_falco" {
  count = var.enable_falco && var.enable_monitoring ? 1 : 0

  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "ServiceMonitor"
    metadata = {
      name      = "falco"
      namespace = "falco"
      labels = {
        release = "kube-prometheus-stack"
      }
    }
    spec = {
      endpoints = [
        { port = "http", interval = "30s", path = "/metrics" },
      ]
      namespaceSelector = {
        matchNames = ["falco"]
      }
      selector = {
        matchLabels = {
          "app.kubernetes.io/name" = "falcosidekick"
        }
      }
    }
  }

  depends_on = [helm_release.prometheus_stack, helm_release.falco]
}

# Harbor metrics — ServiceMonitor created by Harbor Helm chart (metrics.serviceMonitor.enabled)
