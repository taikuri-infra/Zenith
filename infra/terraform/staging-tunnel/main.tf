# =============================================================================
# Zenith Staging — Cloudflare Zero Trust Tunnel
# =============================================================================
#
# Standalone config for the Cloudflare Tunnel. Separated from staging-k8s
# because the platform module's helm_release resources have label drift
# that blocks targeted applies.
#
# Usage:
#   terraform init
#   terraform apply
# =============================================================================

terraform {
  required_version = ">= 1.5"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.35"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

module "tunnel" {
  source = "../modules/cloudflare-tunnel"

  account_id  = var.cloudflare_account_id
  zone_id     = var.freezenith_zone_id
  tunnel_name = "zenith-staging"
  environment = "staging"

  # Use 1st-level subdomains (grafana-stage vs grafana.stage) because
  # Cloudflare free Universal SSL only covers *.freezenith.com, not *.stage.freezenith.com
  services = {
    grafana = {
      subdomain = "grafana-stage"
      service   = "http://kube-prometheus-stack-grafana.monitoring.svc.cluster.local:80"
    }
    prometheus = {
      subdomain = "prometheus-stage"
      service   = "http://kube-prometheus-stack-prometheus.monitoring.svc.cluster.local:9090"
    }
    loki = {
      subdomain = "loki-stage"
      service   = "http://loki.monitoring.svc.cluster.local:3100"
    }
    alertmanager = {
      subdomain = "alerts-stage"
      service   = "http://kube-prometheus-stack-alertmanager.monitoring.svc.cluster.local:9093"
    }
    hubble = {
      subdomain = "hubble-stage"
      service   = "http://hubble-ui.kube-system.svc.cluster.local:80"
    }
    argocd = {
      subdomain = "argocd-stage"
      service   = "https://argocd-server.argocd.svc.cluster.local:443"
    }
    harbor = {
      subdomain = "harbor-stage"
      service   = "http://harbor-core.harbor.svc.cluster.local:80"
    }
    tempo = {
      subdomain = "tempo-stage"
      service   = "http://tempo.monitoring.svc.cluster.local:3100"
    }
    temporal = {
      subdomain = "temporal-stage"
      service   = "http://temporal-web.temporal.svc.cluster.local:8080"
    }
    policy_reporter = {
      subdomain = "kyverno-stage"
      service   = "http://policy-reporter-ui.kyverno.svc.cluster.local:8080"
    }
    mc = {
      subdomain = "mc-stage"
      service   = "http://zenith-mc.zenith-staging.svc.cluster.local:3100"
    }
  }

  access_emails = var.cloudflare_access_emails

  google_oauth_client_id     = var.google_oauth_client_id
  google_oauth_client_secret = var.google_oauth_client_secret
}

output "tunnel_id" {
  value = module.tunnel.tunnel_id
}

output "tunnel_cname" {
  value = module.tunnel.tunnel_cname
}

output "service_urls" {
  value = module.tunnel.service_urls
}
