# =============================================================================
# Cloudflare DNS Records for Zenith Platform
# All records are non-proxied (DNS only) for cert-manager compatibility
# =============================================================================

# ---- freezenith.com zone ----

resource "cloudflare_record" "freezenith_root" {
  zone_id = var.freezenith_zone_id
  name    = "freezenith.com"
  content = var.server_ip
  type    = "A"
  proxied = false
  ttl     = 1
}

resource "cloudflare_record" "freezenith_www" {
  zone_id = var.freezenith_zone_id
  name    = "www"
  content = var.server_ip
  type    = "A"
  proxied = false
  ttl     = 1
}

resource "cloudflare_record" "freezenith_api" {
  zone_id = var.freezenith_zone_id
  name    = "api"
  content = var.server_ip
  type    = "A"
  proxied = false
  ttl     = 1
}

# ---- embermind.app zone ----

resource "cloudflare_record" "embermind_ms" {
  zone_id = var.embermind_zone_id
  name    = "ms"
  content = var.server_ip
  type    = "A"
  proxied = false
  ttl     = 1
}

resource "cloudflare_record" "embermind_cloud" {
  zone_id = var.embermind_zone_id
  name    = "cloud"
  content = var.server_ip
  type    = "A"
  proxied = false
  ttl     = 1
}
