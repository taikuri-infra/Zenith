terraform {
  required_providers {
    zenith = {
      source  = "dotechhq/zenith"
      version = "~> 0.1"
    }
  }
}

provider "zenith" {
  api_url   = "https://api.stage.freezenith.com"
  api_token = var.zenith_token
}

variable "zenith_token" {
  type      = string
  sensitive = true
}

# Deploy a web app from a Git repo
resource "zenith_app" "backend" {
  name          = "my-api"
  deploy_source = "git"
  repo_url      = "https://github.com/myorg/my-api"
  branch        = "main"
  port          = 3000
  app_type      = "web"
}

# Deploy a worker from a Docker image
resource "zenith_app" "worker" {
  name          = "queue-worker"
  deploy_source = "image"
  image_url     = "registry.stage.freezenith.com/myorg/worker:latest"
  app_type      = "worker"
  command       = "./worker --queue=default"
}

# Provision a PostgreSQL database
resource "zenith_database" "main_db" {
  name   = "myapp-db"
  engine = "postgresql"
}

# Provision a Redis instance
resource "zenith_database" "cache" {
  name   = "myapp-cache"
  engine = "redis"
}

# Create a storage bucket
resource "zenith_storage" "uploads" {
  name   = "user-uploads"
  access = "private"
}

resource "zenith_storage" "assets" {
  name   = "public-assets"
  access = "public"
}

# Set up an API Gateway
resource "zenith_gateway" "api" {
  name = "main-gateway"

  routes {
    name    = "backend-api"
    path    = "/api/*"
    methods = ["GET", "POST", "PUT", "DELETE"]
    app_id  = zenith_app.backend.id
    auth    = "jwt"
  }

  routes {
    name         = "public-health"
    path         = "/health"
    methods      = ["GET"]
    app_id       = zenith_app.backend.id
    strip_prefix = true
    auth         = "none"
  }
}

# Attach a custom domain
resource "zenith_domain" "api_domain" {
  app_id = zenith_app.backend.id
  domain = "api.mycompany.com"
}

# Outputs
output "backend_url" {
  value = zenith_app.backend.url
}

output "db_connection" {
  value     = zenith_database.main_db.connection_string
  sensitive = true
}

output "gateway_endpoint" {
  value = zenith_gateway.api.endpoint
}

output "domain_status" {
  value = zenith_domain.api_domain.status
}
