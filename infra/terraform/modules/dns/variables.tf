variable "zone_id" {
  description = "Cloudflare zone ID for platform domain"
  type        = string
}

variable "server_ip" {
  description = "IP address to point DNS records to"
  type        = string
}

variable "platform_records" {
  description = "Map of platform DNS records to create"
  type = map(object({
    name = string
  }))
  default = {}
}

variable "customer_records" {
  description = "Map of customer DNS records (may span multiple zones)"
  type = map(object({
    zone_id = string
    name    = string
  }))
  default = {}
}
