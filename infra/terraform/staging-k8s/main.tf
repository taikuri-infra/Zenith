# =============================================================================
# Zenith Staging — K8s Platform (Phase 3)
# =============================================================================
#
# Architecture:
#
#   Internet
#      │
#      ▼
#   ┌──────────┐
#   │ Traefik  │  Ingress Controller (TLS termination, L7 routing)
#   └────┬─────┘  Installed by k3s. Listens on :80 / :443.
#        │
#        ├──→ stage.freezenith.com           → zenith-landing     (:3000)
#        ├──→ ms.stage.freezenith.com        → zenith-mc-demo     (:3100)
#        ├──→ cloud.stage.freezenith.com     → zenith-web-demo    (:3000)
#        │
#        ├──→ api.stage.freezenith.com       → ┌──────────────┐
#        │                                      │ Kong Gateway │  API Gateway
#        │                                      │ rate-limit   │  (rate limiting,
#        │                                      │ cors         │   CORS, JWT)
#        │                                      └──────┬───────┘
#        │                                             │
#        │                                             └──→ zenith-api (:8080)
#        │
#        ├──→ embermind-ms.stage.freezenith.com  → zenith-mc   (:3100)  [tenant]
#        └──→ embermind.stage.freezenith.com     → zenith-web  (:3000)  [tenant]
#
# Helm Releases managed by this module:
#
#   ┌─────────────────┬────────────────┬──────────────────────────────────────┐
#   │ Release         │ Namespace      │ Purpose                              │
#   ├─────────────────┼────────────────┼──────────────────────────────────────┤
#   │ cert-manager    │ cert-manager   │ TLS cert automation (Let's Encrypt)  │
#   │ cnpg            │ cnpg-system    │ CloudNativePG PostgreSQL operator    │
#   │ zenith          │ zenith-staging │ Platform apps (API, landing, PG,     │
#   │                 │                │ tenants, ingress, certs)             │
#   │ kong            │ kong           │ API gateway (DB-less, ClusterIP)     │
#   │ keda            │ keda           │ Event-driven autoscaling             │
#   │ keda-http-addon │ keda           │ HTTP-based scale-to-zero             │
#   │ monitoring      │ monitoring     │ Prometheus + Grafana + Loki          │
#   └─────────────────┴────────────────┴──────────────────────────────────────┘
#
# Images pulled from: registry.stage.freezenith.com/zenith-stage/*
# Chart pulled from:  oci://registry.stage.freezenith.com/zenith-stage/zenith
#
# Usage:
#   terraform init
#   terraform apply            # auto-loads terraform.tfvars
#
# =============================================================================

terraform {
  required_version = ">= 1.5"

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

# --- Providers (connect to k3s via kubeconfig) ---

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

provider "helm" {
  kubernetes {
    config_path = var.kubeconfig_path
  }
}

# --- K8s Platform ---

module "platform" {
  source = "../modules/k8s-platform"

  platform_namespace = "zenith-staging"
  cert_issuer_email  = "admin@freezenith.com"

  # Helm chart from Harbor OCI registry
  zenith_chart_repository = "oci://${var.registry_host}/zenith-stage"
  zenith_chart_version    = var.zenith_chart_version
  zenith_values_file      = "${path.module}/../../helm/zenith/values-staging.yaml"

  # Registry credentials (for OCI pull + imagePullSecret)
  registry_host     = var.registry_host
  registry_username = var.registry_username
  registry_password = var.registry_password

  # App secrets
  jwt_secret     = var.jwt_secret
  admin_email    = var.admin_email
  admin_password = var.admin_password

  # Feature flags
  enable_cnpg       = true
  enable_kong       = true
  enable_keda       = true
  enable_monitoring = true

  # Monitoring chart
  monitoring_chart_path  = "${path.module}/../../helm/monitoring"
  monitoring_values_file = "${path.module}/../../helm/monitoring/values.yaml"
  monitoring_domain      = "stage.freezenith.com"
}
