# --- Cloud Providers ---

variable "cloudflare_api_token" {
  description = "Cloudflare API token"
  type        = string
  sensitive   = true
}

variable "hcloud_token" {
  description = "Hetzner Cloud API token"
  type        = string
  sensitive   = true
  default     = ""
}

variable "hetzner_s3_access_key" {
  description = "Hetzner Object Storage access key"
  type        = string
  sensitive   = true
  default     = ""
}

variable "hetzner_s3_secret_key" {
  description = "Hetzner Object Storage secret key"
  type        = string
  sensitive   = true
  default     = ""
}

# --- Server Config ---

variable "create_server" {
  description = "Create a new Hetzner server for staging, or use existing (e.g. ghasi)"
  type        = bool
  default     = false
}

variable "existing_server_ip" {
  description = "IP of existing server (used when create_server = false)"
  type        = string
  default     = "161.35.82.211" # ghasi
}

variable "server_type" {
  description = "Hetzner server type"
  type        = string
  default     = "cx22"
}

variable "hetzner_location" {
  description = "Hetzner datacenter location"
  type        = string
  default     = "nbg1"
}

variable "ssh_public_key" {
  description = "SSH public key content"
  type        = string
  default     = ""
}

variable "ssh_allowed_ips" {
  description = "IPs allowed for SSH access"
  type        = list(string)
  default     = ["0.0.0.0/0", "::/0"]
}

# --- DNS ---

variable "freezenith_zone_id" {
  description = "Cloudflare zone ID for freezenith.com"
  type        = string
  default     = "37ac5735b1cf9099ccedd4e038d99465"
}
