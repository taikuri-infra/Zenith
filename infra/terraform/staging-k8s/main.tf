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

  # Helm chart paths (relative to this directory)
  zenith_chart_path  = "${path.module}/../../helm/zenith"
  zenith_values_file = "${path.module}/../../helm/zenith/values-staging.yaml"

  # Secrets
  jwt_secret     = var.jwt_secret
  admin_email    = var.admin_email
  admin_password = var.admin_password
  db_password    = var.db_password

  # Staging: no KEDA, no monitoring
  enable_keda       = false
  enable_monitoring = false
}
