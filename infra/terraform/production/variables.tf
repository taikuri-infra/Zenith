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

variable "management_server_type" {
  description = "Server type for management plane"
  type        = string
  default     = "cx32" # 4 vCPU, 8GB RAM
}

variable "cluster_server_type" {
  description = "Server type for customer workload cluster"
  type        = string
  default     = "cx32" # 4 vCPU, 8GB RAM
}

variable "hetzner_location" {
  description = "Hetzner datacenter location"
  type        = string
  default     = "nbg1"
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

variable "customer_domains" {
  description = "Customer DNS records (keyed by unique name)"
  type = map(object({
    zone_id = string
    name    = string
  }))
  default = {
    embermind_ms = {
      zone_id = "3a9f6cca73ea2653b2bdef6cc6a203b8"
      name    = "ms"
    }
    embermind_cloud = {
      zone_id = "3a9f6cca73ea2653b2bdef6cc6a203b8"
      name    = "cloud"
    }
  }
}
