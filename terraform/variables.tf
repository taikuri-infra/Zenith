variable "cloudflare_api_token" {
  description = "Cloudflare API token with DNS edit permissions"
  type        = string
  sensitive   = true
}

variable "server_ip" {
  description = "IP address of the Zenith server"
  type        = string
  default     = "161.35.82.211"
}

variable "freezenith_zone_id" {
  description = "Cloudflare zone ID for freezenith.com"
  type        = string
  default     = "37ac5735b1cf9099ccedd4e038d99465"
}

variable "embermind_zone_id" {
  description = "Cloudflare zone ID for embermind.app"
  type        = string
  default     = "3a9f6cca73ea2653b2bdef6cc6a203b8"
}
