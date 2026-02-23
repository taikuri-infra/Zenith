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

  platform_namespace = "zenith-platform"
  cert_issuer_email  = "admin@freezenith.com"

  # Helm chart paths
  zenith_chart_path  = "${path.module}/../../helm/zenith"
  zenith_values_file = "${path.module}/../../helm/zenith/values-production.yaml"

  # Monitoring chart
  monitoring_chart_path  = "${path.module}/../../helm/monitoring"
  monitoring_values_file = "${path.module}/../../helm/monitoring/values.yaml"

  # Secrets
  jwt_secret     = var.jwt_secret
  admin_email    = var.admin_email
  admin_password = var.admin_password
  db_password    = var.db_password

  # Production: enable KEDA + monitoring
  enable_keda       = var.enable_keda
  enable_monitoring = var.enable_monitoring
}
