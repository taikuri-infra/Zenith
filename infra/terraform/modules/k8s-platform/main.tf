# =============================================================================
# k8s-platform module — V2
# =============================================================================
#
# Installs the full Zenith platform stack on a k3s cluster.
# Designed for parity between staging and production (same components,
# different resource limits).
#
# Architecture (3-layer):
#   1. Terraform (staging/)       → Hetzner server + Cloudflare DNS
#   2. Ansible                    → k3s binary + Cilium CLI + prerequisite secrets
#   3. Terraform (staging-k8s/)   → This module: all Helm releases INTO the cluster
#
# V2 Components:
#   Infrastructure:
#     1.  cert-manager       — TLS automation (DNS-01 via Cloudflare)
#     2.  Sealed Secrets     — Encrypted secrets for GitOps
#     3.  CNPG Operator      — PostgreSQL operator
#     4.  CNPG Keycloak      — Dedicated PG cluster for Keycloak
#     5.  CNPG Free          — Shared PG for free-tier customers
#     6.  Keycloak           — Identity provider (realm per customer)
#     7.  APISIX + etcd      — API Gateway (replaces Kong)
#     8.  external-dns       — Automatic Cloudflare DNS from Ingress
#     9.  ArgoCD             — GitOps engine (App-of-Apps)
#     10. Harbor             — Container & Helm chart registry
#     11. Temporal           — Provisioning workflow engine
#     12. Kyverno            — Admission policy engine
#     13. Falco              — Runtime security (eBPF)
#     14. Velero             — Cluster backup to S3
#     15. Prometheus+Grafana — Monitoring & alerting
#     16. Loki               — Log aggregation
#     17. Tempo              — Distributed tracing
#     18. OTel Collector     — Trace & metric pipeline
#
#   Application Charts:
#     19. zenith-platform    — Shared resources (secrets, certs, middleware)
#     20. zenith-api         — Go API server
#     21. zenith-landing     — Next.js landing page
#     22. zenith-demo        — Demo instances (optional)
#     23. zenith-tenant      — Per-customer deployments (optional)
#
# Files:
#   main.tf            — This file (terraform block + PriorityClasses)
#   certmanager.tf     — cert-manager + ClusterIssuer
#   sealed_secrets.tf  — Sealed Secrets
#   storage.tf         — CNPG operator + Keycloak PG + Free PG clusters
#   identity.tf        — Keycloak
#   gateway.tf         — APISIX + external-dns
#   gitops.tf          — ArgoCD + Image Updater
#   registry.tf        — Harbor
#   temporal.tf        — Temporal workflow engine
#   security.tf        — Kyverno + Falco + Velero
#   observability.tf   — Prometheus + Loki + Tempo + OTel + Hubble UI
#   autoscaling.tf     — KEDA
#   apps.tf            — zenith-platform, api, landing, demo
#   tenant.tf          — zenith-tenant (per-customer)
#   variables.tf       — All input variables
#   outputs.tf         — All outputs
#
# =============================================================================

terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.17"
    }
  }
}

# =============================================================================
# PriorityClasses — Pod eviction ordering (5.1)
# =============================================================================

resource "kubernetes_priority_class" "system_critical" {
  metadata {
    name = "system-critical"
  }
  value          = 1000000
  global_default = false
  description    = "Cilium, CoreDNS, Traefik — never evict"
}

resource "kubernetes_priority_class" "infra_critical" {
  metadata {
    name = "infra-critical"
  }
  value          = 500000
  global_default = false
  description    = "CNPG, Keycloak, APISIX, cert-manager — evict last"
}

resource "kubernetes_priority_class" "platform" {
  metadata {
    name = "platform"
  }
  value          = 100000
  global_default = false
  description    = "zenith-api, Temporal, Harbor, monitoring"
}

resource "kubernetes_priority_class" "customer" {
  metadata {
    name = "customer"
  }
  value          = 10000
  global_default = true
  description    = "All customer workloads — evict first"
}
