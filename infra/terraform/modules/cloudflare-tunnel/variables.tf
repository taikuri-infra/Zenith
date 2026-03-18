# =============================================================================
# Cloudflare Zero Trust Tunnel — Variables
# =============================================================================

variable "account_id" {
  description = "Cloudflare account ID"
  type        = string
}

variable "zone_id" {
  description = "Cloudflare zone ID for DNS records"
  type        = string
}

variable "tunnel_name" {
  description = "Name for the Cloudflare tunnel"
  type        = string
}

variable "environment" {
  description = "Environment name (staging, production)"
  type        = string
}

variable "namespace" {
  description = "Kubernetes namespace for cloudflared deployment"
  type        = string
  default     = "cloudflare-tunnel"
}

variable "cloudflared_image" {
  description = "cloudflared Docker image"
  type        = string
  default     = "cloudflare/cloudflared:2024.12.2"
}

variable "cloudflared_replicas" {
  description = "Number of cloudflared replicas"
  type        = number
  default     = 1
}

variable "services" {
  description = "Map of services to expose through the tunnel"
  type = map(object({
    subdomain = string # e.g. "grafana.stage" → grafana.stage.freezenith.com
    service   = string # e.g. "http://kube-prometheus-stack-grafana.monitoring.svc.cluster.local:80"
  }))
}

variable "access_emails" {
  description = "List of email addresses allowed through Cloudflare Access"
  type        = list(string)
  default     = []
}

variable "access_email_domains" {
  description = "List of email domains allowed through Cloudflare Access (e.g. freezenith.com)"
  type        = list(string)
  default     = []
}

variable "session_duration" {
  description = "Access session duration"
  type        = string
  default     = "24h"
}

variable "manage_dns" {
  description = "Create DNS CNAME records for services. Disable if DNS is managed elsewhere."
  type        = bool
  default     = true
}

variable "domain" {
  description = "Root domain for service hostnames"
  type        = string
  default     = "freezenith.com"
}

variable "google_oauth_client_id" {
  description = "Google OAuth Client ID for Cloudflare Access login"
  type        = string
  default     = ""
}

variable "google_oauth_client_secret" {
  description = "Google OAuth Client Secret for Cloudflare Access login"
  type        = string
  sensitive   = true
  default     = ""
}
