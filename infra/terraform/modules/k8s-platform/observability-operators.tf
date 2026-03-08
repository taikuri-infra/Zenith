# =============================================================================
# V3 Observability Operators — Loki Operator, OTel Operator, Tempo Operator
# =============================================================================
# These replace the existing Helm-based Loki/Tempo/OTel deployments.
# Enable with enable_v3_operators = true. Once validated, disable the old
# enable_monitoring Helm releases and remove them.
# =============================================================================

# --- Loki Operator (P1-01) ---

resource "helm_release" "loki_operator" {
  count = var.enable_v3_operators ? 1 : 0

  name             = "loki-operator"
  repository       = "https://grafana.github.io/helm-charts"
  chart            = "loki-operator"
  version          = var.loki_operator_version
  namespace        = "monitoring"
  create_namespace = true
  wait             = true
  timeout          = 600

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }

  depends_on = [helm_release.prometheus_stack]
}

# --- LokiStack CRD (P1-02) ---

resource "kubernetes_manifest" "lokistack" {
  count = var.enable_v3_operators ? 1 : 0

  manifest = {
    apiVersion = "loki.grafana.com/v1"
    kind       = "LokiStack"
    metadata = {
      name      = "zenith-logs"
      namespace = "monitoring"
    }
    spec = {
      size = "1x.extra-small"
      storage = {
        schemas = [{
          version       = "v13"
          effectiveDate = "2024-04-01"
        }]
        secret = {
          name = "loki-s3-credentials"
          type = "s3"
        }
      }
      storageClassName = "hcloud-volumes"
      tenants = {
        mode = "openshift-logging"
      }
      limits = {
        global = {
          retention = {
            days = var.environment == "production" ? 90 : 15
          }
        }
      }
    }
  }

  depends_on = [helm_release.loki_operator]
}

# --- Loki S3 Credentials Secret (P1-03) ---

resource "kubernetes_secret" "loki_s3_credentials" {
  count = var.enable_v3_operators ? 1 : 0

  metadata {
    name      = "loki-s3-credentials"
    namespace = "monitoring"
  }

  data = {
    access_key_id     = var.s3_access_key
    access_key_secret = var.s3_secret_key
    bucketnames       = "zenith-loki-${var.environment}"
    endpoint          = var.s3_endpoint
    region            = "fsn1"
  }
}

# --- OTel Operator (P1-05) ---

resource "helm_release" "otel_operator" {
  count = var.enable_v3_operators ? 1 : 0

  name             = "opentelemetry-operator"
  repository       = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart            = "opentelemetry-operator"
  version          = var.otel_operator_version
  namespace        = "monitoring"
  create_namespace = true
  wait             = true
  timeout          = 600

  set {
    name  = "manager.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "manager.resources.requests.memory"
    value = "128Mi"
  }

  # cert-manager is already installed, OTel Operator uses it for webhook certs
  set {
    name  = "admissionWebhooks.certManager.enabled"
    value = "true"
  }

  depends_on = [helm_release.prometheus_stack]
}

# --- OpenTelemetryCollector CRD — DaemonSet mode for logs/metrics (P1-06) ---

resource "kubernetes_manifest" "otel_collector_daemonset" {
  count = var.enable_v3_operators ? 1 : 0

  manifest = {
    apiVersion = "opentelemetry.io/v1beta1"
    kind       = "OpenTelemetryCollector"
    metadata = {
      name      = "zenith-collector"
      namespace = "monitoring"
    }
    spec = {
      mode = "daemonset"
      resources = {
        requests = {
          cpu    = "50m"
          memory = "128Mi"
        }
        limits = {
          memory = "256Mi"
        }
      }
      config = yamlencode({
        receivers = {
          otlp = {
            protocols = {
              grpc = { endpoint = "0.0.0.0:4317" }
              http = { endpoint = "0.0.0.0:4318" }
            }
          }
        }
        processors = {
          batch = {
            timeout         = "5s"
            send_batch_size = 1024
          }
          memory_limiter = {
            check_interval         = "1s"
            limit_percentage       = 75
            spike_limit_percentage = 25
          }
        }
        exporters = {
          otlp = {
            endpoint = "tempo-zenith-traces-distributor.monitoring.svc.cluster.local:4317"
            tls      = { insecure = true }
          }
          prometheusremotewrite = {
            endpoint = "http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090/api/v1/write"
          }
        }
        service = {
          pipelines = {
            traces = {
              receivers  = ["otlp"]
              processors = ["memory_limiter", "batch"]
              exporters  = ["otlp"]
            }
            metrics = {
              receivers  = ["otlp"]
              processors = ["memory_limiter", "batch"]
              exporters  = ["prometheusremotewrite"]
            }
          }
        }
      })
    }
  }

  depends_on = [helm_release.otel_operator]
}

# --- Instrumentation CRD for auto-instrumentation (P1-07) ---

resource "kubernetes_manifest" "otel_instrumentation" {
  count = var.enable_v3_operators ? 1 : 0

  manifest = {
    apiVersion = "opentelemetry.io/v1alpha1"
    kind       = "Instrumentation"
    metadata = {
      name      = "zenith-instrumentation"
      namespace = "zenith-apps"
    }
    spec = {
      exporter = {
        endpoint = "http://zenith-collector-collector.monitoring.svc.cluster.local:4317"
      }
      propagators = ["tracecontext", "baggage"]
      sampler = {
        type     = "parentbased_traceidratio"
        argument = "0.1"
      }
      go = {
        image = "ghcr.io/open-telemetry/opentelemetry-go-instrumentation/autoinstrumentation-go:latest"
      }
      nodejs = {
        image = "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:latest"
      }
    }
  }

  depends_on = [helm_release.otel_operator]
}

# --- Tempo Operator (P1-08) ---

resource "helm_release" "tempo_operator" {
  count = var.enable_v3_operators ? 1 : 0

  name             = "tempo-operator"
  repository       = "https://grafana.github.io/helm-charts"
  chart            = "tempo-operator"
  version          = var.tempo_operator_version
  namespace        = "monitoring"
  create_namespace = true
  wait             = true
  timeout          = 600

  set {
    name  = "resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "resources.requests.memory"
    value = "128Mi"
  }

  depends_on = [helm_release.prometheus_stack]
}

# --- TempoStack CRD with S3 backend (P1-09) ---

resource "kubernetes_manifest" "tempostack" {
  count = var.enable_v3_operators ? 1 : 0

  manifest = {
    apiVersion = "tempo.grafana.com/v1alpha1"
    kind       = "TempoStack"
    metadata = {
      name      = "zenith-traces"
      namespace = "monitoring"
    }
    spec = {
      storage = {
        secret = {
          name = "tempo-s3-credentials"
          type = "s3"
        }
      }
      storageSize = "10Gi"
      resources = {
        total = {
          limits = {
            cpu    = "200m"
            memory = "256Mi"
          }
        }
      }
      template = {
        queryFrontend = {
          jaegerQuery = {
            enabled = true
          }
        }
      }
      retention = {
        global = {
          traces = var.environment == "production" ? "720h" : "168h"
        }
      }
    }
  }

  depends_on = [helm_release.tempo_operator]
}

# --- Tempo S3 Credentials Secret (P1-10) ---

resource "kubernetes_secret" "tempo_s3_credentials" {
  count = var.enable_v3_operators ? 1 : 0

  metadata {
    name      = "tempo-s3-credentials"
    namespace = "monitoring"
  }

  data = {
    access_key_id     = var.s3_access_key
    access_key_secret = var.s3_secret_key
    bucket            = "zenith-tempo-${var.environment}"
    endpoint          = var.s3_endpoint
    region            = "fsn1"
  }
}

# --- Grafana Datasources Update (P1-11) ---
# Grafana in kube-prometheus-stack auto-discovers datasources via ConfigMaps
# with the label grafana_datasource: "1"

resource "kubernetes_config_map" "grafana_datasources_v3" {
  count = var.enable_v3_operators ? 1 : 0

  metadata {
    name      = "grafana-datasources-v3"
    namespace = "monitoring"
    labels = {
      grafana_datasource = "1"
    }
  }

  data = {
    "v3-datasources.yaml" = yamlencode({
      apiVersion = 1
      datasources = [
        {
          name      = "Loki (Operator)"
          type      = "loki"
          url       = "http://zenith-logs-gateway.monitoring.svc.cluster.local:3100"
          access    = "proxy"
          isDefault = false
        },
        {
          name      = "Tempo (Operator)"
          type      = "tempo"
          url       = "http://tempo-zenith-traces-query-frontend.monitoring.svc.cluster.local:3200"
          access    = "proxy"
          isDefault = false
        },
      ]
    })
  }
}
