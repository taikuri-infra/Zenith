# Phase 7: Observability — Monitoring, Logging, Health

## Summary

Deploy monitoring stack (Prometheus + Grafana), logging (Loki), and health checks so both operators (MC) and customers (Web) can see what's happening.

## Prerequisites

- Phase 6 complete (data services)
- Sufficient cluster resources for monitoring stack

## Steps

### Step 7.1: Deploy Monitoring Stack via Terraform

**What:** Install kube-prometheus-stack + Loki via Helm release.

**Build:**
- Enable `monitoring = true` in Terraform staging-k8s
- Uses existing `infra/helm/monitoring/` chart
- Grafana accessible at `grafana.stage.freezenith.com` (or internal only)
- Prometheus scrapes all Zenith services
- Pre-built dashboards for: node metrics, API latency, pod resource usage

**Your manual work:**
- Add DNS record for Grafana (if external access wanted)

**Verify:**
```bash
kubectl get pods -n monitoring
# prometheus, grafana, loki pods running

# Grafana UI
open https://grafana.stage.freezenith.com
```

### Step 7.2: API Metrics Endpoint

**What:** API server exposes Prometheus metrics.

**Build:**
- `/metrics` endpoint on port 9090 (already partially implemented)
- Metrics: request count, latency histogram, active connections, error rate
- ServiceMonitor CRD for Prometheus to discover

**Your manual work:** None

**Verify:**
```bash
curl http://api-service:9090/metrics | grep zenith_
```

### Step 7.3: MC Infrastructure Dashboard

**What:** MC infrastructure page shows real cluster metrics.

**Build:**
- MC `/infrastructure` page queries Prometheus via API proxy
- Shows: CPU/RAM usage per node, pod counts, storage usage
- MC `/` dashboard stats are real numbers

**Your manual work:** None

**Verify:**
1. Login to MC → Infrastructure
2. See real server metrics, not mock data

### Step 7.4: Web Platform Monitoring Page

**What:** Web Platform monitoring page shows real app metrics.

**Build:**
- `/monitoring` page embedded Grafana panels or custom charts
- Per-app: request rate, error rate, latency, resource usage
- Log viewer pulling from Loki

**Your manual work:** None

**Verify:**
1. Login to Web Platform → Monitoring
2. See real app metrics for deployed apps

## Acceptance Criteria

- [ ] Prometheus + Grafana + Loki running in cluster
- [ ] Pre-built dashboards available
- [ ] API metrics scraped by Prometheus
- [ ] MC shows real infrastructure metrics
- [ ] Web Platform shows real app metrics
- [ ] Alerting rules configured for critical conditions
- [ ] Logs searchable via Loki
