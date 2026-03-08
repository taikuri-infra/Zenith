# =============================================================================
# V3 Messaging — NATS JetStream, RabbitMQ Operator, Strimzi Kafka Operator
# =============================================================================
# P3 migration: Internal event bus (NATS) + customer message queues
# Enable with enable_v3_operators = true
# =============================================================================

# --- NATS JetStream for internal events (P3-01) ---

resource "helm_release" "nats" {
  count = var.enable_v3_operators ? 1 : 0

  name             = "nats"
  repository       = "https://nats-io.github.io/k8s/helm/charts/"
  chart            = "nats"
  version          = var.nats_version
  namespace        = "nats"
  create_namespace = true
  wait             = true
  timeout          = 300

  values = [yamlencode({
    config = {
      jetstream = {
        enabled = true
        fileStore = {
          pvc = {
            size             = "5Gi"
            storageClassName = "hcloud-volumes"
          }
        }
      }
      cluster = {
        enabled  = false # Single node for staging; enable for production
        replicas = 1
      }
    }
    container = {
      merge = {
        resources = {
          requests = {
            cpu    = "50m"
            memory = "128Mi"
          }
          limits = {
            memory = "256Mi"
          }
        }
      }
    }
    promExporter = {
      enabled = true
      podMonitor = {
        enabled = true
      }
    }
  })]

  depends_on = [helm_release.prometheus_stack]
}

# --- RabbitMQ Cluster Operator (P3-06) ---

resource "helm_release" "rabbitmq_operator" {
  count = var.enable_v3_operators ? 1 : 0

  name             = "rabbitmq-cluster-operator"
  repository       = "https://charts.bitnami.com/bitnami"
  chart            = "rabbitmq-cluster-operator"
  version          = var.rabbitmq_operator_version
  namespace        = "rabbitmq-operator"
  create_namespace = true
  wait             = true
  timeout          = 300

  set {
    name  = "clusterOperator.resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "clusterOperator.resources.requests.memory"
    value = "64Mi"
  }

  set {
    name  = "msgTopologyOperator.resources.requests.cpu"
    value = "25m"
  }

  set {
    name  = "msgTopologyOperator.resources.requests.memory"
    value = "64Mi"
  }
}

# --- Strimzi Kafka Operator (P3-11) ---

resource "helm_release" "strimzi_kafka_operator" {
  count = var.enable_v3_operators ? 1 : 0

  name             = "strimzi-kafka-operator"
  repository       = "https://strimzi.io/charts/"
  chart            = "strimzi-kafka-operator"
  version          = var.strimzi_version
  namespace        = "kafka-operator"
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

  # Watch all namespaces for Kafka CRDs
  set {
    name  = "watchNamespaces"
    value = "*"
  }
}
