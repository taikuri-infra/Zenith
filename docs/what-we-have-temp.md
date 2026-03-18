# Zenith Technology Stack — Current State (Updated 2026-03-18)

## Networking

| # | Technology | Version | What it does | Score | Notes |
|---|-----------|---------|-------------|:-----:|-------|
| 1 | Cilium | 1.16.5 | eBPF CNI — pod networking + NetworkPolicy | 9/10 | Flannel replaced, eBPF-based, code-managed via Terraform |
| 2 | Traefik | 3.5.1 | Ingress controller + reverse proxy | 9/10 | IngressRoute CRD, middleware, cross-namespace |
| 3 | APISIX | 3.15.0 | API Gateway — rate limit, cors, WAF | 9/10 | WAF active: uri-blocker + ua-restriction |
| 4 | External-DNS | 0.18.0 | Auto DNS record management | 9/10 | Cloudflare, IngressRoute source |

## Security

| # | Technology | Version | What it does | Score | Notes |
|---|-----------|---------|-------------|:-----:|-------|
| 5 | Cloudflare Zero Trust | - | Tunnel + Access SSO | 10/10 | Google OAuth, 9 services, auth.proxy SSO |
| 6 | Cert-Manager | 1.17.2 | Auto TLS certificates | 10/10 | DNS-01 Cloudflare solver |
| 7 | Kyverno | 3.7.1 | Policy engine — pod security | 9/10 | 6 policies: image arch, privileged, non-root, host-ns, resource limits, cosign |
| 8 | Falco + Sidekick | 4.18.0 | Runtime threat detection + alerts | 8/10 | Sidekick enabled, Slack webhook ready (needs URL) |
| 9 | Sealed Secrets | 2.16.1 | Encrypt secrets in Git | 9/10 | GitOps-safe secrets |
| 10 | NetworkPolicies | - | Network segmentation | 9/10 | 11 policies, default-deny + explicit allow, Hubble for debugging |
| 11 | WAF (APISIX) | - | SQL injection/XSS/path traversal block | 9/10 | uri-blocker + ua-restriction on all routes |

## Observability

| # | Technology | Version | What it does | Score | Notes |
|---|-----------|---------|-------------|:-----:|-------|
| 12 | Prometheus | 61.3.1 | Metrics collection | 9/10 | kube-prometheus-stack |
| 13 | Grafana | 61.3.1 | Dashboards + visualization | 10/10 | SSO via Cloudflare auth.proxy, auto Admin role |
| 14 | Loki | 6.6.4 | Log aggregation | 8/10 | SingleBinary, ok for staging |
| 15 | Tempo | 1.10.1 | Distributed tracing | 7/10 | API integration pending |
| 16 | Promtail | - | Log shipper to Loki | 8/10 | Part of monitoring stack |
| 17 | OTEL Collector | 0.96.0 | Trace/metric pipeline | 7/10 | Needs more integration |
| 18 | Hubble UI | 1.16.5 | Network flow visualization | 9/10 | Behind Zero Trust tunnel, Cilium-powered |

## Data

| # | Technology | Version | What it does | Score | Notes |
|---|-----------|---------|-------------|:-----:|-------|
| 19 | CNPG | 0.23.0 | PostgreSQL operator | 10/10 | 3 clusters, backup/restore tested, S3 WAL |
| 20 | Harbor | 1.15.1 | Container registry + Trivy scan | 8/10 | Internal + customer registries |
| 21 | Redis Operator | 0.18.0 | Managed Redis instances | 9/10 | OpsTree operator, CRDs ready |
| 22 | MongoDB Operator | 1.18.0 | Managed MongoDB instances | 9/10 | Percona PSMDB operator, CRDs ready |

## Platform

| # | Technology | Version | What it does | Score | Notes |
|---|-----------|---------|-------------|:-----:|-------|
| 23 | ArgoCD | 7.3.11 | GitOps CD | 9/10 | Image Updater, staging branch |
| 24 | ArgoCD Image Updater | 0.11.0 | Auto image update | 8/10 | SHA-based staging deploys |
| 25 | KEDA | 2.16.0 | Scale-to-zero | 9/10 | HTTP Add-on, cold-start splash |
| 26 | Keycloak | 25.2.0 | Identity provider | 5/10 | Installed, not yet integrated |
| 27 | Temporal | 0.45.0 | Workflow orchestration | 6/10 | Installed, API integration pending |
| 28 | Velero | 11.4.0 | Cluster backup/restore | 9/10 | Daily backup S3, 30-day retention |
| 29 | Cosign (Kyverno) | - | Image signature verification | 7/10 | Policy ready (Audit mode), needs key generation |

## Average Score: 8.6/10

## Remaining Items for 10/10

| Item | Current | Target | Priority |
|------|---------|--------|----------|
| Falco Slack webhook URL | Config ready, no URL | Active alerting | HIGH |
| Cosign key generation | Policy in Audit mode | Enforce + CI signing | MEDIUM |
| Tempo API integration | Installed | Traces from API | MEDIUM |
| Keycloak integration | Installed | SSO for apps | LOW |
| Temporal API integration | Installed | Workflow automation | LOW |
| K8s Audit logging | Not installed | API audit trail | MEDIUM |

## Services Behind Zero Trust Tunnel

| Service | URL | SSO Auto-login |
|---------|-----|:-:|
| Grafana | grafana-stage.freezenith.com | Yes (auth.proxy) |
| Prometheus | prometheus-stage.freezenith.com | No login needed |
| Loki | loki-stage.freezenith.com | No login needed |
| Alertmanager | alerts-stage.freezenith.com | No login needed |
| ArgoCD | argocd-stage.freezenith.com | Has own login |
| Harbor | harbor-stage.freezenith.com | Has own login |
| Tempo | tempo-stage.freezenith.com | No login needed |
| Temporal | temporal-stage.freezenith.com | No login needed |
| Hubble UI | hubble-stage.freezenith.com | No login needed |

## Public Services (No Tunnel)

| Service | URL |
|---------|-----|
| Landing | stage.freezenith.com |
| API | api.stage.freezenith.com |
| Web App | app.stage.freezenith.com |
| Keycloak | auth.stage.freezenith.com |
