# =============================================================================
# Zenith Staging — K8s Platform
# =============================================================================
#
# Architecture:
#
#   Internet
#      |
#      v
#   +-----------+
#   | Traefik   |  Ingress Controller (TLS termination, L7 routing)
#   +-----+-----+  Installed by k3s. Listens on :80 / :443.
#         |
#         +---> stage.freezenith.com           -> zenith-landing     (:3000)
#         +---> ms.stage.freezenith.com        -> zenith-mc-demo     (:3100)
#         +---> cloud.stage.freezenith.com     -> zenith-web-demo    (:3000)
#         |
#         +---> api.stage.freezenith.com       -> +---------------+
#         |                                       | Kong Gateway  |
#         |                                       | rate-limit    |
#         |                                       | cors          |
#         |                                       +-------+-------+
#         |                                               |
#         |                                               +---> zenith-api (:8080)
#         |
#         +---> embermind-ms.stage.freezenith.com  -> zenith-mc   (:3100)  [tenant]
#         +---> embermind.stage.freezenith.com     -> zenith-web  (:3000)  [tenant]
#
# Helm Releases (modular — independently deployable):
#
#   +-------------------+----------------+--------------------------------------+
#   | Release           | Namespace      | Purpose                              |
#   +-------------------+----------------+--------------------------------------+
#   | cert-manager      | cert-manager   | TLS cert automation (Let's Encrypt)  |
#   | cnpg              | cnpg-system    | CloudNativePG PostgreSQL operator    |
#   | zenith-platform   | zenith-staging | Secrets, postgres CR, certs, midware |
#   | zenith-api        | zenith-staging | Go API server                        |
#   | zenith-landing    | zenith-staging | Next.js landing page                 |
#   | zenith-demo       | zenith-staging | MC + Web demo (disabled)             |
#   | zenith-tenant     | zenith-staging | Per-customer MC + Web deployments    |
#   | kong              | kong           | API gateway (DB-less, ClusterIP)     |
#   | keda              | keda           | Event-driven autoscaling             |
#   | keda-http-addon   | keda           | HTTP-based scale-to-zero             |
#   | monitoring        | monitoring     | Prometheus + Grafana + Loki          |
#   +-------------------+----------------+--------------------------------------+
#
# Images pulled from: registry.stage.freezenith.com/zenith-stage/*
# Charts pulled from: oci://registry.stage.freezenith.com/zenith-stage/<chart>
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

  # Helm charts from Harbor OCI registry
  chart_repository = "oci://${var.registry_host}/zenith-stage"
  chart_version    = var.zenith_chart_version

  # Per-chart values files
  platform_values_file = "${path.module}/../../helm/zenith-platform/values-staging.yaml"
  api_values_file      = "${path.module}/../../helm/zenith-api/values-staging.yaml"
  landing_values_file  = "${path.module}/../../helm/zenith-landing/values-staging.yaml"
  demo_values_file     = "${path.module}/../../helm/zenith-demo/values-staging.yaml"
  tenant_values_file   = "${path.module}/../../helm/zenith-tenant/values-staging.yaml"

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
  enable_demo       = false
  enable_tenants    = true

  # Monitoring chart
  monitoring_chart_path  = "${path.module}/../../helm/monitoring"
  monitoring_values_file = "${path.module}/../../helm/monitoring/values.yaml"
  monitoring_domain      = "stage.freezenith.com"
}
