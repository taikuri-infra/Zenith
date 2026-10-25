# Zenith

**Open-source PaaS for Kubernetes. Deploy any container image on your own server.**

Zenith is a self-hosted platform-as-a-service that gives you Heroku-like simplicity on your own infrastructure. Point it at a container image and get a running app, with databases, object storage and automatic HTTPS built in. It runs on k3s (or any Kubernetes) and manages databases through Kubernetes operators.

## Quick Start

On a fresh Linux server (root, ports 80 and 443 free, about 4 GB of RAM):

```bash
curl -fsSL https://raw.githubusercontent.com/taikuri-infra/Zenith/main/infra/scripts/install-k3s.sh | sudo bash
```

The installer sets up k3s, cert-manager, the database operators (CloudNativePG, mariadb-operator,
redis-operator) and Zenith, then prints your dashboard URL and the admin credentials. Open the URL and log in.
Set `ZENITH_DOMAIN=zenith.example.com` and `ZENITH_EMAIL=you@example.com` to serve it on your own domain
(the name must resolve to the server); without them it uses `<public-ip>.sslip.io`.

Get the `zen` CLI from the [releases page](https://github.com/taikuri-infra/Zenith/releases):
`zen login` against your dashboard URL to deploy from your terminal, `zen backup` and `zen backup restore`
to protect your data, `zen upgrade` to move to a new release. To add capacity, join more k3s nodes
with the token in `/var/lib/rancher/k3s/server/node-token`.

What works today, and what does not yet, is tracked in [docs/alpha/core-audit.md](docs/alpha/core-audit.md).

## Features

- **App Deployment** — Deploy any container image, or import an existing Docker Compose file as a set of apps.
- **Databases** — Managed PostgreSQL, MySQL, and Redis per app.
- **Object Storage** — S3-compatible storage buckets.
- **Built-in Auth** — Per-app user authentication out of the box.
- **Custom Domains** — Attach your own domains with automatic TLS.
- **Secrets Management** — Encrypted environment variables and secrets.
- **Deploy Previews** — Preview branches before merging.
- **Releases** — Versioned deployments with instant rollback.
- **SSE Events** — Real-time build logs and deployment status.
- **Role-Based Access** — Fine-grained permissions and custom roles.
- **API Keys** — Programmatic access to the platform API.
- **Webhooks** — Get notified on deployment events.
- **SSO** — SAML and OIDC single sign-on.
- **MFA** — Two-factor authentication for platform accounts.
- **IP Allowlisting** — Restrict dashboard access by IP.
- **Compliance Dashboard** — Security posture overview.
- **Audit Log** — Track all platform actions with CSV/JSON export.

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Dashboard   │────▶│   API (Go)   │────▶│  PostgreSQL  │
│  (Next.js)   │     │   :8080      │     │   :5432      │
│  :3000       │     │              │──┐  └──────────────┘
└──────────────┘     └──────────────┘  │  ┌──────────────┐
                                        └─▶│ RustFS (S3)  │
                                           │  :9000       │
                                           └──────────────┘
```

- **API** — Go server (Fiber framework). Handles auth, app management, deployments, databases, and all platform operations.
- **Dashboard** — Next.js web UI. Full management interface for apps, databases, domains, settings, and monitoring.
- **PostgreSQL** — Persistent storage for users, apps, deployments, and platform state. Falls back to in-memory stores when no database is configured.
- **RustFS** — Self-hosted, S3-compatible object storage for app buckets and backups. Runs inside the cluster, so no external cloud storage account is required.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | *(required)* | Secret key for JWT signing (min 32 chars) |
| `ADMIN_EMAIL` | `admin@localhost` | Initial admin account email |
| `ADMIN_PASSWORD` | `changeme` | Initial admin account password |
| `DB_PASSWORD` | `zenith` | PostgreSQL password |
| `ZENITH_MODE` | `standalone` | `standalone` or `saas` |
| `CORS_ORIGINS` | `http://localhost:3000` | Allowed CORS origins |
| `BASE_DOMAIN` | `freezenith.com` | Base domain for app routing |
| `SECRETS_ENCRYPTION_KEY` | *(optional)* | 64-char hex key for secrets encryption |
| `ENVIRONMENT` | `development` | `development` or `production` |

## Development

```bash
# API
cd services/api
go run ./cmd/server

# Dashboard
cd apps/web
pnpm install && pnpm dev

# Tests
cd services/api && go test ./...
cd apps/web && pnpm build
```

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for the full development guide.

## License

Zenith is source-available under the [Business Source License 1.1](LICENSE)
(it converts to Apache-2.0 in 2030). You are free to self-host and run it; you
may not offer it as a competing managed service. Plain-English summary:
[LICENSE-SUMMARY.md](LICENSE-SUMMARY.md).
