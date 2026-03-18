# =============================================================================
# Cloudflare Zero Trust Tunnel
# =============================================================================
#
# Creates a Cloudflare Tunnel with:
#   1. Tunnel resource + secret
#   2. cloudflared Deployment in K8s
#   3. DNS CNAME records pointing to the tunnel
#   4. Cloudflare Access Application + Policy per service
#
# Architecture:
#   Internet → Cloudflare Edge → Tunnel → cloudflared pod → K8s Service
#   No public ports exposed. Zero Trust access policies enforce auth.
#
# =============================================================================

terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0"
    }
  }
}

# --- Tunnel ---

resource "random_id" "tunnel_secret" {
  byte_length = 32
}

resource "cloudflare_tunnel" "this" {
  account_id = var.account_id
  name       = var.tunnel_name
  secret     = random_id.tunnel_secret.b64_std
}

# --- DNS CNAME records (each service → tunnel) ---

resource "cloudflare_record" "tunnel_cname" {
  for_each = var.manage_dns ? var.services : {}

  zone_id = var.zone_id
  name    = each.value.subdomain
  type    = "CNAME"
  value   = "${cloudflare_tunnel.this.id}.cfargotunnel.com"
  proxied = true
  ttl     = 1 # Auto (proxied)
}

# --- Google OAuth Identity Provider (optional) ---

resource "cloudflare_access_identity_provider" "google" {
  count = var.google_oauth_client_id != "" ? 1 : 0

  account_id = var.account_id
  name       = "${var.tunnel_name}-google"
  type       = "google"

  config {
    client_id     = var.google_oauth_client_id
    client_secret = var.google_oauth_client_secret
  }
}

# --- Cloudflare Access Application + Policy ---

resource "cloudflare_access_application" "this" {
  for_each = var.services

  account_id       = var.account_id
  name             = "${var.tunnel_name}-${each.key}"
  domain           = "${each.value.subdomain}.${var.domain}"
  type             = "self_hosted"
  session_duration = var.session_duration

  # Force Google OAuth only (no OTP fallback)
  dynamic "saas_app" {
    for_each = []
    content {}
  }
}

resource "cloudflare_access_policy" "allow_emails" {
  for_each = length(var.access_emails) > 0 ? var.services : {}

  account_id     = var.account_id
  application_id = cloudflare_access_application.this[each.key].id
  name           = "Allow ${each.key} by email"
  decision       = "allow"
  precedence     = 1

  include {
    email = var.access_emails
  }
}

resource "cloudflare_access_policy" "allow_domains" {
  for_each = length(var.access_email_domains) > 0 ? var.services : {}

  account_id     = var.account_id
  application_id = cloudflare_access_application.this[each.key].id
  name           = "Allow ${each.key} by domain"
  decision       = "allow"
  precedence     = length(var.access_emails) > 0 ? 2 : 1

  include {
    email_domain = var.access_email_domains
  }
}

# --- Kubernetes Namespace ---

resource "kubernetes_namespace_v1" "tunnel" {
  metadata {
    name = var.namespace
    labels = {
      "app.kubernetes.io/managed-by" = "terraform"
      "app.kubernetes.io/component"  = "cloudflare-tunnel"
    }
  }
}

# --- Tunnel credentials secret ---

resource "kubernetes_secret_v1" "tunnel_credentials" {
  metadata {
    name      = "cloudflared-credentials"
    namespace = kubernetes_namespace_v1.tunnel.metadata[0].name
  }

  data = {
    "credentials.json" = jsonencode({
      AccountTag   = var.account_id
      TunnelID     = cloudflare_tunnel.this.id
      TunnelName   = var.tunnel_name
      TunnelSecret = random_id.tunnel_secret.b64_std
    })
  }
}

# --- Tunnel config ---

resource "kubernetes_config_map_v1" "tunnel_config" {
  metadata {
    name      = "cloudflared-config"
    namespace = kubernetes_namespace_v1.tunnel.metadata[0].name
  }

  data = {
    "config.yaml" = yamlencode({
      tunnel           = cloudflare_tunnel.this.id
      credentials-file = "/etc/cloudflared/credentials/credentials.json"
      metrics          = "0.0.0.0:2000"
      no-autoupdate    = true
      ingress = concat(
        [for key, svc in var.services : {
          hostname = "${svc.subdomain}.${var.domain}"
          service  = svc.service
        }],
        [{ service = "http_status:404" }]
      )
    })
  }
}

# --- cloudflared Deployment ---

resource "kubernetes_deployment_v1" "cloudflared" {
  metadata {
    name      = "cloudflared"
    namespace = kubernetes_namespace_v1.tunnel.metadata[0].name
    labels = {
      app = "cloudflared"
    }
  }

  spec {
    replicas = var.cloudflared_replicas

    selector {
      match_labels = {
        app = "cloudflared"
      }
    }

    template {
      metadata {
        labels = {
          app = "cloudflared"
        }
      }

      spec {
        container {
          name  = "cloudflared"
          image = var.cloudflared_image
          args  = ["tunnel", "--config", "/etc/cloudflared/config/config.yaml", "run"]

          port {
            container_port = 2000
            name           = "metrics"
          }

          resources {
            requests = {
              cpu    = "25m"
              memory = "64Mi"
            }
            limits = {
              memory = "128Mi"
            }
          }

          volume_mount {
            name       = "credentials"
            mount_path = "/etc/cloudflared/credentials"
            read_only  = true
          }

          volume_mount {
            name       = "config"
            mount_path = "/etc/cloudflared/config"
            read_only  = true
          }

          liveness_probe {
            http_get {
              path = "/ready"
              port = 2000
            }
            initial_delay_seconds = 10
            period_seconds        = 10
          }
        }

        volume {
          name = "credentials"
          secret {
            secret_name = kubernetes_secret_v1.tunnel_credentials.metadata[0].name
          }
        }

        volume {
          name = "config"
          config_map {
            name = kubernetes_config_map_v1.tunnel_config.metadata[0].name
          }
        }
      }
    }
  }
}
