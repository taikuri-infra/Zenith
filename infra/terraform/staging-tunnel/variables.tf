variable "cloudflare_api_token" {
  description = "Cloudflare API token"
  type        = string
  sensitive   = true
}

variable "cloudflare_account_id" {
  description = "Cloudflare account ID"
  type        = string
  default     = "800cc319b1bda193dbcc3ee55db19a87"
}

variable "freezenith_zone_id" {
  description = "Cloudflare zone ID for freezenith.com"
  type        = string
  default     = "37ac5735b1cf9099ccedd4e038d99465"
}

variable "kubeconfig_path" {
  description = "Path to kubeconfig file"
  type        = string
  default     = "~/.kube/zenith-staging.yaml"
}

variable "cloudflare_access_emails" {
  description = "Email addresses allowed through Cloudflare Access"
  type        = list(string)
  default     = ["babak.dorani@gmail.com", "admin@freezenith.com"]
}

variable "google_oauth_client_id" {
  description = "Google OAuth Client ID for Cloudflare Access"
  type        = string
  default     = ""
}

variable "google_oauth_client_secret" {
  description = "Google OAuth Client Secret for Cloudflare Access"
  type        = string
  sensitive   = true
  default     = ""
}
