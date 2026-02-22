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

  # Required for S3-compatible providers
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  s3_use_path_style           = true
}

# --- Staging Server (optional — can use existing ghasi or create new) ---

module "staging_server" {
  source = "../modules/k3s-server"
  count  = var.create_server ? 1 : 0

  name           = "zenith-staging"
  server_type    = var.server_type
  location       = var.hetzner_location
  environment    = "staging"
  role           = "all-in-one"
  ssh_public_key = var.ssh_public_key
  ssh_allowed_ips = var.ssh_allowed_ips
}

# --- DNS ---

module "dns" {
  source = "../modules/dns"

  zone_id   = var.freezenith_zone_id
  server_ip = var.create_server ? module.staging_server[0].server_ip : var.existing_server_ip

  platform_records = {
    root      = { name = "staging" }
    api       = { name = "api-staging" }
    demo_ms   = { name = "demo-ms-staging" }
    demo_cloud = { name = "demo-cloud-staging" }
  }

  customer_records = {}
}

# --- S3 Storage ---

module "storage" {
  source = "../modules/storage"

  bucket_name = "zenith-staging"
  environment = "staging"
}
