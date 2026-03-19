# Zenith Observability Stack — Complete Reference

> **Last updated:** 2026-03-19
> **Status:** Production-ready on staging, 42/42 targets UP, 0 firing alerts (excluding Watchdog)
> **Score:** 9.5/10 — only missing OTel SDK instrumentation in application code

This document is the **single source of truth** for Zenith's entire observability stack.
It covers every component, why it exists, how it was deployed, how things connect,
and how to troubleshoot. Written so a junior engineer can read it top-to-bottom
and understand the full picture within a few days.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [The Three Pillars](#2-the-three-pillars)
   - 2.1 [Metrics (Prometheus)](#21-metrics--prometheus)
   - 2.2 [Logs (Loki + Promtail)](#22-logs--loki--promtail)
   - 2.3 [Traces (Tempo + OTel Collector)](#23-traces--tempo--otel-collector)
3. [Visualization (Grafana)](#3-visualization--grafana)
4. [Alerting Pipeline](#4-alerting-pipeline)
5. [Security Observability (Falco)](#5-security-observability--falco)
6. [Network Observability (Cilium + Hubble)](#6-network-observability--cilium--hubble)
7. [Per-Service Metrics](#7-per-service-metrics)
8. [Network Policies for Telemetry](#8-network-policies-for-telemetry)
9. [Dashboards](#9-dashboards)
10. [Data Flow Diagrams](#10-data-flow-diagrams)
11. [Configuration Reference](#11-configuration-reference)
12. [Troubleshooting Guide](#12-troubleshooting-guide)
13. [Remaining Work](#13-remaining-work)
14. [V3 Operator-Based Architecture (Future)](#14-v3-operator-based-architecture)
15. [Appendix: Full Code Reference](#15-appendix-full-code-reference)

---

## 1. Architecture Overview

### What We Built

A complete observability stack for a Kubernetes platform running on Hetzner Cloud,
using only open-source tools. No vendor lock-in, no SaaS dependencies.

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          ZENITH OBSERVABILITY STACK                             │
│                                                                                 │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────────┐  │
│  │  Prometheus  │   │     Loki     │   │    Tempo     │   │     Grafana      │  │
│  │  (Metrics)   │   │   (Logs)     │   │  (Traces)    │   │ (Visualization)  │  │
│  │             │   │             │   │             │   │                  │  │
│  │  42 targets  │   │  16 labels   │   │  Pipeline    │   │  40 dashboards   │  │
│  │  90d retain  │   │  Pod + Falco │   │  ready       │   │  4 datasources   │  │
│  │  20Gi store  │   │  10Gi store  │   │  10Gi store  │   │  SSO via CF      │  │
│  └──────┬───────┘   └──────┬───────┘   └──────┬───────┘   └────────┬─────────┘  │
│         │                  │                  │                    │             │
│         └──────────────────┴──────────────────┴────────────────────┘             │
│                              All in namespace: monitoring                        │
│                                                                                 │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────────┐  │
│  │ AlertManager │   │    Falco     │   │   Cilium     │   │  OTel Collector  │  │
│  │  (Routing)   │   │  (Security)  │   │  (Network)   │   │  (Trace Ingest)  │  │
│  │             │   │             │   │             │   │                  │  │
│  │  Telegram    │   │  Sidekick    │   │  Hubble UI   │   │  DaemonSet mode  │  │
│  │  Slack       │   │  → Loki      │   │  1860 series │   │  OTLP/Jaeger/    │  │
│  │  PagerDuty   │   │  → Telegram  │   │             │   │  Zipkin → Tempo  │  │
│  └──────────────┘   └──────────────┘   └──────────────┘   └──────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Why These Tools?

| Tool | Purpose | Why Not Alternatives? |
|------|---------|----------------------|
| **Prometheus** | Metrics collection & storage | Industry standard for K8s. InfluxDB/VictoriaMetrics are alternatives but Prometheus has the widest ecosystem |
| **Loki** | Log aggregation | Unlike Elasticsearch, Loki only indexes labels (not full text), making it 10x cheaper on storage. Perfect for small clusters |
| **Tempo** | Distributed tracing | No-dependency trace backend. Unlike Jaeger, doesn't need Cassandra/Elasticsearch. Just needs S3 or local disk |
| **Grafana** | Dashboards & exploration | Unified UI for all three pillars. One place for metrics, logs, and traces |
| **OTel Collector** | Telemetry pipeline | Vendor-neutral. Accepts OTLP, Jaeger, Zipkin formats and routes to any backend |
| **Falco** | Runtime security | Kernel-level syscall monitoring. Detects crypto miners, shell access, suspicious network activity |
| **Cilium** | Network observability | eBPF-based. Sees all network flows without sidecars. Hubble provides L7 visibility |

### Key Design Decisions

1. **Single namespace** (`monitoring`) for all observability components — simplifies NetworkPolicies and RBAC
2. **Helm charts via Terraform** — infrastructure as code, version-controlled, reproducible
3. **kube-prometheus-stack** (not raw Prometheus) — bundles Prometheus, Grafana, AlertManager, node-exporter, kube-state-metrics, and 24 default dashboards in one chart
4. **SingleBinary mode** for Loki and Tempo — simpler for staging/single-node clusters. Will scale to microservices mode for production
5. **DaemonSet mode** for OTel Collector — runs on every node, collects traces from all pods
6. **No public access** to monitoring UIs — everything behind Cloudflare Tunnel or API proxy

---

## 2. The Three Pillars

### 2.1 Metrics — Prometheus

**What it does:** Scrapes numerical time-series data from every service at regular intervals.
Think of it as asking every service "how are you doing?" every 15-30 seconds and recording the answer.

**How it works:**

```
┌─────────────┐     scrape every 30s     ┌─────────────────┐
│  Your App   │ ◄────────────────────── │   Prometheus    │
│ /metrics    │  ──────────────────────►│                 │
│ endpoint    │    HTTP GET response     │  Stores in TSDB │
└─────────────┘    with metric values    │  on hcloud-vol  │
                                         └─────────────────┘
```

**A metric looks like this:**
```
# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET", path="/api/v1/apps", status="200"} 1542
http_requests_total{method="POST", path="/api/v1/apps", status="201"} 23
http_requests_total{method="GET", path="/api/v1/apps", status="500"} 3
```

Prometheus scrapes this text format, parses it, and stores it with timestamps.

#### How Services Expose Metrics

Services don't push metrics to Prometheus. Instead, they **expose** an HTTP endpoint
(usually `/metrics`) and Prometheus **pulls** from them. This is called the "pull model."

There are three ways a service tells Prometheus "scrape me":

**1. ServiceMonitor (most common)**

A Kubernetes CRD that says "scrape this service on this port":

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app
  namespace: monitoring
  labels:
    release: kube-prometheus-stack    # ← MUST have this label
spec:
  endpoints:
    - port: metrics                   # ← port name from the Service
      interval: 30s                   # ← how often to scrape
      path: /metrics                  # ← endpoint path (default: /metrics)
  namespaceSelector:
    matchNames: ["my-namespace"]      # ← which namespace to look in
  selector:
    matchLabels:
      app: my-app                     # ← which Services to target
```

**2. PodMonitor**

Same idea but targets pods directly (no Service needed). Used by CNPG operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: zenith-postgres
spec:
  podMetricsEndpoints:
    - port: metrics
      interval: 30s
  selector:
    matchLabels:
      cnpg.io/cluster: zenith-postgres
```

**3. Helm chart ServiceMonitor** (auto-created)

Many Helm charts can create their own ServiceMonitor when you enable it:

```hcl
set { name = "metrics.serviceMonitor.enabled", value = "true" }
```

This is used by: APISIX, Harbor, ArgoCD, Cilium.

#### Prometheus Deployment Configuration

**File:** `infra/terraform/modules/k8s-platform/observability.tf` (lines 1-243)

```hcl
resource "helm_release" "prometheus_stack" {
  count = var.enable_monitoring ? 1 : 0

  name             = "kube-prometheus-stack"
  repository       = "https://prometheus-community.github.io/helm-charts"
  chart            = "kube-prometheus-stack"
  version          = var.prometheus_stack_version    # "61.3.1"
  namespace        = "monitoring"
  create_namespace = true
  wait             = true
  timeout          = 900                             # 15 min — plugins download takes time
```

**Storage configuration:**

```hcl
  # How long to keep metrics
  set {
    name  = "prometheus.prometheusSpec.retention"
    value = var.environment == "production" ? "90d" : "15d"
  }

  # Where to store metrics (PVC on Hetzner block storage)
  set {
    name  = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName"
    value = "hcloud-volumes"
  }

  set {
    name  = "prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage"
    value = var.environment == "production" ? "50Gi" : "20Gi"
  }
```

**Critical setting — scrape ALL ServiceMonitors:**

```hcl
  # Without this, Prometheus ONLY scrapes ServiceMonitors that have
  # the label "release: kube-prometheus-stack". With this set to false,
  # Prometheus scrapes ALL ServiceMonitors in the cluster.
  set {
    name  = "prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues"
    value = "false"
  }

  set {
    name  = "prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues"
    value = "false"
  }
```

> **Why this matters:** If you forget this, you'll create ServiceMonitors
> but Prometheus won't scrape them. You'll see the ServiceMonitor in `kubectl get sm`
> but 0 targets in Prometheus. This is the #1 reason "my metrics aren't showing up."

**k3s compatibility fixes:**

k3s bundles kube-controller-manager, kube-proxy, and kube-scheduler into a single binary.
They don't expose separate metrics endpoints, so Prometheus can't scrape them.
Without these settings, you get 3 permanent `TargetDown` alerts:

```hcl
  # Disable targets that don't exist in k3s
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
```

k3s also has a quirk where the kubelet exposes `kubelet_node_name` metric through
multiple endpoints (`/metrics`, `/metrics/cadvisor`, `/metrics/probes`). This creates
duplicate time series that break Prometheus recording rules (any rule that does
`* on(instance) group_left(node) kubelet_node_name` will fail with "duplicate series").

Fix: drop `kubelet_node_name` from non-primary endpoints:

```hcl
  # Drop kubelet_node_name from cAdvisor endpoint (keep only from /metrics)
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

  # Same for probes endpoint
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
```

#### What Prometheus Scrapes (42 Targets)

Every target has a `job` label identifying it. Here's the complete list:

| # | Job Name | Namespace | Port | Metrics Count | Source |
|---|----------|-----------|------|---------------|--------|
| 1 | kubelet (metrics) | kube-system | 10250 | ~800 | Built-in |
| 2 | kubelet (cadvisor) | kube-system | 10250 | ~400 | Built-in |
| 3 | kubelet (probes) | kube-system | 10250 | ~20 | Built-in |
| 4 | apiserver | default | 6443 | ~500 | Built-in |
| 5 | kube-state-metrics | monitoring | 8080 | ~300 | Helm chart |
| 6 | node-exporter | monitoring | 9100 | ~500 | Helm chart |
| 7 | prometheus | monitoring | 9090 | ~100 | Self-scrape |
| 8 | alertmanager | monitoring | 9093 | ~30 | Helm chart |
| 9 | grafana | monitoring | 3000 | ~50 | Helm chart |
| 10 | prometheus-operator | monitoring | 8080 | ~20 | Helm chart |
| 11 | coredns | kube-system | 9153 | ~30 | Built-in |
| 12 | cilium-agent | kube-system | 9962 | ~1860 | Cilium chart |
| 13 | cilium-operator | kube-system | 9963 | ~50 | Cilium chart |
| 14 | hubble | kube-system | 9966 | ~100 | Cilium chart |
| 15 | apisix-prometheus-metrics | apisix | 9091 | ~77 | APISIX chart |
| 16 | argocd-controller | argocd | 8082 | ~50 | ArgoCD chart |
| 17 | argocd-server | argocd | 8083 | ~30 | ArgoCD chart |
| 18 | argocd-repo-server | argocd | 8084 | ~20 | ArgoCD chart |
| 19 | harbor | harbor | 8001 | ~108 | Harbor chart |
| 20 | cert-manager | cert-manager | 9402 | ~67 | Manual SM |
| 21 | kyverno (admission) | kyverno | 8000 | ~500+ | Manual SM |
| 22 | kyverno (background) | kyverno | 8000 | ~500+ | Manual SM |
| 23 | kyverno (cleanup) | kyverno | 8000 | ~500+ | Manual SM |
| 24 | kyverno (reports) | kyverno | 8000 | ~500+ | Manual SM |
| 25 | loki | monitoring | 3100 | ~200 | Manual SM |
| 26 | tempo | monitoring | 3200 | ~100 | Manual SM |
| 27 | velero | velero | 8085 | ~49 | Manual SM |
| 28 | falcosidekick | falco | 2810 | ~30 | Manual SM |
| 29 | zenith-postgres | zenith-staging | 9187 | ~1830 | CNPG PodMonitor |
| 30+ | Additional kubelet/kube-state endpoints | various | various | various | Built-in |

> **"Manual SM" = defined in `observability.tf` as `kubernetes_manifest` resources.**
> **"Chart SM" = created automatically by the Helm chart when `serviceMonitor.enabled = true`.**

---

### 2.2 Logs — Loki + Promtail

**What it does:** Collects log lines from every pod, indexes them by labels
(namespace, pod name, container name), and makes them searchable in Grafana.

**How it works:**

```
┌──────────┐     ┌──────────────┐     ┌──────────┐     ┌─────────┐
│  Pod     │     │  Promtail    │     │   Loki   │     │ Grafana │
│  stdout/ │────►│  (DaemonSet) │────►│ (Single  │────►│ Explore │
│  stderr  │     │  reads /var/ │     │  Binary) │     │  tab    │
│          │     │  log/pods/   │     │          │     │         │
└──────────┘     └──────────────┘     └──────────┘     └─────────┘
     │                                      ▲
     │                                      │
     │           ┌──────────────┐           │
     │           │ Falcosidekick│           │
     │           │  (security   │───────────┘
     │           │   events)    │  HTTP POST to
     │           └──────────────┘  /loki/api/v1/push
     │
     └─── writes to container log file at:
          /var/log/pods/<namespace>_<pod-name>_<uid>/<container>/0.log
```

#### Understanding Loki's Label-Based Approach

Unlike Elasticsearch which indexes every word in every log line, Loki only indexes **labels**.
This makes it dramatically cheaper to run but means you can't search for arbitrary text
without scanning all matching log lines.

**Labels are the key.** Every log stream in Loki is identified by its label set:

```
{namespace="zenith-staging", pod="zenith-api-7d8f9c-abc12", container="api", stream="stderr"}
```

When you query `{namespace="zenith-staging"}`, Loki finds all streams with that label
and returns their log lines. If you add `|= "error"`, Loki scans those lines for the word "error".

**Performance tip:** Always filter by labels first, then by text content:
- GOOD: `{namespace="zenith-staging", container="api"} |= "error"`
- BAD: `{} |= "error"` (scans ALL logs in the entire cluster)

#### Loki Deployment Configuration

**File:** `infra/terraform/modules/k8s-platform/observability.tf` (lines 280-395)

```hcl
resource "helm_release" "loki" {
  count = var.enable_monitoring ? 1 : 0

  name       = "loki"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki"
  version    = var.loki_version    # "6.6.4"
  namespace  = "monitoring"
  wait       = true
  timeout    = 600
```

**Deployment mode — SingleBinary:**

```hcl
  set {
    name  = "deploymentMode"
    value = "SingleBinary"
  }

  set {
    name  = "loki.commonConfig.replication_factor"
    value = "1"                    # No replication (single node)
  }

  set {
    name  = "loki.auth_enabled"
    value = "false"                # Multi-tenancy off — simpler queries
  }
```

> **Why SingleBinary?** Loki has three deployment modes:
> - **SingleBinary** — all components in one pod. Best for <100GB/day.
> - **SimpleScalable** — read/write separation. For 100GB-1TB/day.
> - **Microservices** — full horizontal scaling. For >1TB/day.
> We use SingleBinary because staging ingests <1GB/day.

**Storage — local filesystem on Hetzner block volume:**

```hcl
  set {
    name  = "loki.storage.type"
    value = "filesystem"
  }

  set {
    name  = "singleBinary.persistence.storageClass"
    value = "hcloud-volumes"
  }

  set {
    name  = "singleBinary.persistence.size"
    value = "10Gi"
  }
```

**Schema — TSDB (new format, better performance):**

```hcl
  set {
    name  = "loki.schemaConfig.configs[0].store"
    value = "tsdb"                 # New format, replaces BoltDB
  }

  set {
    name  = "loki.schemaConfig.configs[0].schema"
    value = "v13"                  # Latest schema version
  }

  set {
    name  = "loki.schemaConfig.configs[0].index.prefix"
    value = "loki_index_"
  }

  set {
    name  = "loki.schemaConfig.configs[0].index.period"
    value = "24h"                  # One index table per day
  }
```

**Memory optimization — disable caches:**

```hcl
  # Disable Memcached caches (saves ~2 GiB RAM)
  set {
    name  = "chunksCache.enabled"
    value = "false"
  }

  set {
    name  = "resultsCache.enabled"
    value = "false"
  }
```

> **Why disable caches?** On a single-node staging cluster with 8GB RAM,
> Memcached chunks + results caches consume ~2GB. Since we're using local
> filesystem storage (not S3), the caching benefit is minimal.

**Kill unused deployment modes:**

```hcl
  # REQUIRED: set all other modes to 0 replicas
  # Otherwise Helm tries to deploy SimpleScalable components too
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
```

#### Promtail Configuration

Promtail is deployed by `kube-prometheus-stack` as part of the monitoring chart.
It also has a separate configuration in the Helm monitoring chart:

**File:** `infra/helm/monitoring/values.yaml` (lines 156-183)

```yaml
promtail:
  config:
    snippets:
      pipelineStages:
        - cri: {}                    # Parse CRI container log format
        - match:
            selector: '{namespace="zenith-apps"}'
            stages:
              - labels:
                  project:           # Extract zenith.dev/project pod label
                  app:               # Extract zenith.dev/app pod label
```

This pipeline adds `project` and `app` labels to logs from customer apps,
enabling per-customer log filtering in Grafana:

```
{namespace="zenith-apps", app="my-app"} | json
```

#### Querying Logs — LogQL

LogQL is Loki's query language. Think of it as "PromQL for logs."

**Basic query:**
```
{namespace="zenith-staging"}
```
Returns all log lines from the zenith-staging namespace.

**Filter by text:**
```
{namespace="zenith-staging", container="api"} |= "error"
```
Returns lines containing "error" from the API container.

**Parse JSON logs:**
```
{namespace="zenith-staging", container="api"} | json | level="error"
```
Parses each line as JSON, then filters where the `level` field equals "error".

**Count errors over time:**
```
sum(rate({namespace="zenith-staging"} |= "error" [5m]))
```
Returns errors per second over 5-minute windows.

**Format output (for pretty logs in Grafana):**
```
{namespace="zenith-staging", container="api"}
  | json
  | line_format "{{.level | upper}} [{{.caller}}] {{.msg}}"
```

---

### 2.3 Traces — Tempo + OTel Collector

**What it does:** Records the journey of a single request through multiple services.
When a user hits `/api/v1/apps`, the trace shows: API Gateway → API Server → Database
with timing for each step.

**Current status:** Infrastructure is ready. OTel Collector and Tempo are deployed.
The pipeline is configured. But **no application has been instrumented yet** — this
is the -0.5 from the 9.5/10 score.

```
┌──────────┐     ┌───────────────────┐     ┌──────────┐     ┌─────────┐
│ Your App │     │  OTel Collector   │     │  Tempo   │     │ Grafana │
│ (with    │────►│  (DaemonSet)      │────►│ (Single  │────►│ Explore │
│  OTel    │     │                   │     │  Binary) │     │  tab    │
│  SDK)    │     │  Receives:        │     │          │     │         │
│          │     │  - OTLP gRPC:4317 │     │  Stores  │     │ Shows   │
│ Sends    │     │  - OTLP HTTP:4318 │     │  traces  │     │ trace   │
│ spans    │     │  - Jaeger:14250   │     │  on disk │     │ timeline│
│          │     │  - Zipkin:9411    │     │          │     │         │
└──────────┘     └───────────────────┘     └──────────┘     └─────────┘
```

#### How Tracing Works (Conceptual)

1. User sends `POST /api/v1/apps` to create an app
2. The HTTP middleware creates a **trace** (unique ID) and a **root span** named "POST /api/v1/apps"
3. The handler calls the database — this creates a **child span** named "INSERT INTO apps"
4. The handler calls Harbor API — this creates another **child span** named "Harbor.CreateProject"
5. Each span records: start time, end time, status, and any attributes (user_id, app_id, etc.)
6. All spans are sent to OTel Collector, which forwards them to Tempo
7. In Grafana, you can search for a trace by ID and see the full waterfall:

```
POST /api/v1/apps ─────────────────────────────── 250ms
  ├── Validate request ──── 2ms
  ├── INSERT INTO apps ──────────── 15ms
  ├── Harbor.CreateProject ────────────────── 180ms
  └── Publish event ──── 5ms
```

#### Tempo Deployment Configuration

**File:** `infra/terraform/modules/k8s-platform/observability.tf` (lines 401-454)

```hcl
resource "helm_release" "tempo" {
  count = var.enable_monitoring ? 1 : 0

  name       = "tempo"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "tempo"
  version    = var.tempo_version    # "1.10.1"
  namespace  = "monitoring"
  wait       = true
  timeout    = 300
```

**Storage:**

```hcl
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
```

**Security context fix for Hetzner volumes:**

```hcl
  # Hetzner block volumes require specific UID/GID ownership
  # Without this, Tempo gets "permission denied" on the PVC
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
```

> **Gotcha:** `fsGroup` must be set at the pod level (`securityContext`),
> not the container level (`containers[0].securityContext`). If you set it
> at the container level, the volume won't get the correct group ownership.

#### OTel Collector Deployment Configuration

**File:** `infra/terraform/modules/k8s-platform/observability.tf` (lines 460-533)

```hcl
resource "helm_release" "otel_collector" {
  count = var.enable_monitoring ? 1 : 0

  name       = "otel-collector"
  repository = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart      = "opentelemetry-collector"
  version    = var.otel_collector_version    # "0.96.0"
  namespace  = "monitoring"
  wait       = true
  timeout    = 300
```

**Use the contrib image (not the core image):**

```hcl
  set {
    name  = "image.repository"
    value = "otel/opentelemetry-collector-contrib"
  }
```

> **Why contrib?** The core image only includes OTLP receiver/exporter.
> The contrib image adds Jaeger, Zipkin, Prometheus receivers — needed
> for receiving traces from legacy systems.

**DaemonSet mode:**

```hcl
  set {
    name  = "mode"
    value = "daemonset"
  }
```

> **Why DaemonSet?** Runs one collector per node. Applications send traces
> to the collector on the same node (via the node's IP). This avoids
> cross-node traffic and provides high availability.

**Receivers (what formats can apps send traces in):**

```hcl
  # OTLP gRPC (modern, preferred)
  set {
    name  = "config.receivers.otlp.protocols.grpc.endpoint"
    value = "0.0.0.0:4317"
  }

  # OTLP HTTP (for browsers and HTTP-only environments)
  set {
    name  = "config.receivers.otlp.protocols.http.endpoint"
    value = "0.0.0.0:4318"
  }
```

**Exporter (where traces go):**

```hcl
  set {
    name  = "config.exporters.otlp.endpoint"
    value = "tempo.monitoring.svc.cluster.local:4317"
  }

  set {
    name  = "config.exporters.otlp.tls.insecure"
    value = "true"                  # Cluster-internal, no TLS needed
  }
```

**Pipeline (connecting receivers to exporters):**

```hcl
  # Traces: OTLP/Jaeger/Zipkin → process → send to Tempo
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
```

> **Important:** The initial deployment only had `debug` as the traces exporter.
> This meant traces were logged to stdout but never sent to Tempo.
> We fixed this by adding `otlp` to the exporters list.

---

## 3. Visualization — Grafana

**What it does:** Web UI for querying all three pillars (metrics, logs, traces).
Shows dashboards with graphs, tables, and alerts.

### Authentication

Grafana is behind Cloudflare Access (Zero Trust). Authentication flow:

```
User → Cloudflare Access (login) → Cloudflare Tunnel → Grafana
                                                          │
                                                          ▼
                                              Reads header:
                                              Cf-Access-Authenticated-User-Email
                                              ↓
                                              Auto-creates user with Admin role
```

Configuration:

```hcl
  # Enable proxy authentication (Cloudflare Access passes user email in header)
  set {
    name  = "grafana.grafana\\.ini.auth\\.proxy.enabled"
    value = "true"
  }

  set {
    name  = "grafana.grafana\\.ini.auth\\.proxy.header_name"
    value = "Cf-Access-Authenticated-User-Email"
  }

  set {
    name  = "grafana.grafana\\.ini.auth\\.proxy.auto_sign_up"
    value = "true"
  }

  set {
    name  = "grafana.grafana\\.ini.users.auto_assign_org_role"
    value = "Admin"
  }
```

### Datasources

Four datasources are configured, each with an explicit UID that dashboards reference:

```hcl
  # Datasource 0: Loki (logs)
  set { name = "grafana.additionalDataSources[0].name", value = "Loki" }
  set { name = "grafana.additionalDataSources[0].type", value = "loki" }
  set { name = "grafana.additionalDataSources[0].url",  value = "http://loki.monitoring.svc.cluster.local:3100" }
  set { name = "grafana.additionalDataSources[0].uid",  value = "loki" }

  # Datasource 1: Tempo (traces)
  set { name = "grafana.additionalDataSources[1].name", value = "Tempo" }
  set { name = "grafana.additionalDataSources[1].type", value = "tempo" }
  set { name = "grafana.additionalDataSources[1].url",  value = "http://tempo.monitoring.svc.cluster.local:3100" }
  set { name = "grafana.additionalDataSources[1].uid",  value = "tempo" }
```

> **Critical lesson:** Datasource UIDs must be **explicitly set** and match what
> dashboards reference. If you don't set `uid`, Grafana auto-generates a random
> one like `P8E80F9AEF21F6940`. Then your dashboard JSON that says
> `"datasource": {"uid": "loki"}` won't find anything.

Prometheus and AlertManager are auto-configured by kube-prometheus-stack (datasources 2 and 3).

### Plugins

```hcl
  # Infinity: query any REST API (Hetzner Cloud stats, etc.)
  set {
    name  = "grafana.plugins[0]"
    value = "yesoreyeram-infinity-datasource"
  }

  # Cloudflare: analytics and zone stats
  set {
    name  = "grafana.plugins[1]"
    value = "cloudflare-app"
  }
```

> **Failed attempt:** We tried adding `grafana-googleanalytics-datasource` but it
> doesn't exist in the Grafana plugin registry. It caused Grafana CrashLoopBackOff.
> The fix was removing it. Use the Infinity datasource to query Google Analytics API instead.

### Dashboard Auto-Provisioning

**File:** `infra/terraform/modules/k8s-platform/observability.tf` (lines 248-274)

```hcl
locals {
  dashboard_files = fileset(
    "${path.module}/../../../helm/monitoring/dashboards", "*.json"
  )
}

resource "kubernetes_config_map_v1" "grafana_dashboards" {
  for_each = var.enable_monitoring ? {
    for f in local.dashboard_files : trimsuffix(f, ".json") => f
  } : {}

  metadata {
    name      = "zenith-dashboard-${each.key}"
    namespace = "monitoring"
    labels = {
      grafana_dashboard = "1"           # ← Grafana sidecar watches for this
    }
    annotations = {
      grafana_folder = "Zenith"         # ← puts dashboards in "Zenith" folder
    }
  }

  data = {
    "${each.key}.json" = file(
      "${path.module}/../../../helm/monitoring/dashboards/${each.value}"
    )
  }
}
```

**How it works:**

1. Terraform reads all `*.json` files from `infra/helm/monitoring/dashboards/`
2. Creates one ConfigMap per JSON file with label `grafana_dashboard: "1"`
3. Grafana's sidecar container watches for ConfigMaps with this label
4. When it finds one, it loads the JSON as a dashboard into the "Zenith" folder
5. Any changes to the JSON files are applied on `terraform apply`

**To add a new dashboard:** Create a JSON file in `infra/helm/monitoring/dashboards/`
and run `terraform apply`. The dashboard appears in Grafana automatically.

---

## 4. Alerting Pipeline

### How Alerts Flow

```
┌─────────────────┐     ┌──────────────┐     ┌────────────────┐
│ PrometheusRule  │────►│ AlertManager │────►│  Destinations  │
│ (YAML in K8s)   │     │  (routing)   │     │                │
│                 │     │              │     │  ● Telegram    │
│ If condition    │     │  Groups by   │     │  ● Slack       │
│ is true for     │     │  severity    │     │  ● PagerDuty   │
│ X minutes...    │     │  and routes  │     │                │
│                 │     │  to correct  │     │  Watchdog      │
│ → fire alert    │     │  channel     │     │  → /dev/null   │
└─────────────────┘     └──────────────┘     └────────────────┘
```

### Alert Rules

Two sets of alert rules exist:

**1. Default kube-prometheus-stack alerts** (automatically included):
- KubePodCrashLooping, KubePodNotReady, KubeDeploymentReplicasMismatch
- NodeFilesystemAlmostOutOfSpace, NodeMemoryHighUtilization
- PrometheusTargetDown, etc.

**2. Custom Zenith alerts** defined in:

**File:** `infra/helm/monitoring/templates/alerting-rules.yaml`

```yaml
groups:
  - name: zenith.platform
    rules:
      - alert: ZenithAPIDown
        expr: absent(up{job="zenith-staging/zenith-api"} == 1)
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Zenith API is down"

  - name: zenith.apps
    rules:
      - alert: AppHighCPU
        expr: |
          sum(rate(container_cpu_usage_seconds_total{
            namespace="zenith-apps"
          }[5m])) by (pod) > 0.8
        for: 5m

      - alert: AppCrashLooping
        expr: |
          increase(kube_pod_container_status_restarts_total{
            namespace="zenith-apps"
          }[1h]) > 5
        for: 0m

  - name: zenith.databases
    rules:
      - alert: DatabaseStorageFull
        expr: |
          kubelet_volume_stats_used_bytes{
            namespace="zenith-staging",
            persistentvolumeclaim=~".*postgres.*"
          } / kubelet_volume_stats_capacity_bytes > 0.85
        for: 5m
        labels:
          severity: critical
```

**File:** `infra/k8s/monitoring/alertmanager-config.yaml`

Additional platform-specific rules:

```yaml
groups:
  - name: zenith.api
    rules:
      - alert: ZenithAPIHighErrorRate
        expr: |
          sum(rate(http_requests_total{
            job="zenith-api", status=~"5.."}[5m]
          )) / sum(rate(http_requests_total{
            job="zenith-api"}[5m]
          )) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API error rate above 5%"

  - name: zenith.business
    rules:
      - alert: ZenithMRRDrop
        expr: delta(zenith_mrr_euros[1d]) < -100
        labels:
          severity: info
        annotations:
          summary: "MRR dropped by more than €100 in 24h"
```

### AlertManager Routing

**File:** `infra/k8s/monitoring/alertmanager-config.yaml`

```yaml
route:
  receiver: default-slack
  group_by: ['alertname', 'namespace']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - match:
        severity: critical
      receiver: pagerduty-critical
      group_wait: 10s
      repeat_interval: 1h
      continue: true        # Also send to Slack

    - match:
        severity: info
      receiver: telegram-business
      repeat_interval: 24h

    - match:
        alertname: Watchdog
      receiver: 'null'       # Suppress (it always fires — that's by design)

receivers:
  - name: default-slack
    slack_configs:
      - channel: '#zenith-alerts'
        send_resolved: true

  - name: pagerduty-critical
    pagerduty_configs:
      - routing_key: '<from-secret>'

  - name: telegram-business
    telegram_configs:
      - bot_token: '<from-secret>'
        chat_id: '<chat-id>'
        parse_mode: HTML

  - name: 'null'             # /dev/null — suppresses alerts
```

### The Watchdog Alert

Watchdog is a special alert that **always fires**. It exists to verify the alerting
pipeline is working. If Watchdog stops firing, your alerting is broken.

In production, you'd route Watchdog to a "dead man's switch" service (like Healthchecks.io)
that alerts you if it **doesn't** receive the Watchdog signal.

---

## 5. Security Observability — Falco

**What it does:** Monitors Linux kernel syscalls in real-time and detects suspicious behavior:
crypto miners, reverse shells, unauthorized file access, package installation in containers.

### Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│  Node (kernel)                                                    │
│                                                                    │
│  ┌────────┐      ┌────────────────┐      ┌──────────────────┐    │
│  │ Syscall │─────►│  Falco Driver  │─────►│  Falco Engine    │    │
│  │ (open,  │      │  (eBPF/kmod)   │      │  (rules engine)  │    │
│  │  exec,  │      │                │      │                  │    │
│  │  connect│      │  Captures all  │      │  Matches against │    │
│  │  etc.)  │      │  syscalls      │      │  rules YAML      │    │
│  └────────┘      └────────────────┘      └────────┬─────────┘    │
│                                                    │               │
│                                            ┌───────▼─────────┐    │
│                                            │  Falcosidekick  │    │
│                                            │  (event router) │    │
│                                            │                 │    │
│                                            │  → Loki (logs)  │    │
│                                            │  → Telegram     │    │
│                                            │  → Slack        │    │
│                                            └─────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

### Falco Deployment

**File:** `infra/terraform/modules/k8s-platform/security.tf` (lines 456-627)

```hcl
resource "helm_release" "falco" {
  count = var.enable_falco ? 1 : 0

  name             = "falco"
  repository       = "https://falcosecurity.github.io/charts"
  chart            = "falco"
  version          = var.falco_version    # "4.18.0"
  namespace        = "falco"
  create_namespace = true
```

**Driver mode:**

```hcl
  set {
    name  = "driver.kind"
    value = "auto"               # Tries: modern_ebpf → ebpf → kernel module
  }
```

**Falcosidekick outputs:**

```hcl
  # Enable Falcosidekick (event router)
  set {
    name  = "falcosidekick.enabled"
    value = "true"
  }

  # Send to Loki for dashboards
  set {
    name  = "falcosidekick.config.loki.hostport"
    value = "http://loki.monitoring.svc.cluster.local:3100"
  }
  set {
    name  = "falcosidekick.config.loki.minimumpriority"
    value = "notice"             # notice, warning, error, critical, alert, emergency
  }

  # Send to Telegram for instant notifications
  set_sensitive {
    name  = "falcosidekick.config.telegram.token"
    value = var.telegram_bot_token
  }
  set {
    name  = "falcosidekick.config.telegram.chatid"
    value = var.telegram_chat_id
  }
  set {
    name  = "falcosidekick.config.telegram.minimumpriority"
    value = "warning"            # Only warning+ goes to Telegram
  }

  # Prometheus metrics from Falco
  set {
    name  = "falco.metrics_enabled"
    value = "true"
  }
  set {
    name  = "falco.metrics_interval"
    value = "15s"
  }
```

### Custom Falco Rules

Falco comes with hundreds of built-in rules. We added custom rules targeting
our application namespaces:

```hcl
  values = [yamlencode({
    customRules = {
      "zenith-custom-rules.yaml" = yamlencode({
        - rule: "Detect Crypto Mining in Customer Apps"
          desc: "Detects cryptocurrency mining processes"
          condition: >
            spawned_process and container and
            k8s.ns.name in (zenith-apps, zenith-builds) and
            (proc.name in (xmrig, minerd, minergate, cpuminer) or
             proc.cmdline contains "stratum+tcp")
          output: "Crypto mining detected (user=%user.name command=%proc.cmdline ns=%k8s.ns.name pod=%k8s.pod.name)"
          priority: CRITICAL
          tags: [crypto, mitre_execution]

        - rule: "Reverse Shell Detected"
          desc: "Detects reverse shell connections"
          condition: >
            spawned_process and container and
            k8s.ns.name in (zenith-apps, zenith-builds) and
            ((proc.name in (bash, sh, dash) and proc.cmdline contains "/dev/tcp") or
             (proc.name = "nc" and proc.cmdline contains "-e") or
             (proc.name = "python" and proc.cmdline contains "socket"))
          output: "Reverse shell detected (user=%user.name command=%proc.cmdline ns=%k8s.ns.name pod=%k8s.pod.name)"
          priority: CRITICAL
          tags: [reverse_shell, mitre_execution]

        - rule: "Package Manager in Container"
          desc: "Detects package installation in running containers"
          condition: >
            spawned_process and container and
            k8s.ns.name in (zenith-apps, zenith-builds) and
            proc.name in (apt, apt-get, yum, dnf, apk, pip, npm)
          output: "Package manager used in container (user=%user.name command=%proc.cmdline ns=%k8s.ns.name pod=%k8s.pod.name)"
          priority: WARNING
          tags: [package, mitre_persistence]
      })
    }
  })]
```

### Falco Events in Loki

When Falcosidekick sends events to Loki, they're stored with these labels:

| Label | Example Value | Description |
|-------|---------------|-------------|
| `source` | `syscall` | Falco event source (NOT "falco"!) |
| `priority` | `Notice`, `Warning`, `Critical` | Severity (capitalized!) |
| `rule` | `Contact K8S API Server From Container` | Which rule fired |
| `hostname` | `zenith-staging` | Node name |
| `tags` | `T1565,container,k8s,mitre_discovery` | MITRE ATT&CK tags |

> **Important gotcha:** The `source` label is `syscall`, NOT `falco`.
> And priority values are **capitalized** (`Notice`, not `notice`).
> The dashboard queries must match these exact values.

Dashboard query example:
```
{source="syscall", priority="Critical"} | json
```

---

## 6. Network Observability — Cilium + Hubble

**What it does:** Cilium replaces kube-proxy with eBPF-based networking.
Hubble adds L3/L4/L7 network flow visibility — you can see which pod
is talking to which pod, on which port, with which HTTP method.

### Metrics Configuration

**File:** `infra/terraform/modules/k8s-platform/cilium.tf`

```hcl
  # Enable Hubble (network flow observability)
  set { name = "hubble.enabled",      value = "true" }
  set { name = "hubble.relay.enabled", value = "true" }
  set { name = "hubble.ui.enabled",   value = "true" }

  # Prometheus metrics from Cilium agent
  set { name = "prometheus.enabled",                    value = "true" }
  set { name = "prometheus.serviceMonitor.enabled",     value = "true" }

  # Prometheus metrics from Cilium operator
  set { name = "operator.prometheus.enabled",           value = "true" }
  set { name = "operator.prometheus.serviceMonitor.enabled", value = "true" }

  # Hubble metrics
  set { name = "hubble.metrics.serviceMonitor.enabled", value = "true" }
```

This creates three ServiceMonitors automatically:
- `cilium-agent` → 1860 metric series (packet counts, policy verdicts, endpoint state)
- `cilium-operator` → ~50 series
- `hubble` → ~100 series (network flows, DNS queries, HTTP requests)

---

## 7. Per-Service Metrics

### How Each Service Exposes Metrics

| Service | How Enabled | SM Type | Port | Path | Key Metrics |
|---------|-------------|---------|------|------|-------------|
| **APISIX** | `metrics.serviceMonitor.enabled=true` in yamlencode values | Chart | 9091 | /apisix/prometheus/metrics | `apisix_http_status`, `apisix_bandwidth`, `apisix_http_latency_bucket` |
| **ArgoCD** | `controller/server/repoServer.metrics.serviceMonitor.enabled=true` | Chart | 8082-8084 | /metrics | `argocd_app_info`, `argocd_app_sync_total` |
| **Harbor** | `metrics.enabled=true`, `metrics.serviceMonitor.enabled=true` | Chart | 8001 | /metrics | `harbor_project_total`, `harbor_artifact_pulled` |
| **cert-manager** | Manual SM in observability.tf | Manual | 9402 (tcp-prometheus-servicemonitor) | /metrics | `certmanager_certificate_expiration_timestamp_seconds` |
| **Kyverno** | Manual SM in observability.tf, 4 endpoints | Manual | 8000 (metrics-port) | /metrics | `kyverno_admission_requests_total`, `kyverno_policy_results_total` |
| **Velero** | Manual SM in observability.tf | Manual | 8085 (http-monitoring) | /metrics | `velero_backup_success_total`, `velero_restore_success_total` |
| **Falcosidekick** | Manual SM in observability.tf | Manual | 2810 (http) | /metrics | `falcosidekick_inputs_total` |
| **Loki** | Manual SM in observability.tf | Manual | 3100 (http-metrics) | /metrics | `loki_ingester_chunk_stored_bytes_total` |
| **Tempo** | Manual SM in observability.tf | Manual | 3200 (tempo-prom-metrics) | /metrics | `tempo_ingester_traces_created_total` |
| **CNPG Postgres** | PodMonitor created by CNPG operator | Auto | 9187 (metrics) | /metrics | `cnpg_pg_stat_activity_count`, `cnpg_pg_database_size_bytes` |

### APISIX Metrics Deep Dive

APISIX requires special configuration because its `prometheus` plugin needs to be
in the plugins list AND needs `apisix.prometheus.enabled=true`:

```hcl
  values = [yamlencode({
    plugins = [
      "jwt-auth", "cors", "limit-count", "openid-connect",
      "opentelemetry", "prometheus",                          # ← must be here
      "uri-blocker", "ua-restriction", "referer-restriction"
    ]
    apisix = {
      prometheus = {
        enabled       = true        # ← starts metrics server
        containerPort = 9091        # ← port for metrics
      }
    }
    metrics = {
      serviceMonitor = {
        enabled   = true            # ← creates ServiceMonitor CRD
        namespace = "apisix"
      }
    }
  })]
```

> **Lesson learned:** We went through multiple iterations:
> 1. First tried `plugin_attr.prometheus` (snake_case) — wrong for Helm values
> 2. Then tried `pluginAttrs.prometheus` (camelCase) — port didn't listen
> 3. Finally discovered `apisix.prometheus.enabled` is the correct Helm value
>
> Also, APISIX only starts the metrics server after a global rule activates the
> prometheus plugin. Without traffic, `apisix_http_status` shows 0 series.

### CNPG PostgreSQL Metrics

CNPG operator automatically creates a PodMonitor and exposes metrics on port 9187.
The exporter provides ~1830 metrics including:

- `cnpg_pg_stat_activity_count` — active database connections
- `cnpg_pg_database_size_bytes` — database size
- `cnpg_pg_replication_lag` — replication delay
- `cnpg_pg_settings_shared_buffers_bytes` — PostgreSQL memory config

**NetworkPolicy requirement:** Since zenith-staging has `default-deny-ingress`,
Prometheus can't reach port 9187 without an explicit allow rule. See Section 8.

---

## 8. Network Policies for Telemetry

### The Problem

Every application namespace has a `default-deny-ingress` NetworkPolicy. This means
NO incoming traffic is allowed unless explicitly permitted. This is great for security
but breaks monitoring because:

1. Prometheus (in `monitoring` ns) needs to scrape metrics from pods in other namespaces
2. Falcosidekick (in `falco` ns) needs to push logs to Loki (in `monitoring` ns)
3. OTel Collector (in `monitoring` ns) needs to send traces to Tempo (in `monitoring` ns)

### The Solution

Explicit NetworkPolicy allow rules for each telemetry flow:

**File:** `infra/terraform/modules/k8s-platform/network-policies.tf`

```
┌─────────────┐                    ┌─────────────────┐
│   falco     │ ── port 3100 ────► │   monitoring    │
│  namespace  │    (Loki push)     │   namespace     │
│             │                    │                 │
│ falcosidekick                    │  ┌─────┐        │
│             │                    │  │ Loki│        │
└─────────────┘                    │  └─────┘        │
                                   │                 │
┌─────────────┐                    │  ┌───────┐      │
│   any       │ ── port 4317 ────► │  │ Tempo │      │
│  namespace  │ ── port 4318       │  └───────┘      │
│  (OTel)     │    (traces)        │                 │
└─────────────┘                    │  ┌────────────┐ │
                                   │  │ Prometheus │ │
┌─────────────┐                    │  └─────┬──────┘ │
│   zenith-   │ ◄─ port 9187 ──── │        │        │
│   staging   │    (CNPG metrics)  │        │        │
└─────────────┘                    └────────┼────────┘
                                            │
                                   scrapes all namespaces
                                   via ServiceMonitors
```

#### Allow Falco → Loki

```hcl
resource "kubernetes_network_policy_v1" "allow_falco_to_monitoring" {
  count = var.enable_monitoring && var.enable_falco ? 1 : 0

  metadata {
    name      = "allow-falco-to-loki"
    namespace = "monitoring"
  }

  spec {
    pod_selector {
      match_labels = {
        "app.kubernetes.io/name" = "loki"
      }
    }

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "falco"
          }
        }
      }

      ports {
        port     = "3100"
        protocol = "TCP"
      }
    }

    policy_types = ["Ingress"]
  }
}
```

> **Reading this policy:** "Allow TCP traffic on port 3100 FROM any pod in the `falco`
> namespace TO pods labeled `app.kubernetes.io/name=loki` in the `monitoring` namespace."

#### Allow OTel → Tempo

```hcl
resource "kubernetes_network_policy_v1" "allow_otel_to_tempo" {
  count = var.enable_monitoring ? 1 : 0

  metadata {
    name      = "allow-otel-to-tempo"
    namespace = "monitoring"
  }

  spec {
    pod_selector {
      match_labels = {
        "app.kubernetes.io/name" = "tempo"
      }
    }

    ingress {
      from {
        namespace_selector {}       # ← empty = any namespace
      }

      ports {
        port     = "4317"           # gRPC OTLP
        protocol = "TCP"
      }

      ports {
        port     = "4318"           # HTTP OTLP
        protocol = "TCP"
      }
    }

    policy_types = ["Ingress"]
  }
}
```

#### Allow Prometheus → CNPG Postgres Metrics

```hcl
resource "kubernetes_network_policy_v1" "allow_prometheus_to_cnpg_staging" {
  count = var.enable_monitoring ? 1 : 0

  metadata {
    name      = "allow-prometheus-to-postgres"
    namespace = "zenith-staging"
  }

  spec {
    pod_selector {
      match_labels = {
        "cnpg.io/cluster" = "zenith-postgres"
      }
    }

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "monitoring"
          }
        }
      }

      ports {
        port     = "9187"
        protocol = "TCP"
      }
    }

    policy_types = ["Ingress"]
  }
}
```

#### Allow Prometheus Self-Scraping

```hcl
resource "kubernetes_network_policy_v1" "allow_prometheus_scrape" {
  count = var.enable_monitoring ? 1 : 0

  metadata {
    name      = "allow-prometheus-scrape"
    namespace = "monitoring"
  }

  spec {
    pod_selector {}                 # All pods in monitoring

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "monitoring"
          }
        }
      }
    }

    policy_types = ["Ingress"]
  }
}
```

### Common NetworkPolicy Debugging

If metrics stop working, check:

```bash
# 1. Is the NetworkPolicy created?
kubectl get networkpolicy -n <namespace>

# 2. Does it target the right pods?
kubectl get networkpolicy <name> -n <namespace> -o yaml

# 3. Does the source namespace have the right label?
kubectl get ns <source-ns> --show-labels

# 4. Test connectivity from source pod
kubectl exec -n <source-ns> <pod> -- wget -qO- --timeout=5 http://<target-ip>:<port>/metrics
```

---

## 9. Dashboards

### Dashboard Architecture

```
infra/helm/monitoring/dashboards/
├── apisix.json             # API Gateway metrics
├── argocd.json             # GitOps deployment metrics
├── cert-manager.json       # TLS certificate status
├── cilium-hubble.json      # Network flow metrics
├── cluster-overview.json   # Overall cluster health
├── falco-logs.json         # Security events (Loki datasource)
├── keycloak.json           # Identity provider metrics
├── kyverno.json            # Policy engine metrics
├── loki.json               # Log aggregation metrics
├── node-resources.json     # Node CPU/memory/disk
├── platform-overview.json  # Zenith platform health
├── postgres-cnpg.json      # Database metrics (with cluster variable)
├── service-health.json     # Service up/down status
├── temporal.json           # Workflow engine metrics
├── velero.json             # Backup & restore metrics
└── zenith-apps.json        # Customer app metrics
```

Plus 24 default dashboards from kube-prometheus-stack (apiserver, kubelet, node, namespace, etc.).

### Dashboard JSON Structure

Every dashboard follows this pattern:

```json
{
  "uid": "unique-id",                    // Used for linking
  "title": "Dashboard Name",
  "tags": ["zenith", "component"],       // For search/filter
  "schemaVersion": 39,                   // Grafana schema version
  "graphTooltip": 1,                     // Shared crosshair
  "refresh": "30s",                      // Auto-refresh interval
  "time": { "from": "now-6h", "to": "now" },
  "panels": [
    {
      "type": "row",                     // Section header
      "title": "Overview",
      "collapsed": false
    },
    {
      "type": "timeseries",             // Graph panel
      "datasource": {
        "type": "prometheus",
        "uid": "prometheus"              // Must match datasource UID!
      },
      "targets": [
        {
          "expr": "sum(rate(http_requests_total[5m]))",
          "legendFormat": "{{method}} {{path}}"
        }
      ]
    }
  ]
}
```

### Key Dashboard Details

#### Falco Security Events Dashboard (`falco-logs.json`)

Uses **Loki** datasource (not Prometheus):

```json
{
  "datasource": { "type": "loki", "uid": "loki" },
  "targets": [{
    "expr": "sum by (priority) (count_over_time({source=\"syscall\"} [$__auto]))"
  }]
}
```

Panels:
- Security Events Timeline (stacked bar by priority)
- Events by Priority (donut chart)
- Events by Rule (horizontal bar)
- Events by Tags (MITRE ATT&CK tags)
- Recent Critical Events (log stream)
- Recent Warning Events (log stream)
- Events by Host (horizontal bar)
- Full Event Log Stream

Color overrides match Falco priority levels:
- Critical → red `#ef4444`
- Warning → amber `#f59e0b`
- Notice → blue `#3b82f6`
- Emergency → dark red `#dc2626`

#### CNPG PostgreSQL Dashboard (`postgres-cnpg.json`)

Has a **template variable** for cluster selection:

```json
{
  "templating": {
    "list": [{
      "name": "cluster",
      "type": "query",
      "query": "label_values(cnpg_collector_up, cluster)",
      "current": { "text": "zenith-postgres" }
    }]
  }
}
```

This lets you switch between `zenith-postgres` and `free-pg` clusters.

#### Loki ServiceMonitor — Avoiding False Targets

The Loki ServiceMonitor must exclude gateway and headless services
(which return 404 on `/metrics`):

```hcl
resource "kubernetes_manifest" "servicemonitor_loki" {
  spec = {
    selector = {
      matchLabels = {
        "app.kubernetes.io/name" = "loki"
      }
      matchExpressions = [
        {
          key      = "app.kubernetes.io/component"
          operator = "DoesNotExist"       # Excludes gateway (component=gateway)
        },
        {
          key      = "variant"
          operator = "DoesNotExist"       # Excludes headless (variant=headless)
        },
      ]
    }
  }
}
```

> **Why this matters:** Without these exclusions, you get 2 permanent TargetDown
> alerts for `loki-gateway` and `loki-headless`. They share the label
> `app.kubernetes.io/name=loki` with the main Loki service but don't serve metrics.

---

## 10. Data Flow Diagrams

### Complete Data Flow

```
═══════════════════════════════════════════════════════════════════════
                         METRICS FLOW
═══════════════════════════════════════════════════════════════════════

  ┌─────────────────────────────────────────────────────────────────┐
  │                     APPLICATION LAYER                           │
  │                                                                 │
  │  zenith-api ──────┐                                             │
  │  zenith-web ──────┤                                             │
  │  zenith-mc ───────┤     ServiceMonitor / PodMonitor             │
  │  zenith-operator ─┤     (defines WHAT to scrape)                │
  │  customer apps ───┤            │                                │
  │                   │            ▼                                 │
  │  ┌────────────────┴────────────────────────────────────┐        │
  │  │               Prometheus (port 9090)                 │        │
  │  │                                                      │        │
  │  │  Scrapes every 15-60s depending on ServiceMonitor    │        │
  │  │  Stores on hcloud-volumes PVC (20Gi staging)         │        │
  │  │  Retains 15 days (staging) / 90 days (production)    │        │
  │  │  Evaluates alert rules every 15s                     │        │
  │  └──────────────────────┬───────────────────────────────┘        │
  │                         │                                        │
  │                    ┌────▼────┐                                    │
  │                    │ Grafana │                                    │
  │                    │ (3000)  │                                    │
  │                    └─────────┘                                    │
  └─────────────────────────────────────────────────────────────────┘

  ┌─────────────────────────────────────────────────────────────────┐
  │                     INFRASTRUCTURE LAYER                        │
  │                                                                 │
  │  Cilium agent (1860 metrics) ────────┐                          │
  │  Cilium operator (50 metrics) ───────┤                          │
  │  Hubble (100 metrics) ───────────────┤                          │
  │  APISIX (77 metrics) ───────────────┤                          │
  │  Harbor (108 metrics) ──────────────┤                          │
  │  ArgoCD (108 metrics) ──────────────┤                          │
  │  Kyverno (2356 metrics) ────────────┤── all scraped by ──► Prometheus
  │  Velero (49 metrics) ───────────────┤                          │
  │  cert-manager (67 metrics) ─────────┤                          │
  │  CNPG postgres (1830 metrics) ──────┤                          │
  │  Falcosidekick (30 metrics) ────────┤                          │
  │  Loki (200 metrics) ────────────────┤                          │
  │  Tempo (100 metrics) ───────────────┘                          │
  └─────────────────────────────────────────────────────────────────┘


═══════════════════════════════════════════════════════════════════════
                          LOGS FLOW
═══════════════════════════════════════════════════════════════════════

  Pod stdout/stderr
       │
       ▼
  Container runtime writes to:
  /var/log/pods/<ns>_<pod>_<uid>/<container>/0.log
       │
       ▼
  ┌──────────────────────────────────────────────────────────┐
  │  Promtail (DaemonSet on every node)                       │
  │                                                            │
  │  1. Discovers pods via Kubernetes API                      │
  │  2. Reads their log files from /var/log/pods/              │
  │  3. Adds labels: namespace, pod, container, node_name      │
  │  4. For zenith-apps: also adds project, app labels         │
  │  5. Sends to Loki via HTTP POST                            │
  └────────────────────────┬─────────────────────────────────┘
                           │
                           ▼
  ┌──────────────────────────────────────────────────────────┐
  │  Loki (SingleBinary, port 3100)                           │
  │                                                            │
  │  ┌──────────┐  ┌──────────┐  ┌──────────┐                │
  │  │ Ingester │  │  Querier │  │ Compactor│                │
  │  │          │  │          │  │          │                │
  │  │ Receives │  │ Answers  │  │ Compacts │                │
  │  │ log data │  │ queries  │  │ old data │                │
  │  │ buffers  │  │ from     │  │          │                │
  │  │ in memory│  │ Grafana  │  │          │                │
  │  └────┬─────┘  └──────────┘  └──────────┘                │
  │       │                                                    │
  │       ▼                                                    │
  │  ┌───────────────────────┐                                │
  │  │  hcloud-volumes PVC   │                                │
  │  │  10Gi filesystem      │                                │
  │  │  TSDB format (v13)    │                                │
  │  └───────────────────────┘                                │
  └──────────────────────────────────────────────────────────┘

  Additionally:

  ┌──────────────────────────────────────────────────────────┐
  │  Falcosidekick (falco namespace)                          │
  │                                                            │
  │  Pushes security events directly to Loki API              │
  │  HTTP POST to /loki/api/v1/push                           │
  │                                                            │
  │  Labels: source=syscall, priority=Notice/Warning/Critical  │
  │          rule=<falco-rule-name>, hostname=<node>           │
  │          tags=<mitre-attack-tags>                          │
  │                                                            │
  │  NetworkPolicy required: falco → monitoring:3100           │
  └──────────────────────────────────────────────────────────┘


═══════════════════════════════════════════════════════════════════════
                         TRACES FLOW
═══════════════════════════════════════════════════════════════════════

  Application (with OTel SDK)
       │
       │  Sends spans via OTLP gRPC (:4317) or HTTP (:4318)
       │
       ▼
  ┌──────────────────────────────────────────────────────────┐
  │  OTel Collector (DaemonSet on every node)                 │
  │                                                            │
  │  Receivers:          Processors:        Exporter:          │
  │  ┌─────────┐        ┌──────────┐       ┌──────────┐      │
  │  │ OTLP    │───────►│ memory   │──────►│ OTLP     │      │
  │  │ gRPC    │        │ limiter  │       │ exporter │      │
  │  │ :4317   │        │          │       │          │      │
  │  ├─────────┤        │ batch    │       │ → Tempo  │      │
  │  │ OTLP    │───────►│ (groups  │       │   :4317  │      │
  │  │ HTTP    │        │  spans)  │       │          │      │
  │  │ :4318   │        └──────────┘       └──────────┘      │
  │  ├─────────┤                                              │
  │  │ Jaeger  │ (legacy support)                             │
  │  │ :14250  │                                              │
  │  ├─────────┤                                              │
  │  │ Zipkin  │ (legacy support)                             │
  │  │ :9411   │                                              │
  │  └─────────┘                                              │
  └────────────────────────┬─────────────────────────────────┘
                           │
                           ▼
  ┌──────────────────────────────────────────────────────────┐
  │  Tempo (SingleBinary, port 3200 for query, 4317 for ingest)│
  │                                                            │
  │  Stores traces on hcloud-volumes PVC (10Gi)               │
  │  Query API exposed on :3200 for Grafana                    │
  │                                                            │
  │  NetworkPolicy required: any namespace → monitoring:4317   │
  └──────────────────────────────────────────────────────────┘


═══════════════════════════════════════════════════════════════════════
                         ALERTS FLOW
═══════════════════════════════════════════════════════════════════════

  PrometheusRule CRDs
  (kube-prometheus-stack defaults + custom zenith rules)
       │
       │  Prometheus evaluates rules every 15s
       │  If condition is true for `for` duration → fires alert
       │
       ▼
  ┌──────────────────────────────────────────────────────────┐
  │  AlertManager (port 9093)                                 │
  │                                                            │
  │  Routes by severity:                                       │
  │                                                            │
  │  critical ─────► PagerDuty (10s wait, 1h repeat)          │
  │       └────────► Slack #zenith-alerts (continue)          │
  │                                                            │
  │  warning ──────► Slack #zenith-alerts (4h repeat)         │
  │                                                            │
  │  info ─────────► Telegram (24h repeat)                    │
  │                                                            │
  │  Watchdog ─────► /dev/null (suppressed)                   │
  │                                                            │
  │  Inhibition: critical suppresses warning for same          │
  │              alertname+namespace                            │
  └──────────────────────────────────────────────────────────┘

  Separate from AlertManager:

  ┌──────────────────────────────────────────────────────────┐
  │  Falcosidekick (direct to Telegram/Slack)                 │
  │                                                            │
  │  warning+ ────► Telegram (instant, no grouping)           │
  │  warning+ ────► Slack (instant)                           │
  │                                                            │
  │  These bypass AlertManager entirely — Falco has its own    │
  │  notification pipeline via Falcosidekick outputs.          │
  └──────────────────────────────────────────────────────────┘

  ┌──────────────────────────────────────────────────────────┐
  │  Daily Summary CronJob (08:00 UTC)                        │
  │                                                            │
  │  Queries Prometheus for key stats, sends digest to         │
  │  Telegram with:                                            │
  │  - Cluster health summary                                  │
  │  - Top firing alerts                                       │
  │  - Resource utilization                                    │
  └──────────────────────────────────────────────────────────┘
```

---

## 11. Configuration Reference

### Terraform Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `enable_monitoring` | bool | `false` | Master toggle — deploys Prometheus, Grafana, Loki, Tempo, OTel |
| `prometheus_stack_version` | string | `"61.3.1"` | kube-prometheus-stack Helm chart version |
| `loki_version` | string | `"6.6.4"` | Loki Helm chart version |
| `tempo_version` | string | `"1.10.1"` | Tempo Helm chart version |
| `otel_collector_version` | string | `"0.96.0"` | OTel Collector Helm chart version |
| `enable_falco` | bool | `true` | Runtime security monitoring |
| `falco_version` | string | `"4.18.0"` | Falco Helm chart version |
| `telegram_bot_token` | string (sensitive) | `""` | For alert notifications |
| `telegram_chat_id` | string | `""` | Telegram chat target |
| `slack_webhook_url` | string (sensitive) | `""` | For Slack notifications |
| `enable_v3_operators` | bool | `false` | V3 operator-based deployment (future) |

### Kubernetes Resources Created

| Resource Type | Count | Namespace | Created By |
|---------------|-------|-----------|------------|
| Helm Release | 5 | monitoring | Terraform |
| ServiceMonitor | ~20 | various | Terraform + Helm charts |
| PodMonitor | 2 | zenith-staging, zenith-shared | CNPG operator |
| ConfigMap (dashboards) | 16 | monitoring | Terraform |
| NetworkPolicy | 6 | monitoring, zenith-staging | Terraform |
| PrometheusRule | 3+ | monitoring | Helm chart + manual |
| Secret (AlertManager) | 1 | monitoring | Manual |

### Port Reference

| Service | Port | Protocol | Purpose |
|---------|------|----------|---------|
| Prometheus | 9090 | HTTP | Query API + UI |
| Grafana | 3000 | HTTP | Dashboard UI |
| AlertManager | 9093 | HTTP | Alert management UI |
| Loki | 3100 | HTTP | Log query + push API |
| Tempo | 3200 | HTTP | Trace query API |
| Tempo | 4317 | gRPC | OTLP trace ingest |
| OTel Collector | 4317 | gRPC | OTLP receiver |
| OTel Collector | 4318 | HTTP | OTLP receiver |
| OTel Collector | 14250 | gRPC | Jaeger receiver |
| OTel Collector | 9411 | HTTP | Zipkin receiver |
| APISIX | 9091 | HTTP | Prometheus metrics |
| CNPG | 9187 | HTTP | Postgres exporter |
| Falcosidekick | 2810 | HTTP | Prometheus metrics |
| Kyverno | 8000 | HTTP | Metrics |
| Velero | 8085 | HTTP | Metrics |
| cert-manager | 9402 | HTTP | Metrics |
| node-exporter | 9100 | HTTP | Node metrics |

---

## 12. Troubleshooting Guide

### "My dashboard shows No Data"

**Check 1: Is the datasource configured?**
```bash
# List Grafana datasources
kubectl exec -n monitoring deploy/kube-prometheus-stack-grafana -- \
  curl -s http://localhost:3000/api/datasources | jq '.[].name'
```

Expected: `Prometheus`, `Alertmanager`, `Loki`, `Tempo`

**Check 2: Does the datasource UID match?**

Open the dashboard JSON and look for `"datasource": {"uid": "loki"}`.
Then verify the Loki datasource has `uid: "loki"` (not auto-generated).

**Check 3: Is the target UP in Prometheus?**
```bash
# Check all targets
kubectl exec -n monitoring prometheus-kube-prometheus-stack-prometheus-0 -- \
  wget -qO- http://localhost:9090/api/v1/targets | python3 -c "
import json,sys
for t in json.load(sys.stdin)['data']['activeTargets']:
    if t['health']=='down':
        print(f\"DOWN: {t['labels'].get('job')} - {t.get('lastError')}\")"
```

**Check 4: Does the service actually expose metrics?**
```bash
# Test from within the cluster
kubectl run test-metrics --rm -it --restart=Never --image=busybox -- \
  wget -qO- --timeout=5 http://<service-ip>:<port>/metrics | head -10
```

**Check 5: Is there a NetworkPolicy blocking?**
```bash
kubectl get networkpolicy -n <target-namespace>
```

### "TargetDown alert is firing"

1. Find which target is down (see Check 3 above)
2. Common causes:
   - **NetworkPolicy blocking** — add an allow rule
   - **Service label mismatch** — ServiceMonitor selector doesn't match Service labels
   - **Wrong port name** — ServiceMonitor references port name that doesn't exist
   - **Pod not ready** — the pod itself is crashing

### "PrometheusRuleFailures is firing"

```bash
# Find which rules are failing
kubectl exec -n monitoring prometheus-kube-prometheus-stack-prometheus-0 -- \
  wget -qO- http://localhost:9090/api/v1/rules | python3 -c "
import json,sys
for g in json.load(sys.stdin)['data']['groups']:
    for r in g['rules']:
        if r.get('lastError'):
            print(f\"{r['name']}: {r['lastError'][:200]}\")"
```

Common causes:
- **Duplicate series** — two ServiceMonitors scraping the same endpoint (like the k3s kubelet issue)
- **Missing metric** — rule references a metric that doesn't exist yet
- **Syntax error** — invalid PromQL in PrometheusRule CRD

### "Falco events not showing in Grafana"

1. Check Falcosidekick is sending successfully:
```bash
kubectl logs -n falco -l app.kubernetes.io/name=falcosidekick --tail=5 | grep Loki
# Should show: "Loki - POST OK (204)"
```

2. Check the correct Loki query:
```
{source="syscall"} | json
```
NOT `{source="falco"}` — Falco events use `source=syscall`.

3. Check NetworkPolicy allows falco → loki on port 3100.

### "OTel traces not showing in Tempo"

1. Check OTel Collector is running:
```bash
kubectl get pods -n monitoring -l app.kubernetes.io/name=opentelemetry-collector
```

2. Check the pipeline config:
```bash
kubectl get cm -n monitoring -l app.kubernetes.io/name=opentelemetry-collector -o yaml | grep -A5 "traces:"
```
The `traces` pipeline must have `otlp` in exporters (not just `debug`).

3. Check your application is sending traces to the right endpoint:
```
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector-opentelemetry-collector.monitoring.svc.cluster.local:4317
```

4. Check NetworkPolicy allows your namespace → monitoring:4317.

### Useful Prometheus Queries

```promql
# How many active series total
prometheus_tsdb_head_series

# Top 10 jobs by series count
topk(10, count by (job) ({__name__!=""}))

# Memory usage by pod
container_memory_working_set_bytes{namespace="monitoring"} / 1024 / 1024

# Disk usage by PVC
kubelet_volume_stats_used_bytes / kubelet_volume_stats_capacity_bytes * 100

# All firing alerts
ALERTS{alertstate="firing"}
```

---

## 13. Remaining Work

### Must Do (for 10/10 score)

#### 1. OTel SDK Instrumentation in zenith-api (2-3 hours)

The traces pipeline is ready (OTel Collector → Tempo → Grafana). But no application
sends traces yet. Need to add OpenTelemetry SDK to the Go API:

```go
// What needs to be added to zenith-api:

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// 1. Initialize tracer provider (in main.go)
func initTracer() (*trace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("otel-collector-opentelemetry-collector.monitoring:4317"),
        otlptracegrpc.WithInsecure(),
    )
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.ServiceNameKey.String("zenith-api"),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}

// 2. Wrap HTTP handler (in router setup)
handler := otelhttp.NewHandler(router, "zenith-api")

// 3. Wrap database calls
// Use otelsql or manual spans for DB queries
```

**Environment variable needed in the API deployment:**
```yaml
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "otel-collector-opentelemetry-collector.monitoring.svc.cluster.local:4317"
  - name: OTEL_SERVICE_NAME
    value: "zenith-api"
```

#### 2. AlertManager → Telegram Routing (30 min)

Currently, only Falcosidekick sends to Telegram. Prometheus alerts go nowhere
(except Slack, if configured). Need to apply the AlertManager config:

```bash
kubectl apply -f infra/k8s/monitoring/alertmanager-config.yaml
```

#### 3. Custom Zenith Alert Rules (1-2 hours)

The alert rules exist in `infra/helm/monitoring/templates/alerting-rules.yaml`
but need to be deployed via the monitoring Helm chart or applied manually:

```bash
kubectl apply -f infra/k8s/monitoring/alertmanager-config.yaml
```

The rules cover: API down, high error rate, build failures, DB issues, business metrics.

#### 4. Blackbox Exporter (1 hour)

External probe that checks if services respond from outside the cluster:

```yaml
# Would add to observability.tf:
resource "helm_release" "blackbox_exporter" {
  name  = "blackbox-exporter"
  chart = "prometheus-blackbox-exporter"
  # ...
}
```

Probes: `https://api.stage.freezenith.com/health`,
`https://app.stage.freezenith.com`, `https://hub.stage.freezenith.com`.

---

## 14. V3 Operator-Based Architecture

A future version of the stack is pre-configured but disabled (`enable_v3_operators = false`).

**File:** `infra/terraform/modules/k8s-platform/observability-operators.tf`

Key differences from V2:

| Aspect | V2 (Current) | V3 (Future) |
|--------|-------------|-------------|
| Loki | Helm chart, SingleBinary | Loki Operator + LokiStack CRD |
| Tempo | Helm chart, SingleBinary | Tempo Operator + TempoStack CRD |
| OTel | Helm chart, DaemonSet | OTel Operator + OpenTelemetryCollector CRD |
| Storage | Local filesystem (hcloud-volumes) | Hetzner S3 Object Storage |
| Scaling | Manual replica count | Operator auto-scaling |
| Auto-instrumentation | None | OTel Instrumentation CRD (Go/Node.js) |
| Sampling | 100% (no sampling) | 10% parent-based trace ID ratio |

V3 adds auto-instrumentation — annotate a pod with
`instrumentation.opentelemetry.io/inject-go: "true"` and the operator
automatically injects the OTel agent. No code changes needed.

---

## 15. Appendix: Full Code Reference

### File Map

| File | Lines | Components |
|------|-------|------------|
| `infra/terraform/modules/k8s-platform/observability.tf` | ~810 | Prometheus, Grafana, Loki, Tempo, OTel, ServiceMonitors, Dashboards |
| `infra/terraform/modules/k8s-platform/observability-operators.tf` | ~364 | V3 Loki/Tempo/OTel operators (disabled) |
| `infra/terraform/modules/k8s-platform/security.tf` | ~627 | Falco, custom rules, Falcosidekick outputs |
| `infra/terraform/modules/k8s-platform/gateway.tf` | ~250 | APISIX metrics/ServiceMonitor config |
| `infra/terraform/modules/k8s-platform/cilium.tf` | ~200 | Hubble, Cilium agent/operator metrics |
| `infra/terraform/modules/k8s-platform/registry.tf` | ~150 | Harbor metrics/ServiceMonitor |
| `infra/terraform/modules/k8s-platform/gitops.tf` | ~300 | ArgoCD metrics/ServiceMonitor |
| `infra/terraform/modules/k8s-platform/network-policies.tf` | ~400 | All monitoring-related NetworkPolicies |
| `infra/terraform/modules/k8s-platform/variables.tf` | ~250 | All monitoring variables |
| `infra/helm/monitoring/dashboards/*.json` | ~9228 | 16 custom Grafana dashboards |
| `infra/helm/monitoring/templates/alerting-rules.yaml` | ~150 | Custom PrometheusRule alerts |
| `infra/helm/monitoring/templates/servicemonitors.yaml` | ~50 | Template for Helm-based ServiceMonitors |
| `infra/k8s/monitoring/alertmanager-config.yaml` | ~200 | AlertManager routing + Zenith alert rules |
| `infra/k8s/monitoring/grafana-dashboard-platform.yaml` | ~200 | Platform health dashboard |
| `infra/k8s/monitoring/grafana-dashboard-business.yaml` | ~250 | Business metrics dashboard |
| `infra/k8s/monitoring/daily-summary-cronjob.yaml` | ~50 | Daily Telegram summary |

### Lessons Learned (Mistakes We Made)

1. **Datasource UID mismatch** — Auto-generated UIDs broke all dashboards. Always set explicit UIDs.
2. **APISIX metrics config** — Tried 3 different config formats before finding `apisix.prometheus.enabled`.
3. **Loki ServiceMonitor too broad** — Selected gateway/headless services that return 404. Use `matchExpressions` to exclude.
4. **k3s kubelet duplicate series** — Old helm release left a stale service. Deleting it fixed the duplicate.
5. **Falco labels are capitalized** — `priority="Notice"` not `"notice"`. Dashboard queries must match.
6. **Falco source label is "syscall"** — Not "falco". This is the Falco event source type.
7. **OTel traces pipeline had only debug exporter** — Traces went to stdout, not Tempo. Must add `otlp` exporter.
8. **NetworkPolicies block telemetry** — Every monitoring connection across namespaces needs an explicit allow.
9. **Grafana CrashLoopBackOff from invalid plugin** — Non-existent plugin name causes crash. Check plugin registry first.
10. **Tempo permission denied on hcloud-volumes** — `fsGroup` must be pod-level, not container-level.
11. **Helm `set` blocks vs `yamlencode`** — Use `values = [yamlencode({...})]` for complex config, not dozens of `set` blocks.
12. **Stale kubelet service from old Helm release** — `zenith-monitoring-kube-pro-kubelet` coexisted with `kube-prometheus-stack-kubelet`, causing duplicate scrapes.

---

*This document covers the complete Zenith observability stack as of 2026-03-19.
For updates, check the git log on the relevant terraform and helm files.*
