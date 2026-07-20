# FreeZenith subdomain-registration service

The only place the Cloudflare token lives. Customer installers (`zen install
--edition compose --free-domain`) POST their public IP; this service creates
`<slug>.apps.freezenith.com -> <ip>` and returns just the hostname. **The
customer's box never sees the token** — only its own subdomain name.

## API

- `GET /health` → `{"status":"healthy"}`
- `POST /register` `{"ip":"1.2.3.4"}` → `{"hostname":"swift-otter-3e0b.apps.freezenith.com"}`
  (falls back to the request source IP if `ip` is omitted; rate-limited per source IP)
- `POST /release` `{"hostname":"…apps.freezenith.com"}` → `{"status":"released"}`
  (only deletes records inside the operated base domain)

## Config (env)

| Var | Default | Notes |
|-----|---------|-------|
| `CLOUDFLARE_DNS_TOKEN` | *(required)* | Cloudflare token, scoped to **DNS:Edit on the target zone only** |
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

## Security notes

- Use a **least-privilege** token: `Zone → DNS → Edit`, scoped to only the
  zone(s) this service manages. Do not reuse an all-zones token.
- Rate limiting is per source IP (10/hour by default). Front it with the usual
  edge protections (Cloudflare in front of `register.freezenith.com`).
- A future cleanup job should reclaim records for installs that never come up
  healthy; not implemented yet.
