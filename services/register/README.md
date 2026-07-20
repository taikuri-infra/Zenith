# FreeZenith subdomain-registration service

The only place the Cloudflare token lives. Customer installers (`zen install
--edition compose --free-domain`) POST their public IP; this service creates
`<slug>.apps.freezenith.com -> <ip>` and returns just the hostname. **The
customer's box never sees the token** — only its own subdomain name.

## API

All mutating endpoints require `Authorization: Bearer $INSTALL_TOKEN`.

- `GET /health` → `{"status":"healthy"}`
- `POST /register` `{"ip":"1.2.3.4"}` → `{"hostname":"swift-otter-3e0b.apps.freezenith.com"}`
  (the IP must be **public**; falls back to the caller IP if omitted; rate-limited per source IP)
- `POST /release` `{"hostname":"…apps.freezenith.com"}` → `{"status":"released"}`
  (only deletes well-formed records inside the operated base domain; rate-limited)

## Config (env)

| Var | Default | Notes |
|-----|---------|-------|
| `CLOUDFLARE_DNS_TOKEN` | *(required)* | Cloudflare token, scoped to **DNS:Edit on the target zone only** |
| `INSTALL_TOKEN` | *(required for register/release)* | shared token installers must present; **unset = /register + /release disabled (fail closed)** |
| `TRUSTED_PROXIES` | *(empty)* | comma-separated CIDRs; only from these peers is `CF-Connecting-IP` honored (else the TCP peer is used) |
| `BASE_DOMAIN` | `apps.freezenith.com` | subdomains are created under this |
| `ZONE_NAME` | `freezenith.com` | the Cloudflare zone that owns `BASE_DOMAIN` |
| `PORT` | `8080` | listen port |

## Run / deploy

```bash
CLOUDFLARE_DNS_TOKEN=cfut_… go run .          # local
docker build -t register . && \
  docker run -e CLOUDFLARE_DNS_TOKEN=cfut_… -p 8080:8080 register
```

Deploy behind TLS at `register.freezenith.com`. Because the customer's install
only needs the hostname back, the token stays server-side here — a compromised
customer box can never leak it.

## Security

Enforced:
- **Auth, fail-closed:** `/register` and `/release` require `Authorization: Bearer
  $INSTALL_TOKEN`; if `INSTALL_TOKEN` is unset they are disabled entirely.
- **Public-IP only:** records can only point at global-unicast IPs — loopback,
  private, link-local, and multicast are rejected (no pointing a `freezenith.com`
  name at an internal target).
- **Injection-safe:** hostnames are validated to DNS-safe characters and all
  values are URL-escaped into the Cloudflare API — no query injection.
- **Spoof-resistant source IP:** `X-Forwarded-For` is ignored; `CF-Connecting-IP`
  is honored only when the peer is a configured `TRUSTED_PROXIES` CIDR.
- **Rate limited** per source IP on both endpoints (10/hour default).
- Use a **least-privilege** Cloudflare token (`Zone → DNS → Edit`, only the
  managed zone) — never an all-zones token.

Still TODO before high-scale public exposure:
- **Proof-of-possession of the IP** (an HTTP-01-style challenge served from the
  installer's box) so records are only created for IPs the caller controls,
  beyond the shared `INSTALL_TOKEN` gate.
- A **cleanup job** to reclaim records for installs that never become healthy.
