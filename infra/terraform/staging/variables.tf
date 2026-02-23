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
}

# --- Server Config ---

variable "create_server" {
  description = "Create a new Hetzner server for staging"
  type        = bool
  default     = true
}

variable "existing_server_ip" {
  description = "IP of existing server (used when create_server = false)"
  type        = string
  default     = ""
}

variable "server_type" {
  description = "Hetzner server type (cx23 = 2 vCPU / 4GB)"
  type        = string
  default     = "cx23"
}

variable "hetzner_location" {
  description = "Hetzner datacenter location"
  type        = string
  default     = "hel1"
}

variable "ssh_public_key" {
  description = "SSH public key content"
  type        = string
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
