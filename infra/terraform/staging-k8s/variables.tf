variable "kubeconfig_path" {
  description = "Path to kubeconfig file for the staging k3s cluster"
  type        = string
  default     = "~/.kube/zenith-staging.yaml"
}

# --- Hetzner Cloud ---

variable "hcloud_token" {
  description = "Hetzner Cloud API token for CSI driver"
  type        = string
  sensitive   = true
}

# --- Domains ---

variable "domain" {
  description = "The parent domain"
  type        = string
  default     = "freezenith.com"
}

variable "cluster_domain" {
  description = "The staging cluster domain"
  type        = string
  default     = "stage.freezenith.com"
}

# --- Registry ---

variable "registry_host" {
  description = "Harbor registry host for deploying internal Zenith apps"
  type        = string
  default     = "registry.stage.freezenith.com"
}

variable "customer_registry_host" {
  description = "Harbor registry host for pro customers"
  type        = string
  default     = "hub.stage.freezenith.com"
}

variable "registry_username" {
  description = "Harbor robot account username"
  type        = string
  sensitive   = true
}

variable "registry_password" {
  description = "Harbor robot account password"
  type        = string
  sensitive   = true
}

variable "zenith_chart_version" {
  description = "Zenith Helm chart version to deploy (shared across all charts)"
  type        = string
  default     = "0.4.0"
}

variable "chart_repository" {
  description = "OCI Helm chart repository URL (leave empty for local chart paths)"
  type        = string
  default     = ""
}

# --- App Secrets ---

variable "jwt_secret" {
  description = "JWT signing secret"
  type        = string
  sensitive   = true
}

variable "admin_email" {
  description = "Admin user email"
  type        = string
  sensitive   = true
}

variable "admin_password" {
  description = "Admin user password (also used for Grafana, Harbor)"
  type        = string
  sensitive   = true
}

variable "github_webhook_secret" {
  description = "HMAC secret for verifying GitHub webhook signatures"
  type        = string
  sensitive   = true
  default     = ""
}

variable "secrets_encryption_key" {
  description = "64-char hex (32 bytes) AES-256-GCM key for encrypting app secrets"
  type        = string
  sensitive   = true
  default     = ""
}

variable "resend_api_key" {
  description = "Resend API key for email verification"
  type        = string
  sensitive   = true
  default     = ""
}

variable "google_client_id" {
  description = "Google OAuth client ID for login"
  type        = string
  sensitive   = true
  default     = ""
}

# --- V2: S3 / Object Storage (Hetzner) ---

variable "keycloak_db_storage_size" {
  description = "Storage size for dedicated Keycloak CNPG cluster"
  type        = string
  default     = "10Gi"
}

variable "free_db_storage_size" {
  description = "Storage size for shared free-tier CNPG cluster"
  type        = string
  default     = "10Gi"
}

variable "s3_access_key" {
  description = "Hetzner S3 access key for CNPG WAL archiving, Harbor, Velero"
  type        = string
  sensitive   = true
  default     = ""
}

variable "s3_secret_key" {
  description = "Hetzner S3 secret key"
  type        = string
  sensitive   = true
  default     = ""
}

variable "s3_endpoint" {
  description = "Hetzner S3 endpoint URL"
  type        = string
  default     = "https://fsn1.your-objectstorage.com"
}

# --- V2: Cloudflare ---

variable "cloudflare_api_token" {
  description = "Cloudflare API token for external-dns and cert-manager DNS-01"
  type        = string
  sensitive   = true
  default     = ""
}

# --- V2: Keycloak ---

variable "keycloak_db_password" {
  description = "Keycloak database password"
  type        = string
  sensitive   = true
  default     = ""
}

variable "keycloak_admin_password" {
  description = "Keycloak admin console password"
  type        = string
  sensitive   = true
  default     = ""
}

# --- V2: Temporal ---

variable "temporal_db_user" {
  description = "Temporal database user"
  type        = string
  sensitive   = true
  default     = "temporal"
}

variable "temporal_db_password" {
  description = "Temporal database password"
  type        = string
  sensitive   = true
  default     = ""
}

# --- V2: ArgoCD / GitOps ---

variable "github_token" {
  description = "GitHub personal access token for ArgoCD repo access"
  type        = string
  sensitive   = true
  default     = ""
}

# --- Cloudflare Zero Trust Tunnel ---

variable "freezenith_zone_id" {
  description = "Cloudflare zone ID for freezenith.com"
  type        = string
  default     = "37ac5735b1cf9099ccedd4e038d99465"
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
  default     = "800cc319b1bda193dbcc3ee55db19a87"
}

variable "enable_cloudflare_tunnel" {
  description = "Enable Cloudflare Zero Trust Tunnel for monitoring access"
  type        = bool
  default     = false
}

variable "cloudflare_access_emails" {
  description = "Email addresses allowed through Cloudflare Access"
  type        = list(string)
  default     = ["babak.dorani@gmail.com", "admin@freezenith.com"]
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "staging"
}
