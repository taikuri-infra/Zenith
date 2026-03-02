terraform {
  required_version = ">= 1.5"

  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.49"
    }
  }

  # TODO: Switch to Hetzner S3 backend when project limits are resolved
  # backend "s3" {
  #   bucket = "zenithstage"
  #   key    = "staging/terraform.tfstate"
  #   endpoints = { s3 = "https://hel1.your-objectstorage.com" }
  #   region                      = "main"
  #   skip_credentials_validation = true
  #   skip_metadata_api_check     = true
  #   skip_region_validation      = true
  #   skip_requesting_account_id  = true
  #   skip_s3_checksum            = true
  #   use_path_style              = true
  # }
}

# --- Providers ---

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

provider "hcloud" {
  token = var.hcloud_token
}

# --- Staging Server ---

module "staging_server" {
  source = "../modules/k3s-server"
  count  = var.create_server ? 1 : 0

  name            = "zenith-staging"
  server_type     = var.server_type
  location        = var.hetzner_location
  environment     = "staging"
  role            = "all-in-one"
  ssh_public_key  = var.ssh_public_key
  ssh_allowed_ips = var.ssh_allowed_ips
}

# --- DNS (stage.freezenith.com) ---

module "dns" {
  source = "../modules/dns"

  zone_id   = var.freezenith_zone_id
  server_ip = var.create_server ? module.staging_server[0].server_ip : var.existing_server_ip

  # Platform services (V1 + V2 additions)
  platform_records = {
    root       = { name = "stage" }
    api        = { name = "api.stage" }
    ms         = { name = "ms.stage" }
    cloud      = { name = "cloud.stage" }
    grafana    = { name = "grafana.stage" }
    prometheus = { name = "prometheus.stage" }
    # --- V2 additions ---
    argocd       = { name = "argocd.stage" }
    keycloak     = { name = "auth.stage" }
    temporal     = { name = "temporal.stage" }
    harbor       = { name = "registry.stage" }
    harbor_hub   = { name = "hub.stage" }
    hubble       = { name = "hubble.stage" }
    tempo        = { name = "tempo.stage" }
    alertmanager = { name = "alerts.stage" }
    # Web dashboard
    app          = { name = "app.stage" }
    # Wildcard for customer apps (*.apps.stage.freezenith.com)
    apps_wildcard = { name = "*.apps.stage" }
  }

  # Customer: embermind on staging
  customer_records = {
    embermind_ms  = { zone_id = var.freezenith_zone_id, name = "embermind-ms.stage" }
    embermind_web = { zone_id = var.freezenith_zone_id, name = "embermind.stage" }
  }
}
