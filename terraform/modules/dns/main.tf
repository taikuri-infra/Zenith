terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}

# Platform DNS records
resource "cloudflare_record" "platform" {
  for_each = var.platform_records

  zone_id = var.zone_id
  name    = each.value.name
  content = var.server_ip
  type    = "A"
  ttl     = 1
  proxied = false
}

# Customer DNS records
resource "cloudflare_record" "customer" {
  for_each = var.customer_records

  zone_id = each.value.zone_id
  name    = each.value.name
  content = var.server_ip
  type    = "A"
  ttl     = 1
  proxied = false
}
