# Cloudflare WAF Rules for Zenith Shared Tiers

These WAF rules are configured in the Cloudflare dashboard (or via Terraform
cloudflare provider) and protect the shared infrastructure tiers (Free, Pro, Team).

Enterprise customers with dedicated clusters get their own Cloudflare zone.

## Rules

### 1. Rate Limiting (Applied to *.freezenith.com)

| Rule | Threshold | Action | Period |
|------|-----------|--------|--------|
| API rate limit | 100 req/min per IP | Challenge | 1 min |
| Login brute force | 10 req/min per IP to `/api/v1/auth/login` | Block | 5 min |
| Registration abuse | 5 req/min per IP to `/api/v1/auth/register` | Challenge | 10 min |
| Webhook abuse | 50 req/min per IP to `/api/v1/internal/*` | Block | 5 min |

### 2. Managed WAF Rules

| Ruleset | Status |
|---------|--------|
| Cloudflare Managed Ruleset | Enabled |
| OWASP Core Ruleset | Enabled (Paranoia Level 2) |
| Cloudflare Bot Management | Enabled (Free/Pro: Definitely Automated → Block) |

### 3. Custom WAF Rules

```
# Block known bad user agents
(http.user_agent contains "sqlmap") or
(http.user_agent contains "nikto") or
(http.user_agent contains "nmap") or
(http.user_agent contains "masscan")
→ Action: Block

# Block requests with SQL injection patterns in query strings
(http.request.uri.query contains "UNION SELECT") or
(http.request.uri.query contains "OR 1=1") or
(http.request.uri.query contains "DROP TABLE")
→ Action: Block

# Block oversized request bodies (DDoS protection)
(http.request.body.size gt 10000000)
→ Action: Block

# Geo-blocking (optional, disabled by default)
# (ip.geoip.country in {"CN" "RU" "KP"})
# → Action: Challenge
```

### 4. Page Rules

| Pattern | Setting |
|---------|---------|
| `api.freezenith.com/*` | SSL: Full (Strict), Cache: Bypass |
| `app.freezenith.com/*` | SSL: Full (Strict), Cache: Bypass |
| `*.apps.freezenith.com/*` | SSL: Full (Strict), Cache: Standard |

### 5. Terraform Implementation

```hcl
# In infra/terraform/modules/dns/cloudflare-waf.tf

resource "cloudflare_ruleset" "zenith_waf" {
  zone_id = var.cloudflare_zone_id
  name    = "Zenith WAF Rules"
  kind    = "zone"
  phase   = "http_ratelimit"

  rules {
    action = "block"
    expression = "(http.request.uri.path contains \"/api/v1/auth/login\" and rate(1m) > 10)"
    description = "Block login brute force"
  }

  rules {
    action = "challenge"
    expression = "(rate(1m) > 100)"
    description = "Rate limit API requests"
  }
}

resource "cloudflare_ruleset" "zenith_custom_waf" {
  zone_id = var.cloudflare_zone_id
  name    = "Zenith Custom WAF"
  kind    = "zone"
  phase   = "http_request_firewall_custom"

  rules {
    action = "block"
    expression = "(http.user_agent contains \"sqlmap\") or (http.user_agent contains \"nikto\")"
    description = "Block security scanner user agents"
  }
}
```
