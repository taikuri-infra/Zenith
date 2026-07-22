# Three-Rung HTTPS for Compose Installs (Internal / Air-Gapped Domains)

**Date:** 2026-07-22
**Status:** Approved — implementing
**Scope:** `zen install --edition compose` only. The cloud/k8s edition is untouched.

## Problem

The compose installer can only give a box HTTPS through Let's Encrypt **HTTP-01**, which
requires two things the box may not have:

1. A **publicly-resolvable** domain (LE refuses `.lan`, `.local`, `.internal`, bare IPs).
2. Inbound reachability on **port 80** so LE's servers can answer the challenge.

A box that has outbound internet but sits behind NAT/firewall, or uses an internal-only
name, therefore gets **no HTTPS today** — even though the install itself works (images
pull fine over outbound internet).

This is **not** an offline/air-gapped *install* (that needs bundled images — out of scope).
It is purely about **getting the best possible TLS cert for the box's actual situation**.

## The three rungs

For an **own-domain** compose install the wizard asks one clearly-worded question and sets
`Config.TLSMode`:

| Rung | Wizard framing | `TLSMode` | Cert mechanism |
|------|----------------|-----------|----------------|
| 1 | "Public IP, ports 80/443 open to the internet" | `http01` | LE HTTP-01 (**existing behaviour**) |
| 2 | "Behind NAT/firewall, but I manage the domain's DNS on Cloudflare" | `dns01` | LE DNS-01 (Cloudflare) |
| 3 | "Internal or offline server (e.g. `zenith.lan`)" | `selfsigned` | Local CA + leaf; operator imports `zenith-ca.crt` |

`localhost` (no TLS) and the free `*.apps.freezenith.com` subdomain path (always `http01`)
are unchanged. Detection is **by asking the user**, not auto-probing — with copy written so
a junior can answer correctly ("A cloud VPS = yes; a machine on your home/office network =
usually no").

## Architecture

### Cert resolver moves to the entrypoint (the key refactor)

Traefik ACME resolvers are **static config only**, and a router that names a non-existent
resolver is a startup error — so a self-signed router cannot carry `tls.certresolver=le`.
To make one code path serve all three rungs:

- **No router names a resolver anywhere.** The dashboard, API, and every deployed user app
  set only `tls=true` + `entrypoints=websecure`.
- The **entrypoint** `websecure` carries the default resolver (`http.tls.certResolver: le`)
  in ACME modes, and **nothing** in self-signed mode (Traefik then serves the default
  certificate — our leaf).

Consequences:
- `docker-compose.yml`: drop `tls.certresolver=le` from the `zenith-web`/`zenith-api` labels
  (keep `tls=true`). Replace the inline `command:` ACME/entrypoint args with a mounted,
  installer-generated `traefik/traefik.yml`. Add `CF_DNS_API_TOKEN` env + a `traefik/`
  bind mount. A default `traefik/traefik.yml` (http01) + `traefik/dynamic/.gitkeep` are
  committed so a standalone `docker compose --profile tls up` still works.
- `services/api/internal/services/deploy/docker_deployer.go`: emit `tls=true` instead of
  `tls.certresolver=le` for deployed apps. Mode-independent — the deployer no longer needs
  to know the TLS mode; the entrypoint decides. Update `docker_deployer_labels_test.go`.

### Installer-generated Traefik config

A new **`Configure TLS`** step runs for every non-localhost install (including free
subdomain). It writes `<dir>/traefik/traefik.yml` for the mode:

- `http01`: `certificatesResolvers.le.acme` with `httpChallenge.entryPoint: web`,
  `email`, `storage: /letsencrypt/acme.json`; entrypoint default `certResolver: le`.
- `dns01`: same but `dnsChallenge.provider: cloudflare`; token supplied to the Traefik
  container via `CF_DNS_API_TOKEN` (written into `.env`).
- `selfsigned`: no resolver; a `traefik/dynamic/certs.yml` file-provider store sets the
  default certificate to the generated leaf.

The email is written literally into `traefik.yml` (Traefik static YAML does not expand
`${ENV}`), sourced from `AdminEmail` (falls back to `admin@<domain>`).

### Cert generation (Go, not openssl-on-box)

New package `cli/internal/tlscert`:
- `GenerateCA() (certPEM, keyPEM []byte, err error)` — ECDSA P-256, `CN=Zenith Local CA`,
  `IsCA`, ~10-year validity.
- `GenerateLeaf(caCertPEM, caKeyPEM []byte, domain string) (certPEM, keyPEM []byte, err error)`
  — SANs `domain` **and** `*.domain` (so deployed app subdomains are covered by the same CA
  import), ~10-year validity so no renewal is needed.

Generating in Go (vs shelling `openssl` on the target) removes a target dependency and is
unit-testable. The leaf+key and the CA cert are pushed to the box (base64, `umask 077`);
the CA **key** is kept owner-only in the install dir for possible re-issue.

### Operator hand-off (self-signed)

After a self-signed install the installer fetches `zenith-ca.crt` back to the operator's
machine (`./zenith-ca.crt`) and prints plain-language import instructions for
macOS / Windows / Linux / iOS / Android — the "explain it for any seniority" requirement.

### DNS step gating

`Configure DNS` (public A-record guidance) runs **only** for `http01` own-domain installs.
`dns01` needs no public A record for issuance (Traefik does the `_acme-challenge` TXT dance);
`selfsigned` uses no public DNS at all.

## Config / CLI surface

- `Config.TLSMode string` with constants `TLSHTTP01="http01"`, `TLSDNS01="dns01"`,
  `TLSSelfSigned="selfsigned"` (default `http01`).
- `install` flag `--tls-mode http01|dns01|selfsigned`. `dns01` reuses `--dns-token`
  (Cloudflare) for both the DNS-01 challenge token and any record management.
- Wizard: own-domain branch gains the three-rung question; `dns01` collects the Cloudflare
  token; `selfsigned` collects nothing extra.

## Testing

- `tlscert`: leaf verifies against the CA pool; SANs include `domain` + `*.domain`; CA has
  `IsCA`; validity > 9 years.
- `traefik.yml` generation: per-mode assertions (resolver present/absent, `httpChallenge`
  vs `dnsChallenge`, self-signed has file-provider default cert, no resolver).
- `needsCustomDNS` / step insertion: DNS step only for `http01`; `Configure TLS` for all
  non-localhost; `Generate certificate` only for `selfsigned`.
- `buildComposeEnv`: `CF_DNS_API_TOKEN` present iff `dns01`.
- Deployer label test updated to expect `tls=true`.
- `go build`, `go vet`, `go test ./...` in `cli` and `services/api`.
- Dry-run for all three modes; best-effort local self-signed run on a Lima VM.

## Out of scope

- Offline/air-gapped *install* (bundled images, local registry).
- Non-Cloudflare DNS-01 providers.
- Automated cross-machine CA distribution (we hand over one file + instructions).
