# Observability: Monitoring, Logging, Tracing

> **Zenith V2 Platform Architecture -- Full-Stack Observability**
>
> This document covers how we see what's happening inside the Zenith platform:
> metrics (numbers), logs (text), traces (request journeys), and network flows.

> **Status:** Design Complete, Implementation Pending
> **Last Updated:** 2026-02-25
> **Author:** Babak + Claude (Platform Architecture Session)
> **Decision:** D13 -- Full-Stack Observability

---

## Table of Contents

1. [Why Observability Matters](#why-observability-matters)
2. [Architecture Overview](#architecture-overview)
3. [The Three Pillars (Plus One)](#the-three-pillars-plus-one)
4. [Metrics: Prometheus](#metrics-prometheus)
   - [What Gets Scraped](#what-gets-scraped)
   - [ServiceMonitor Examples](#servicemonitor-examples)
   - [Custom Metrics from zenith-api](#custom-metrics-from-zenith-api)
   - [Storage and Retention](#storage-and-retention)
5. [Logs: Loki + Promtail](#logs-loki--promtail)
   - [Log Pipeline](#log-pipeline)
   - [Label Strategy](#label-strategy)
   - [Kubernetes Audit Log Integration](#kubernetes-audit-log-integration)
   - [Useful LogQL Queries](#useful-logql-queries)
6. [Traces: Tempo + OpenTelemetry](#traces-tempo--opentelemetry)
   - [How Distributed Tracing Works](#how-distributed-tracing-works)
   - [OpenTelemetry Collector Configuration](#opentelemetry-collector-configuration)
   - [APISIX Trace Propagation](#apisix-trace-propagation)
   - [Go SDK Integration for zenith-api](#go-sdk-integration-for-zenith-api)
   - [Trace-to-Log Correlation](#trace-to-log-correlation)
7. [Network Observability: Hubble](#network-observability-hubble)
   - [Service Map](#service-map)
   - [Flow Visibility](#flow-visibility)
   - [DNS Monitoring](#dns-monitoring)
   - [Policy Verification](#policy-verification)
8. [Dashboards: Grafana](#dashboards-grafana)
   - [Dashboard Catalog](#dashboard-catalog)
   - [Data Source Configuration](#data-source-configuration)
9. [Alerting: Alertmanager](#alerting-alertmanager)
   - [Alert Rules by Severity](#alert-rules-by-severity)
   - [Routing Configuration](#routing-configuration)
   - [Silences and Inhibitions](#silences-and-inhibitions)
10. [Per-Customer Observability](#per-customer-observability)
    - [Tenant-Aware Queries](#tenant-aware-queries)
    - [Customer-Visible Metrics](#customer-visible-metrics)
11. [How to Run](#how-to-run)
    - [Accessing Grafana](#accessing-grafana)
    - [Accessing Hubble UI](#accessing-hubble-ui)
    - [Common Debugging Workflows](#common-debugging-workflows)
12. [Component Summary Table](#component-summary-table)

---

## Why Observability Matters

Running a multi-tenant PaaS is fundamentally different from running a single application.
When 50 customers share the same Kubernetes cluster, every problem becomes harder to
diagnose:

- **Which customer caused the CPU spike?** Without per-namespace metrics, you are blind.
- **Why did that API request fail?** Without distributed tracing, you are guessing which
  service in the chain broke.
- **Who deleted that ConfigMap at 3am?** Without Kubernetes audit logs, you will never know.
- **Why is one customer unable to reach their database?** Without network flow visibility,
  you cannot tell whether a Cilium policy is dropping the traffic.

Observability is not optional for a platform like Zenith. It is the difference between
operating the platform proactively (fixing problems before customers notice) and operating
reactively (scrambling after customer complaints).

The design principle is: **every component emits telemetry, and all telemetry flows to a
single pane of glass (Grafana).** There is no separate tool you need to check. Metrics,
logs, traces, and network flows all land in Grafana, cross-linked so you can jump from an
alert to a dashboard to the relevant logs to the specific trace that failed.

---

## Architecture Overview

This is the complete data flow for all observability signals in Zenith V2:

```
+--------------------------------------------------------------------+
|                        Kubernetes Cluster                           |
|                                                                     |
|  +-----------+  +-----------+  +-----------+  +------------------+  |
|  | Customer  |  | Customer  |  | Platform  |  | Infrastructure   |  |
|  | Pod (A)   |  | Pod (B)   |  | Services  |  | (APISIX, CNPG,  |  |
|  |           |  |           |  | (zenith-  |  |  Keycloak, etc)  |  |
|  |           |  |           |  |  api)     |  |                  |  |
|  +-----+-----+  +-----+-----+  +-----+-----+  +--------+---------+  |
|        |              |              |                   |           |
|        |   stdout/    |   /metrics   |   OTLP traces    |           |
|        |   stderr     |   endpoint   |   (gRPC:4317)    |           |
|        v              v              v                   |           |
|  +----------+   +----------+   +------------------+     |           |
|  | Promtail |   |Prometheus|   | OTel Collector   |     |           |
|  | DaemonSet|   |  server  |   | DaemonSet        |     |           |
|  +----+-----+   +----+-----+   +---+----+---------+     |           |
|       |              |              |    |               |           |
|       v              v              v    v               |           |
|  +--------+   +-----------+   +------+ +----------+     |           |
|  |  Loki  |   | Prometheus|   | Tempo| | Prom     |     |           |
|  |  (logs)|   | (metrics) |   |(trace| | (OTel    |     |           |
|  +---+----+   +-----+-----+   +--+---+ |  metrics)|    |           |
|      |              |             |     +----+-----+     |           |
|      |              |             |          |           |           |
|      +-------+------+------+-----+----------+           |           |
|              |                                           |           |
|              v                                           |           |
|        +-----------+                                     |           |
|        |  Grafana  | <--- Single pane of glass           |           |
|        +-----------+                                     |           |
|              |                                           |           |
|              v                                           |           |
|       +--------------+                                   |           |
|       | Alertmanager |                                   |           |
|       +------+-------+                                   |           |
|              |                                           |           |
+--------------------------------------------------------------------+
               |                                           |
               v                                           v
        +------+-------+                           +-------+-------+
        | Slack /      |                           | Hubble        |
        | PagerDuty    |                           | (Cilium eBPF) |
        +--------------+                           |    |          |
                                                   |    v          |
                                                   | Hubble UI    |
                                                   +---------------+

K8s API Server
     |
     +-- audit.log --> Promtail --> Loki --> Grafana
```

Every signal (metrics, logs, traces, network flows) converges in Grafana. This is
intentional -- operators should never need to switch between five different tools to
debug one incident.

---

## The Three Pillars (Plus One)

Before diving into components, here is the mental model:

| Pillar | What It Captures | Think of It As | Tool |
|--------|-----------------|----------------|------|
| **Metrics** | Numbers over time (CPU, request count, latency) | A speedometer on your car | Prometheus |
| **Logs** | Text events ("user login failed", "pod OOMKilled") | A black box recorder | Loki |
| **Traces** | The journey of a single request through services | GPS tracking for a package | Tempo |
| **Network Flows** | L3/L4/L7 packet information (who talks to whom) | A phone call log | Hubble |

Each pillar answers different questions:

- **Metrics** answer "how much?" and "how fast?" -- request rate, error rate, latency percentiles.
- **Logs** answer "what happened?" -- the specific error message, the stack trace, the audit event.
- **Traces** answer "where did it break?" -- which service in a multi-hop request chain caused the failure.
- **Network flows** answer "who is talking to whom?" -- is traffic being dropped, is DNS resolving, is a policy blocking access.

The magic happens when you cross-link them. A Prometheus alert fires because error rate
is high. You click through to Grafana, filter logs by the same time window, find the
error message, click the trace ID embedded in the log line, and see exactly which
downstream call failed. That workflow -- alert to metric to log to trace -- is what
full-stack observability enables.

---

## Metrics: Prometheus

### What Gets Scraped

Prometheus discovers scrape targets automatically through Kubernetes service discovery
and the ServiceMonitor CRD (from the Prometheus Operator). Every component in the
Zenith stack exposes a `/metrics` endpoint:

| Component | Endpoint | Key Metrics |
|-----------|----------|-------------|
| **zenith-api** | `:8080/metrics` | HTTP request latency, gRPC call count, provisioning duration |
| **APISIX** | `:9091/apisix/prometheus/metrics` | Request count per route, upstream latency, bandwidth |
| **Keycloak** | `:8080/metrics` | Login attempts, token issues, realm user count |
| **CNPG** | `:9187/metrics` | Query latency, active connections, WAL lag, replication delay |
| **Cilium Agent** | `:9962/metrics` | Policy enforcement, drops, forwards, endpoint count |
| **Hubble** | `:9965/metrics` | Flow count, DNS latency, HTTP request/response |
| **cert-manager** | `:9402/metrics` | Certificate ready status, expiry time, renewal failures |
| **ArgoCD** | `:8082/metrics` | Sync status, application health, reconciliation duration |
| **Temporal** | `:9090/metrics` | Workflow execution count, latency, failure rate |
| **Node Exporter** | `:9100/metrics` | CPU, memory, disk, network (per node) |
| **kube-state-metrics** | `:8080/metrics` | Pod status, deployment replicas, resource quotas |

### ServiceMonitor Examples

ServiceMonitors tell Prometheus which services to scrape. They are namespace-scoped
and label-matched:

```yaml
# ServiceMonitor for zenith-api
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: zenith-api
  namespace: monitoring
  labels:
    release: prometheus    # Must match Prometheus operator selector
spec:
  namespaceSelector:
    matchNames:
      - zenith-platform
  selector:
    matchLabels:
      app: zenith-api
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
      scrapeTimeout: 10s
```

```yaml
# ServiceMonitor for APISIX
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: apisix
  namespace: monitoring
  labels:
    release: prometheus
spec:
  namespaceSelector:
    matchNames:
      - apisix
  selector:
    matchLabels:
      app.kubernetes.io/name: apisix
  endpoints:
    - port: prometheus
      path: /apisix/prometheus/metrics
      interval: 15s
```

```yaml
# ServiceMonitor for CNPG clusters (all customer namespaces)
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: cnpg-clusters
  namespace: monitoring
  labels:
    release: prometheus
spec:
  namespaceSelector:
    any: true              # Scrape across ALL namespaces
  selector:
    matchLabels:
      cnpg.io/cluster: ""  # Matches any CNPG cluster service
  endpoints:
    - port: metrics
      path: /metrics
      interval: 30s
```

### Custom Metrics from zenith-api

The zenith-api Go service exposes custom business metrics using the
`prometheus/client_golang` library:

```go
// Metric definitions in zenith-api
var (
    customerProvisionDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "zenith_customer_provision_duration_seconds",
            Help:    "Time to fully provision a new customer",
            Buckets: []float64{10, 30, 60, 120, 300, 600},
        },
        []string{"tier"},   // free, pro, business, enterprise
    )

    activeCustomers = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "zenith_active_customers_total",
            Help: "Number of active customers by tier",
        },
        []string{"tier"},
    )

    apiRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "zenith_api_request_duration_seconds",
            Help:    "API request latency by endpoint and method",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint", "status_code"},
    )

    workflowExecutions = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "zenith_workflow_executions_total",
            Help: "Temporal workflow executions by type and result",
        },
        []string{"workflow", "result"},   // result: success, failure, timeout
    )
)
```

**Useful PromQL queries for zenith-api:**

```promql
# API request rate (per second, by endpoint)
rate(zenith_api_request_duration_seconds_count[5m])

# P95 API latency by endpoint
histogram_quantile(0.95, rate(zenith_api_request_duration_seconds_bucket[5m]))

# Error rate (5xx responses)
sum(rate(zenith_api_request_duration_seconds_count{status_code=~"5.."}[5m]))
  /
sum(rate(zenith_api_request_duration_seconds_count[5m]))

# Average customer provisioning time by tier
histogram_quantile(0.5, rate(zenith_customer_provision_duration_seconds_bucket[1h]))

# Active customers by tier
zenith_active_customers_total
```

### Storage and Retention

Prometheus runs in the `monitoring` namespace with the following storage configuration:

```yaml
# Prometheus storage (via Prometheus Operator)
apiVersion: monitoring.coreos.com/v1
kind: Prometheus
metadata:
  name: prometheus
  namespace: monitoring
spec:
  retention: 15d                    # Keep 15 days of raw metrics
  retentionSize: 50GB              # Hard cap on disk usage
  storage:
    volumeClaimTemplate:
      spec:
        storageClassName: hcloud-volumes
        resources:
          requests:
            storage: 100Gi
  resources:
    requests:
      cpu: 500m
      memory: 2Gi
    limits:
      memory: 4Gi
```

For long-term storage (beyond 15 days), we plan to use Thanos or Mimir in a future
iteration. For V2 launch, 15 days of raw metrics is sufficient -- most operational
queries look at the last few hours to days.

---

## Logs: Loki + Promtail

### Log Pipeline

The log pipeline is straightforward:

```
Container stdout/stderr
        |
        v
/var/log/pods/<namespace>_<pod>_<uid>/<container>/0.log   (on each node)
        |
        v
   Promtail (DaemonSet, one per node)
   - Discovers pods via Kubernetes API
   - Attaches labels: namespace, pod, container, app
   - Streams to Loki
        |
        v
   Loki (StatefulSet)
   - Indexes labels (NOT log content)
   - Stores compressed log chunks on disk
   - Serves queries from Grafana
```

Loki is deliberately designed to be **label-indexed, not full-text-indexed**. This makes
it orders of magnitude cheaper to run than Elasticsearch. You query by labels first
(namespace, pod, app) and then optionally filter/grep within the matching log streams.

### Label Strategy

Labels are critical in Loki. Too many labels cause high cardinality (which kills
performance). Too few labels make queries slow (scanning too many streams). Our label
strategy:

| Label | Source | Example | Cardinality |
|-------|--------|---------|-------------|
| `namespace` | K8s metadata | `customer-acme` | ~100 (grows with customers) |
| `pod` | K8s metadata | `zenith-api-7f8b4c-x9kf2` | ~500 (high, but needed) |
| `container` | K8s metadata | `api` | ~20 |
| `app` | Pod label `app` | `zenith-api` | ~30 |
| `component` | Pod label `component` | `gateway`, `database` | ~10 |
| `node_name` | K8s metadata | `k3s-worker-01` | ~5 |

**Labels we intentionally avoid** (too high cardinality):

- `trace_id` -- extracted via LogQL parser at query time, not a label
- `user_id` -- same, extracted at query time
- `request_path` -- same
- `customer_id` -- we use `namespace` instead (1:1 mapping)

Promtail configuration for label attachment:

```yaml
# Promtail config (relevant section)
scrape_configs:
  - job_name: kubernetes-pods
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      # Keep namespace as label
      - source_labels: [__meta_kubernetes_namespace]
        target_label: namespace
      # Keep pod name
      - source_labels: [__meta_kubernetes_pod_name]
        target_label: pod
      # Keep container name
      - source_labels: [__meta_kubernetes_pod_container_name]
        target_label: container
      # Extract 'app' label from pod labels
      - source_labels: [__meta_kubernetes_pod_label_app]
        target_label: app
      # Extract 'component' label from pod labels
      - source_labels: [__meta_kubernetes_pod_label_component]
        target_label: component
      # Node name
      - source_labels: [__meta_kubernetes_pod_node_name]
        target_label: node_name
      # Drop pods without an 'app' label (noise reduction)
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: drop
        regex: ""
```

### Kubernetes Audit Log Integration

Kubernetes API audit logs capture every API call: who created/deleted/modified which
resource, when, and from where. This is critical for security (see doc 06) and for
debugging "who changed that?".

K3s writes audit logs to `/var/log/kubernetes/audit/audit.log` when configured with an
audit policy. Promtail picks these up via a separate scrape job:

```yaml
# Promtail scrape config for K8s audit logs
scrape_configs:
  - job_name: kubernetes-audit
    static_configs:
      - targets:
          - localhost
        labels:
          job: kubernetes-audit
          __path__: /var/log/kubernetes/audit/audit.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            verb: verb
            user: user.username
            resource: objectRef.resource
            namespace: objectRef.namespace
            name: objectRef.name
            timestamp: stageTimestamp
      - labels:
          level:
          verb:
          user:
      - timestamp:
          source: timestamp
          format: RFC3339Nano
```

### Useful LogQL Queries

LogQL is Loki's query language. It works in two modes: log queries (return log lines)
and metric queries (return numbers computed from logs).

**Basic log queries:**

```logql
# All logs from zenith-api
{app="zenith-api"}

# All logs from a specific customer namespace
{namespace="customer-acme"}

# Errors from APISIX
{app="apisix"} |= "error"

# All logs from CNPG pods containing "FATAL" or "ERROR"
{app=~"cnpg.*"} |~ "FATAL|ERROR"

# Keycloak login failures
{app="keycloak"} |= "LOGIN_ERROR"

# Exclude health check noise from zenith-api logs
{app="zenith-api"} != "/healthz" != "/readyz"
```

**Structured log queries (JSON parsing):**

```logql
# Parse JSON logs from zenith-api and filter by level
{app="zenith-api"} | json | level="error"

# Parse and filter by HTTP status code
{app="zenith-api"} | json | status >= 500

# Extract trace_id from structured logs (for trace correlation)
{app="zenith-api"} | json | trace_id != "" | line_format "{{.trace_id}} {{.msg}}"

# Find slow API requests (>2 seconds)
{app="zenith-api"} | json | duration > 2s

# Show customer provisioning events
{app="zenith-api"} | json | msg=~".*provision.*" | line_format "{{.timestamp}} [{{.customer}}] {{.msg}}"
```

**Metric queries (derive numbers from logs):**

```logql
# Error rate from zenith-api logs (errors per second over 5 minutes)
rate({app="zenith-api"} | json | level="error" [5m])

# Count of 5xx responses per minute by endpoint
sum by (endpoint) (
  count_over_time({app="zenith-api"} | json | status >= 500 [1m])
)

# Bytes of logs per namespace per hour (who is noisy?)
sum by (namespace) (bytes_over_time({namespace=~"customer-.*"}[1h]))

# Count of K8s audit events by verb and resource (what is being changed?)
sum by (verb, resource) (
  count_over_time({job="kubernetes-audit"} | json [1h])
)

# Login failure rate from Keycloak
sum(rate({app="keycloak"} |= "LOGIN_ERROR" [5m]))
```

**Loki storage configuration:**

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: loki
  namespace: monitoring
spec:
  # ...
  template:
    spec:
      containers:
        - name: loki
          args:
            - -config.file=/etc/loki/config.yaml
          resources:
            requests:
              cpu: 250m
              memory: 512Mi
            limits:
              memory: 1Gi
          volumeMounts:
            - name: data
              mountPath: /loki
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        storageClassName: hcloud-volumes
        resources:
          requests:
            storage: 50Gi
```

Loki retention is set to **30 days**. Logs older than 30 days are automatically
compacted and deleted. For audit compliance, K8s audit logs are retained for **90 days**
via a separate retention rule.

---

## Traces: Tempo + OpenTelemetry

### How Distributed Tracing Works

Distributed tracing solves a fundamental problem: when a single user request passes
through multiple services, how do you follow it?

Here is what happens when a customer hits the Zenith API:

```
1. Browser sends request to https://api.freezenith.com/v1/clusters
2. Cloudflare proxies to Hetzner
3. APISIX receives the request
   - APISIX generates a trace ID: abc123
   - APISIX adds header: traceparent: 00-abc123-span01-01
4. APISIX forwards to zenith-api
   - zenith-api reads the traceparent header
   - zenith-api creates a child span (span02, parent=span01)
5. zenith-api calls Keycloak to verify the JWT
   - Keycloak call gets span03 (parent=span02)
6. zenith-api queries CNPG PostgreSQL
   - DB query gets span04 (parent=span02)
7. zenith-api returns the response
   - All spans share trace ID abc123
```

When you look up trace `abc123` in Grafana Tempo, you see:

```
trace: abc123 (total: 245ms)
|
+-- [APISIX] GET /v1/clusters                           0ms - 245ms
    |
    +-- [zenith-api] handleListClusters                 12ms - 240ms
        |
        +-- [keycloak] POST /realms/acme/protocol/...   15ms - 45ms
        |
        +-- [postgres] SELECT * FROM clusters WHERE...  50ms - 230ms  <-- slow!
```

Now you can see that the database query took 180ms out of the total 245ms. Without
tracing, you would only know "the API was slow" but not why.

The standard used is **W3C Trace Context** (`traceparent` header). All components
propagate this header so traces are not broken at service boundaries.

### OpenTelemetry Collector Configuration

The OTel Collector runs as a DaemonSet (one per node) and acts as a central pipeline
for traces and optionally metrics:

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: otel-collector
  namespace: monitoring
spec:
  mode: daemonset
  config: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318

    processors:
      batch:
        timeout: 5s
        send_batch_size: 1000
      memory_limiter:
        check_interval: 1s
        limit_mib: 512
        spike_limit_mib: 128
      resource:
        attributes:
          - key: cluster
            value: zenith-prod
            action: upsert
      # Add namespace as a resource attribute for tenant filtering
      k8sattributes:
        auth_type: "serviceAccount"
        extract:
          metadata:
            - k8s.namespace.name
            - k8s.pod.name
            - k8s.deployment.name

    exporters:
      otlp/tempo:
        endpoint: tempo.monitoring.svc.cluster.local:4317
        tls:
          insecure: true
      prometheus:
        endpoint: 0.0.0.0:8889
        resource_to_telemetry_conversion:
          enabled: true

    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [memory_limiter, k8sattributes, resource, batch]
          exporters: [otlp/tempo]
        metrics:
          receivers: [otlp]
          processors: [memory_limiter, resource, batch]
          exporters: [prometheus]
```

Key design decisions:
- **DaemonSet mode**: Every node has a local collector, so pods send traces to localhost
  (low latency, no cross-node traffic for trace submission).
- **k8sattributes processor**: Automatically enriches every span with the Kubernetes
  namespace, pod name, and deployment name. This is how we filter traces by customer.
- **Batch processor**: Buffers spans for 5 seconds before flushing to Tempo, reducing
  write pressure.
- **Memory limiter**: Prevents the collector from OOMKilling under high trace volume.

### APISIX Trace Propagation

APISIX has a built-in `opentelemetry` plugin. When enabled globally, it generates a
root span for every request that passes through the gateway:

```json
{
  "plugins": {
    "opentelemetry": {
      "sampler": {
        "name": "parent_based_trace_id_ratio",
        "options": {
          "root": {
            "name": "trace_id_ratio",
            "options": {
              "fraction": 1.0
            }
          }
        }
      },
      "additional_attributes": [
        "apisix_route_name"
      ]
    }
  }
}
```

This means:
- Every API request gets a trace, always (fraction: 1.0 = 100% sampling).
- The route name is attached as a span attribute, so you can filter traces by APISIX route.
- APISIX injects the `traceparent` header into the upstream request, so backends
  automatically become child spans.

In a production environment with high traffic, you would reduce the sampling fraction
(e.g., 0.1 for 10% sampling). For Zenith V2 initial rollout, 100% sampling is fine
because traffic volume is moderate.

### Go SDK Integration for zenith-api

The zenith-api Go service uses the OpenTelemetry Go SDK to create spans, propagate
context, and emit traces to the OTel Collector:

```go
package observability

import (
    "context"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func InitTracer(ctx context.Context, serviceName string) (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("localhost:4317"),  // Local OTel DaemonSet
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
    )

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},   // W3C traceparent
        propagation.Baggage{},        // W3C baggage
    ))

    return tp, nil
}
```

Usage in an HTTP handler:

```go
func (h *ClusterHandler) ListClusters(w http.ResponseWriter, r *http.Request) {
    ctx, span := otel.Tracer("zenith-api").Start(r.Context(), "ListClusters")
    defer span.End()

    // The span is now active. Any downstream calls that accept this ctx
    // will automatically become child spans.

    clusters, err := h.clusterService.List(ctx)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        http.Error(w, "internal error", 500)
        return
    }

    span.SetAttributes(attribute.Int("cluster.count", len(clusters)))
    json.NewEncoder(w).Encode(clusters)
}
```

### Trace-to-Log Correlation

The key to powerful debugging is linking traces to logs. When zenith-api writes a log
line, it includes the trace ID:

```go
// In the structured logger (zerolog)
logger := zerolog.Ctx(ctx).With().
    Str("trace_id", span.SpanContext().TraceID().String()).
    Str("span_id", span.SpanContext().SpanID().String()).
    Logger()

logger.Info().Msg("listing clusters for customer")
```

This produces a log line like:

```json
{
  "level": "info",
  "trace_id": "abc123def456...",
  "span_id": "789ghi...",
  "msg": "listing clusters for customer",
  "timestamp": "2026-02-25T14:30:00Z"
}
```

In Grafana, you configure a **derived field** in the Loki data source that links
`trace_id` to Tempo:

```yaml
# Grafana Loki data source config (provisioned)
datasources:
  - name: Loki
    type: loki
    url: http://loki.monitoring.svc.cluster.local:3100
    jsonData:
      derivedFields:
        - name: TraceID
          datasourceUid: tempo
          matcherRegex: '"trace_id":"([a-f0-9]+)"'
          url: '$${__value.raw}'
```

Now when you view logs in Grafana Explore, every log line with a trace_id shows a
clickable link that opens the full trace in Tempo. This is the "log to trace" jump.

The reverse ("trace to log") is also configured: Tempo is set up to link spans back to
Loki queries filtered by the trace ID.

---

## Network Observability: Hubble

Hubble is the observability layer built into Cilium. Because Cilium replaces kube-proxy
and implements networking via eBPF programs in the kernel, it can see every packet
flowing through the cluster without any instrumentation or sidecars.

### Service Map

Hubble automatically builds a real-time service dependency map by observing L3/L4/L7
traffic. In the Hubble UI, you can see:

```
                   +----> keycloak (HTTP 200, 12ms avg)
                   |
apisix ---HTTP---> zenith-api ---TCP---> cnpg-postgres (5432)
                   |
                   +----> temporal (gRPC, 8ms avg)
                   |
                   +----> loki (HTTP 200, push logs)
```

This map is generated automatically -- no configuration needed. It is invaluable for:
- Understanding which services depend on which (did someone add an unexpected dependency?)
- Spotting traffic patterns (is one customer generating 10x the traffic of others?)
- Verifying network policies (is the traffic that should be blocked actually blocked?)

### Flow Visibility

Hubble records individual network flows with rich metadata:

```bash
# Watch all flows in real time
hubble observe --namespace customer-acme

# Watch dropped packets (policy denials)
hubble observe --verdict DROPPED

# Watch HTTP flows to the API
hubble observe --namespace zenith-platform --protocol HTTP --to-pod zenith-api

# Watch DNS queries from a customer namespace
hubble observe --namespace customer-acme --type l7 --protocol DNS

# Export flows as JSON for analysis
hubble observe --namespace customer-acme -o json
```

Example Hubble flow output:

```
TIMESTAMP             SOURCE                      DESTINATION                 TYPE     VERDICT    SUMMARY
Feb 25 14:30:01.234   customer-acme/web-7f8b4c    zenith-platform/zenith-api  L7/HTTP  FORWARDED  GET /v1/clusters => 200
Feb 25 14:30:01.567   customer-acme/web-7f8b4c    customer-beta/api-6d9a3b    L4/TCP   DROPPED    SYN (policy denied)
Feb 25 14:30:02.890   zenith-platform/zenith-api  cnpg/acme-db-1              L4/TCP   FORWARDED  5432
```

The second line shows a cross-namespace access attempt that was denied by Cilium
NetworkPolicy -- exactly what we want for tenant isolation.

### DNS Monitoring

Hubble captures DNS queries and responses at the kernel level. This is useful for
debugging "name resolution failures" which are one of the most common Kubernetes issues:

```bash
# Watch all DNS queries from a namespace
hubble observe --namespace customer-acme --type l7 --protocol DNS

# Output:
# customer-acme/web-7f8b4c -> kube-system/coredns  DNS Query A zenith-api.zenith-platform.svc.cluster.local
# kube-system/coredns -> customer-acme/web-7f8b4c  DNS Response A 10.43.0.15 TTL:30
```

Hubble DNS metrics are also exported to Prometheus, so you can build dashboards for:
- DNS query latency by namespace
- DNS NXDOMAIN rate (broken service references)
- DNS query volume by pod (noisy neighbors)

### Policy Verification

After applying a Cilium NetworkPolicy, you can verify it works using Hubble:

```bash
# Apply a policy that blocks customer-acme from accessing customer-beta
kubectl apply -f network-policy-tenant-isolation.yaml

# Verify: attempt traffic and watch Hubble
hubble observe --from-namespace customer-acme --to-namespace customer-beta

# Expected output: DROPPED verdict on all flows
```

Hubble metrics for policy verification are available in Prometheus:

```promql
# Dropped packets by source and destination namespace
sum by (source_namespace, destination_namespace) (
  rate(hubble_drop_total[5m])
)

# Forwarded vs dropped ratio for a specific namespace
sum(rate(hubble_flows_processed_total{namespace="customer-acme", verdict="FORWARDED"}[5m]))
/
sum(rate(hubble_flows_processed_total{namespace="customer-acme"}[5m]))
```

---

## Dashboards: Grafana

### Dashboard Catalog

Grafana is the single pane of glass. All data sources (Prometheus, Loki, Tempo, Hubble
metrics) are configured as Grafana data sources, and dashboards query across them.

| Dashboard | Data Source | Description |
|-----------|------------|-------------|
| **Platform Overview** | Prometheus | Node CPU/memory/disk, total pod count, API latency p50/p95/p99, cluster health summary |
| **Per-Customer Resources** | Prometheus | CPU/memory usage per namespace, request count, error rate, resource quota usage |
| **APISIX Gateway** | Prometheus | Request rate by route, upstream latency histogram, 4xx/5xx error breakdown, bandwidth |
| **CNPG Database Health** | Prometheus | Query latency, active connections, WAL lag, replication delay, disk usage, backup status |
| **Keycloak Identity** | Prometheus + Loki | Login success/failure rate by realm, token issuance rate, active sessions, error logs |
| **Cilium Network** | Prometheus (Hubble) | Packet drops by policy, forwarded flows, endpoint count, DNS resolution latency |
| **Backup Health** | Prometheus | Last successful backup timestamp, backup duration, S3 bucket size, WAL archive lag |
| **ArgoCD Deployments** | Prometheus | Sync status per application, reconciliation duration, out-of-sync alerts |
| **Temporal Workflows** | Prometheus | Workflow execution count/duration by type, failure rate, queue depth |
| **cert-manager TLS** | Prometheus | Certificate expiry dates, renewal success/failure, ACME challenge duration |
| **Log Explorer** | Loki | Free-form log search with namespace/app/level filters, powered by LogQL |
| **Trace Explorer** | Tempo | Search traces by service, duration, status; drill into span waterfall |

### Data Source Configuration

All data sources are provisioned via Grafana's file-based provisioning (mounted as a
ConfigMap):

```yaml
# grafana-datasources.yaml (ConfigMap)
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-datasources
  namespace: monitoring
data:
  datasources.yaml: |
    apiVersion: 1
    datasources:
      - name: Prometheus
        type: prometheus
        access: proxy
        url: http://prometheus-server.monitoring.svc.cluster.local:9090
        isDefault: true
        jsonData:
          timeInterval: 15s

      - name: Loki
        type: loki
        access: proxy
        url: http://loki.monitoring.svc.cluster.local:3100
        jsonData:
          derivedFields:
            - name: TraceID
              datasourceUid: tempo
              matcherRegex: '"trace_id":"([a-f0-9]+)"'
              url: '$${__value.raw}'

      - name: Tempo
        type: tempo
        access: proxy
        url: http://tempo.monitoring.svc.cluster.local:3200
        uid: tempo
        jsonData:
          tracesToLogsV2:
            datasourceUid: loki
            filterByTraceID: true
          nodeGraph:
            enabled: true
          serviceMap:
            datasourceUid: prometheus

      - name: Alertmanager
        type: alertmanager
        access: proxy
        url: http://alertmanager.monitoring.svc.cluster.local:9093
        jsonData:
          implementation: prometheus
```

The cross-linking between data sources is what enables the seamless "jump from metric to
log to trace" workflow described earlier:

- **Loki -> Tempo**: derived field on `trace_id` links log lines to traces
- **Tempo -> Loki**: `tracesToLogsV2` links trace spans to log queries
- **Tempo -> Prometheus**: `serviceMap` builds a service graph from trace data + metrics
- **Prometheus -> Grafana alerts -> Alertmanager**: alert rules fire and route

---

## Alerting: Alertmanager

### Alert Rules by Severity

Alert rules are defined as PrometheusRule CRDs and evaluated by the Prometheus server.
When a rule fires, the alert is sent to Alertmanager for routing.

**Critical alerts** (immediate human attention required):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: critical-alerts
  namespace: monitoring
spec:
  groups:
    - name: critical
      rules:
        - alert: NodeUnreachable
          expr: up{job="node-exporter"} == 0
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "Node {{ $labels.instance }} is unreachable"
            description: "Node has been down for more than 2 minutes."

        - alert: CNPGPrimaryDown
          expr: cnpg_pg_replication_is_replica == 1 AND on(cluster) count(cnpg_pg_replication_is_replica) == 1
          for: 1m
          labels:
            severity: critical
          annotations:
            summary: "CNPG cluster {{ $labels.cluster }} has no primary"

        - alert: KeycloakDown
          expr: up{job="keycloak"} == 0
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "Keycloak is unreachable"
            description: "All authentication will fail. Immediate attention required."

        - alert: CertificateExpiringSoon
          expr: certmanager_certificate_expiration_timestamp_seconds - time() < 7 * 24 * 3600
          for: 1h
          labels:
            severity: critical
          annotations:
            summary: "Certificate {{ $labels.name }} expires in less than 7 days"

        - alert: APIHighErrorRate
          expr: |
            sum(rate(zenith_api_request_duration_seconds_count{status_code=~"5.."}[5m]))
            / sum(rate(zenith_api_request_duration_seconds_count[5m])) > 0.05
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "zenith-api error rate exceeds 5%"
```

**Warning alerts** (investigate within hours):

```yaml
        - alert: PodCrashLoopBackOff
          expr: rate(kube_pod_container_status_restarts_total[15m]) * 60 * 15 > 3
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "Pod {{ $labels.namespace }}/{{ $labels.pod }} is crash-looping"

        - alert: HighCPUUsage
          expr: |
            100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 85
          for: 10m
          labels:
            severity: warning
          annotations:
            summary: "Node {{ $labels.instance }} CPU usage > 85% for 10 minutes"

        - alert: HighMemoryUsage
          expr: |
            (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 > 90
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "Node {{ $labels.instance }} memory usage > 90%"

        - alert: APISIX5xxSpike
          expr: |
            sum(rate(apisix_http_status{code=~"5.."}[5m])) > 1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "APISIX returning elevated 5xx errors"

        - alert: FalcoAlert
          expr: rate(falco_events_total{priority=~"Critical|Error"}[5m]) > 0
          for: 1m
          labels:
            severity: warning
          annotations:
            summary: "Falco detected suspicious runtime activity"

        - alert: CNPGHighReplicationLag
          expr: cnpg_pg_replication_lag > 30
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "CNPG cluster {{ $labels.cluster }} replication lag > 30s"
```

**Info alerts** (no action needed, for awareness):

```yaml
        - alert: NewCustomerProvisioned
          expr: increase(zenith_workflow_executions_total{workflow="provision-customer", result="success"}[10m]) > 0
          labels:
            severity: info
          annotations:
            summary: "New customer provisioned successfully"

        - alert: BackupCompleted
          expr: increase(cnpg_pg_backup_total{status="completed"}[1h]) > 0
          labels:
            severity: info
          annotations:
            summary: "Database backup completed for {{ $labels.cluster }}"

        - alert: ArgoCDOutOfSync
          expr: argocd_app_info{sync_status!="Synced"} == 1
          for: 10m
          labels:
            severity: info
          annotations:
            summary: "ArgoCD app {{ $labels.name }} is out of sync for 10+ minutes"
```

### Routing Configuration

Alertmanager routes alerts to different channels based on severity:

```yaml
# alertmanager-config.yaml
apiVersion: v1
kind: Secret
metadata:
  name: alertmanager-config
  namespace: monitoring
stringData:
  alertmanager.yml: |
    global:
      resolve_timeout: 5m
      slack_api_url: 'https://hooks.slack.com/services/T00000/B00000/XXXXXXX'
      pagerduty_url: 'https://events.pagerduty.com/v2/enqueue'

    route:
      receiver: slack-default
      group_by: ['alertname', 'namespace']
      group_wait: 30s
      group_interval: 5m
      repeat_interval: 4h
      routes:
        # Critical alerts go to PagerDuty AND Slack
        - match:
            severity: critical
          receiver: pagerduty-critical
          continue: true    # Also send to next matching route
        - match:
            severity: critical
          receiver: slack-critical

        # Warning alerts go to Slack #alerts-warning channel
        - match:
            severity: warning
          receiver: slack-warning

        # Info alerts go to Slack #alerts-info channel
        - match:
            severity: info
          receiver: slack-info

    receivers:
      - name: slack-default
        slack_configs:
          - channel: '#alerts'
            title: '[{{ .Status | toUpper }}] {{ .CommonLabels.alertname }}'
            text: '{{ .CommonAnnotations.summary }}'

      - name: pagerduty-critical
        pagerduty_configs:
          - service_key: '<pagerduty-integration-key>'
            severity: critical
            description: '{{ .CommonAnnotations.summary }}'

      - name: slack-critical
        slack_configs:
          - channel: '#alerts-critical'
            color: 'danger'
            title: 'CRITICAL: {{ .CommonLabels.alertname }}'
            text: '{{ .CommonAnnotations.description }}'

      - name: slack-warning
        slack_configs:
          - channel: '#alerts-warning'
            color: 'warning'
            title: 'WARNING: {{ .CommonLabels.alertname }}'
            text: '{{ .CommonAnnotations.summary }}'

      - name: slack-info
        slack_configs:
          - channel: '#alerts-info'
            color: 'good'
            title: 'INFO: {{ .CommonLabels.alertname }}'
            text: '{{ .CommonAnnotations.summary }}'
```

### Silences and Inhibitions

**Silences** temporarily suppress alerts during planned maintenance:

```bash
# Silence all alerts for node k3s-worker-03 during maintenance window
amtool silence add \
  --alertmanager.url=http://alertmanager:9093 \
  --comment="Scheduled maintenance on worker-03" \
  --duration=2h \
  instance="k3s-worker-03"
```

**Inhibitions** prevent redundant alerts. If a node is down, you do not want separate
alerts for every pod on that node:

```yaml
# In alertmanager.yml
inhibit_rules:
  # If node is unreachable, suppress pod-level alerts on that node
  - source_match:
      alertname: NodeUnreachable
    target_match_re:
      alertname: PodCrashLoopBackOff|HighCPUUsage|HighMemoryUsage
    equal: ['instance']

  # If CNPG primary is down, suppress replication lag warnings
  - source_match:
      alertname: CNPGPrimaryDown
    target_match:
      alertname: CNPGHighReplicationLag
    equal: ['cluster']
```

---

## Per-Customer Observability

### Tenant-Aware Queries

Because each customer gets their own Kubernetes namespace (e.g., `customer-acme`),
the namespace label is the natural tenant identifier across all observability pillars.

**Metrics (PromQL) -- filter by customer:**

```promql
# CPU usage for customer-acme
sum(rate(container_cpu_usage_seconds_total{namespace="customer-acme"}[5m]))

# Memory usage for customer-acme
sum(container_memory_working_set_bytes{namespace="customer-acme"})

# Request rate to customer-acme's services
sum(rate(zenith_api_request_duration_seconds_count{namespace="customer-acme"}[5m]))

# Top 5 customers by CPU usage
topk(5,
  sum by (namespace) (rate(container_cpu_usage_seconds_total{namespace=~"customer-.*"}[5m]))
)

# Customers exceeding their CPU quota
sum by (namespace) (rate(container_cpu_usage_seconds_total{namespace=~"customer-.*"}[5m]))
  / on(namespace)
kube_resourcequota{resource="requests.cpu", type="hard"}
  > 0.8
```

**Logs (LogQL) -- filter by customer:**

```logql
# All logs for customer-acme
{namespace="customer-acme"}

# Errors in customer-acme's pods
{namespace="customer-acme"} | json | level="error"

# Database errors for a specific customer's CNPG cluster
{namespace="customer-acme", app=~"cnpg.*"} |= "ERROR"
```

**Traces (Tempo) -- filter by customer:**

In the Grafana Tempo search UI, filter by:
- `resource.k8s.namespace.name = customer-acme`

This works because the OTel Collector's `k8sattributes` processor automatically
adds the namespace to every span.

**Network (Hubble) -- filter by customer:**

```bash
# All network flows for customer-acme
hubble observe --namespace customer-acme

# Cross-tenant traffic attempts (should all be DROPPED)
hubble observe --from-namespace customer-acme --to-namespace customer-beta
```

### Customer-Visible Metrics

In a future iteration, the Zenith Web Platform will expose a subset of metrics to
customers so they can see their own resource usage. The planned approach:

1. **Grafana with per-tenant RBAC**: Each customer realm in Keycloak maps to a Grafana
   organization. Dashboards in that org are pre-filtered to `namespace=customer-{id}`.
2. **Embedded panels**: The Web Platform embeds Grafana panels via iframe with
   authenticated embed URLs (Grafana's `allow_embedding` + auth proxy).
3. **Exposed metrics**:
   - CPU / memory usage vs. quota
   - Request count and error rate
   - Database connection count and query latency
   - Storage usage (PVC + S3)
   - Certificate expiry dates

This is not in the V2 launch scope but is designed to be straightforward to add because
all the underlying data already exists and is tenant-segmented by namespace.

---

## How to Run

### Accessing Grafana

Grafana is exposed internally within the cluster. For operator access:

```bash
# Port-forward to Grafana (from your local machine)
kubectl port-forward -n monitoring svc/grafana 3000:3000

# Open in browser
open http://localhost:3000

# Default credentials (change after first login)
# Username: admin
# Password: (from secret)
kubectl get secret -n monitoring grafana-admin -o jsonpath='{.data.password}' | base64 -d
```

For production, Grafana will be accessible via an APISIX route at
`https://grafana.internal.freezenith.com`, protected by Keycloak SSO (the `openid-connect`
plugin on the APISIX route).

### Accessing Hubble UI

```bash
# Port-forward to Hubble UI
kubectl port-forward -n kube-system svc/hubble-ui 12000:80

# Open in browser
open http://localhost:12000
```

Hubble CLI (for terminal-based flow observation):

```bash
# Install hubble CLI (macOS)
brew install hubble

# Port-forward the Hubble relay
kubectl port-forward -n kube-system svc/hubble-relay 4245:80 &

# Observe flows
hubble observe --server localhost:4245 --namespace zenith-platform
```

### Common Debugging Workflows

**Workflow 1: "The API is slow"**

1. Open the **APISIX Gateway** dashboard in Grafana.
2. Check the p95/p99 latency panels. Identify which route is slow.
3. Click the route name to filter. Check if upstream latency is high (backend is slow)
   or if APISIX itself is slow (unlikely).
4. Open Grafana Explore with Tempo data source. Search for traces with:
   - Service: `apisix`
   - Min duration: `1s`
5. Open a slow trace. The span waterfall shows exactly which downstream call is slow.
6. If the slow span is a database query, open the **CNPG Database Health** dashboard
   and check query latency, active connections, and WAL lag.

**Workflow 2: "Customer reports errors"**

1. Open Grafana Explore with Loki data source.
2. Query: `{namespace="customer-acme"} | json | level="error"`
3. Scan the log lines. Find one with a trace_id.
4. Click the trace_id link to jump to the trace in Tempo.
5. The trace waterfall shows which service returned the error.
6. Check if the error is in zenith-api (our bug) or in the customer's own app.

**Workflow 3: "Pods cannot reach the database"**

1. Open Hubble CLI or Hubble UI.
2. `hubble observe --namespace customer-acme --to-port 5432 --verdict DROPPED`
3. If you see DROPPED flows, a Cilium NetworkPolicy is blocking the traffic.
4. Check the CiliumNetworkPolicy in the customer's namespace:
   `kubectl get cnp -n customer-acme -o yaml`
5. Fix the policy and verify with Hubble that flows change from DROPPED to FORWARDED.

**Workflow 4: "Who deleted the production ConfigMap?"**

1. Open Grafana Explore with Loki data source.
2. Query:
   ```logql
   {job="kubernetes-audit"} | json | verb="delete" | resource="configmaps" | namespace="zenith-platform"
   ```
3. The log line includes the username, source IP, and timestamp of the deletion.

**Workflow 5: "Is tenant isolation actually working?"**

1. From a pod in customer-acme, attempt to reach a service in customer-beta:
   ```bash
   kubectl exec -n customer-acme deploy/web -- curl -s http://api.customer-beta.svc.cluster.local
   ```
2. This should fail (connection refused or timeout).
3. Verify in Hubble:
   ```bash
   hubble observe --from-namespace customer-acme --to-namespace customer-beta
   ```
4. You should see `DROPPED` verdict with `POLICY_DENIED` reason.
5. If you see `FORWARDED`, there is a gap in your NetworkPolicy -- fix it immediately.

---

## Component Summary Table

| Component | Role | Namespace | Deployment Type | Port | Resource Estimate |
|-----------|------|-----------|-----------------|------|-------------------|
| **Prometheus** | Metrics collection + storage | `monitoring` | StatefulSet | 9090 | 500m CPU, 2-4Gi RAM, 100Gi disk |
| **Grafana** | Visualization + dashboards | `monitoring` | Deployment | 3000 | 200m CPU, 256Mi RAM |
| **Loki** | Log aggregation + storage | `monitoring` | StatefulSet | 3100 | 250m CPU, 512Mi-1Gi RAM, 50Gi disk |
| **Promtail** | Log collection (node agent) | `monitoring` | DaemonSet | 3101 | 100m CPU, 128Mi RAM per node |
| **Tempo** | Trace storage | `monitoring` | StatefulSet | 3200 (query), 4317 (OTLP) | 250m CPU, 512Mi RAM, 20Gi disk |
| **OTel Collector** | Trace/metric pipeline | `monitoring` | DaemonSet | 4317 (gRPC), 4318 (HTTP) | 200m CPU, 512Mi RAM per node |
| **Hubble** | Network flow observation | `kube-system` | Part of Cilium agent | 4245 (relay) | Included in Cilium agent |
| **Hubble UI** | Network flow visualization | `kube-system` | Deployment | 80 | 100m CPU, 128Mi RAM |
| **Alertmanager** | Alert routing + notification | `monitoring` | StatefulSet | 9093 | 100m CPU, 128Mi RAM |

**Total resource estimate for the observability stack:**
- CPU: ~2-3 cores (across all nodes)
- Memory: ~6-8 Gi
- Disk: ~170 Gi (Prometheus 100Gi + Loki 50Gi + Tempo 20Gi)

This is a meaningful resource investment, but it pays for itself the first time you
debug a multi-service incident in minutes instead of hours.

---

## Summary

The Zenith V2 observability stack follows the principle: **instrument everything, store
it cheaply, query it from one place.** Every component in the platform -- from APISIX at
the edge to CNPG at the data layer -- emits metrics, logs, and traces. All of it flows
into the Prometheus/Loki/Tempo backend and surfaces through Grafana.

The key integrations that make this powerful:

1. **APISIX OpenTelemetry plugin** generates traces for every API request, giving
   end-to-end visibility from gateway to database.
2. **Hubble** provides network-level visibility without any application instrumentation,
   powered by Cilium's eBPF dataplane.
3. **Trace-to-log correlation** via embedded trace IDs lets you jump from a slow trace
   to the exact log lines that explain what went wrong.
4. **Per-customer filtering** via namespace labels works consistently across all four
   pillars (metrics, logs, traces, flows).
5. **Alertmanager routing** ensures critical alerts wake someone up (PagerDuty) while
   informational alerts flow quietly to Slack.

When something goes wrong in production, the debugging path is always:
**Alert -> Dashboard -> Logs -> Trace -> Fix.**
