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
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# --- Providers ---

provider "cloudflare" {
  api_token = var.cloudflare_api_token
}

provider "hcloud" {
  token = var.hcloud_token
}

# Hetzner S3-compatible Object Storage
provider "aws" {
  region     = "eu-central-1"
  access_key = var.hetzner_s3_access_key
  secret_key = var.hetzner_s3_secret_key

  endpoints {
    s3 = "https://fsn1.your-objectstorage.com"
  }

  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  s3_use_path_style           = true
}

# =============================================
# Server 1: Management Plane
# Runs: Zenith API, Dashboard, MC, Landing, PostgreSQL, CAPI
# =============================================

module "management" {
  source = "../modules/k3s-server"

  name            = "zenith-prod-management"
  server_type     = var.management_server_type
  location        = var.hetzner_location
  environment     = "production"
  role            = "management"
  ssh_public_key  = var.ssh_public_key
  ssh_allowed_ips = var.ssh_allowed_ips

  extra_labels = {
    "zenith.dev/plane" = "management"
  }
}

# =============================================
# Server 2: Customer Cluster
# Runs: Customer apps, databases, KEDA, workloads
# =============================================

module "cluster" {
  source = "../modules/k3s-server"

  name            = "zenith-prod-cluster"
  server_type     = var.cluster_server_type
  location        = var.hetzner_location
  environment     = "production"
  role            = "cluster"
  ssh_public_key  = var.ssh_public_key
  ssh_allowed_ips = var.ssh_allowed_ips

  extra_labels = {
    "zenith.dev/plane" = "workload"
  }
}

# =============================================
# DNS — Platform domains (freezenith.com)
# =============================================

module "platform_dns" {
  source = "../modules/dns"

  zone_id   = var.freezenith_zone_id
  server_ip = module.management.server_ip

  platform_records = {
    root       = { name = "@" }
    www        = { name = "www" }
    api        = { name = "api" }
    demo_ms    = { name = "demo-ms" }
    demo_cloud = { name = "demo-cloud" }
  }

  customer_records = {}
}

# =============================================
# DNS — Customer domains (embermind.app + others)
# =============================================

module "customer_dns" {
  source = "../modules/dns"

  zone_id   = var.freezenith_zone_id # not used for customer records
  server_ip = module.management.server_ip

  platform_records = {}

  customer_records = {
    for k, v in var.customer_domains : k => {
      zone_id = v.zone_id
      name    = v.name
    }
  }
}

# =============================================
# S3 Storage
# =============================================

module "storage" {
  source = "../modules/storage"

  bucket_name = "zenith-production"
  environment = "production"
}
