# =============================================================================
# Zenith Production — K8s Platform
# =============================================================================
#
# Mirrors the staging-k8s configuration with production-specific values.
# ArgoCD watches the `main` branch (not `staging`).
#
# Usage:
#   terraform init
#   terraform plan -var-file=terraform.tfvars
#   terraform apply -var-file=terraform.tfvars
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
  platform_namespace = "zenith-platform"
  environment        = "production"
  cert_issuer_email  = "admin@freezenith.com"

  # Domains
  domain         = var.domain
  cluster_domain = var.cluster_domain

  # Helm charts — from Harbor OCI registry in production
  chart_repository = var.chart_repository
  chart_version    = var.zenith_chart_version

  # Local chart paths (fallback when chart_repository is empty)
  platform_chart_path = "${path.module}/../../helm/zenith-platform"
  api_chart_path      = "${path.module}/../../helm/zenith-api"
  landing_chart_path  = "${path.module}/../../helm/zenith-landing"
  # Per-chart values files (production overrides)
  platform_values_file = "${path.module}/../../helm/zenith-platform/values-production.yaml"
  api_values_file      = "${path.module}/../../helm/zenith-api/values-production.yaml"
  landing_values_file  = "${path.module}/../../helm/zenith-landing/values-production.yaml"

  # Registry credentials (for OCI pull + imagePullSecret)
  registry_host          = var.registry_host
  customer_registry_host = var.customer_registry_host
  registry_username      = var.registry_username
  registry_password      = var.registry_password

  # App secrets
  jwt_secret             = var.jwt_secret
  admin_email            = var.admin_email
  admin_password         = var.admin_password
  github_webhook_secret  = var.github_webhook_secret
  secrets_encryption_key = var.secrets_encryption_key
  resend_api_key         = var.resend_api_key
  google_client_id       = var.google_client_id

  # S3 / Object Storage (Hetzner)
  s3_access_key = var.s3_access_key
  s3_secret_key = var.s3_secret_key
  s3_endpoint   = var.s3_endpoint

  # Database Storage Sizes (production: larger than staging)
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

  # ArgoCD / GitOps — watches `main` branch in production
  github_token = var.github_token

  # --- Feature flags (all enabled for production) ---
  enable_cnpg           = true
  enable_apisix         = true
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
}
