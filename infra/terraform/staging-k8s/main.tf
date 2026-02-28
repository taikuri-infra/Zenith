# =============================================================================
# Zenith Staging — K8s Platform (V2)
# =============================================================================
#
# Architecture (3-layer):
#
#   1. Terraform (staging/)       → Hetzner server + Cloudflare DNS
#   2. Ansible                    → k3s + Cilium CLI + prerequisite secrets
#   3. Terraform (staging-k8s/)   → This file: all Helm releases INTO the cluster
#
#   Internet
#      |
#      v
#   +-----------+
#   | Traefik   |  Ingress Controller (TLS termination, L7 routing)
#   +-----+-----+  Installed by k3s. Listens on :80 / :443.
#         |
#         +---> stage.freezenith.com           -> zenith-landing     (:3000)
#         +---> api.stage.freezenith.com       -> APISIX Gateway     -> zenith-api (:8080)
#         +---> auth.stage.freezenith.com      -> Keycloak           (:8080)
#         +---> argocd.stage.freezenith.com    -> ArgoCD             (:443)
#         +---> registry.stage.freezenith.com  -> Harbor             (:443)
#         +---> temporal.stage.freezenith.com  -> Temporal Web UI    (:8080)
#         +---> hubble.stage.freezenith.com    -> Hubble UI          (:80)
#         +---> alerts.stage.freezenith.com    -> Alertmanager       (:9093)
#         +---> tempo.stage.freezenith.com     -> Grafana/Tempo      (:3000)
#
# Helm Releases (V2):
#
#   +---------------------+-----------------+--------------------------------------+
#   | Release             | Namespace       | Purpose                              |
#   +---------------------+-----------------+--------------------------------------+
#   | cert-manager        | cert-manager    | TLS automation (DNS-01 Cloudflare)   |
#   | sealed-secrets      | sealed-secrets  | Encrypted secrets for GitOps         |
#   | cnpg                | cnpg-system     | PostgreSQL operator                  |
#   | keycloak            | keycloak        | Identity provider                    |
#   | apisix              | apisix          | API Gateway (replaces Kong)          |
#   | external-dns        | external-dns    | Auto DNS via Cloudflare              |
#   | argocd              | argocd          | GitOps engine                        |
#   | harbor              | harbor          | Container registry                   |
#   | temporal            | temporal        | Workflow engine                      |
#   | kyverno             | kyverno         | Admission policies                   |
#   | falco               | falco           | Runtime security                     |
#   | velero              | velero          | Cluster backup                       |
#   | kube-prometheus-stack| monitoring      | Prometheus + Grafana + Alertmanager  |
#   | loki                | monitoring      | Log aggregation                      |
#   | tempo               | monitoring      | Distributed tracing                  |
#   | otel-collector       | monitoring      | Trace pipeline                       |
#   | keda                | keda            | Scale-to-zero                        |
#   | zenith-platform     | zenith-staging  | Shared resources                     |
#   | zenith-api          | zenith-staging  | Go API server                        |
#   | zenith-landing      | zenith-staging  | Next.js landing page                 |
#   | zenith-tenant       | zenith-staging  | Per-customer deployments             |
#   +---------------------+-----------------+--------------------------------------+
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

  # Hetzner CSI
  hcloud_token = var.hcloud_token

  # Environment
  platform_namespace = "zenith-staging"
  environment        = "staging"
  cert_issuer_email  = "admin@freezenith.com"

  # Domains
  domain         = var.domain
  cluster_domain = var.cluster_domain

  # Helm charts from local filesystem (Harbor is not ready on first deploy)
  chart_repository = ""
  chart_version    = var.zenith_chart_version

  # Local chart paths (used when chart_repository is empty)
  platform_chart_path = "${path.module}/../../helm/zenith-platform"
  api_chart_path      = "${path.module}/../../helm/zenith-api"
  landing_chart_path  = "${path.module}/../../helm/zenith-landing"
  demo_chart_path     = "${path.module}/../../helm/zenith-demo"
  tenant_chart_path   = "${path.module}/../../helm/zenith-tenant"

  # Per-chart values files
  platform_values_file = "${path.module}/../../helm/zenith-platform/values-staging.yaml"
  api_values_file      = "${path.module}/../../helm/zenith-api/values-staging.yaml"
  landing_values_file  = "${path.module}/../../helm/zenith-landing/values-staging.yaml"
  demo_values_file     = "${path.module}/../../helm/zenith-demo/values-staging.yaml"
  tenant_values_file   = "${path.module}/../../helm/zenith-tenant/values-staging.yaml"

  # Registry credentials (for OCI pull + imagePullSecret)
  registry_host          = var.registry_host
  customer_registry_host = var.customer_registry_host
  registry_username      = var.registry_username
  registry_password      = var.registry_password

  # App secrets
  jwt_secret     = var.jwt_secret
  admin_email    = var.admin_email
  admin_password = var.admin_password

  # S3 / Object Storage (Hetzner)
  s3_access_key = var.s3_access_key
  s3_secret_key = var.s3_secret_key
  s3_endpoint   = var.s3_endpoint

  # Database Storage Sizes
  keycloak_db_storage_size = var.keycloak_db_storage_size
  free_db_storage_size     = var.free_db_storage_size

  # Cloudflare
  cloudflare_api_token = var.cloudflare_api_token

  # Keycloak secrets
  keycloak_db_password    = var.keycloak_db_password
  keycloak_admin_password = var.keycloak_admin_password

  # Temporal secrets
  temporal_db_user     = var.temporal_db_user
  temporal_db_password = var.temporal_db_password

  # ArgoCD / GitOps
  github_token = var.github_token

  # --- V2 Feature flags ---
  enable_cnpg           = true
  enable_apisix         = true    # V2: replaces Kong
  enable_keycloak       = true
  enable_external_dns   = true
  enable_argocd         = true
  enable_harbor         = true
  enable_temporal       = true
  enable_kyverno        = true
  enable_falco          = true
  enable_velero         = true
  enable_sealed_secrets = true
  enable_monitoring     = true
  enable_keda           = true
  enable_demo           = true
  enable_tenants        = false  # Tenants provisioned dynamically via purchase flow
}
