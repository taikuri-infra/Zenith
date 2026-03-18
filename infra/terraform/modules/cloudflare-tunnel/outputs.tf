# =============================================================================
# Cloudflare Zero Trust Tunnel — Outputs
# =============================================================================

output "tunnel_id" {
  description = "Cloudflare Tunnel ID"
  value       = cloudflare_tunnel.this.id
}

output "tunnel_cname" {
  description = "Tunnel CNAME (for manual DNS)"
  value       = "${cloudflare_tunnel.this.id}.cfargotunnel.com"
}

output "service_urls" {
  description = "Public URLs for each exposed service"
  value = {
    for key, svc in var.services : key => "https://${svc.subdomain}.${var.domain}"
  }
}
