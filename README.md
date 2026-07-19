# Zenith

**Open-source PaaS. Deploy apps with git push.**

Zenith is a self-hosted platform-as-a-service that gives you Heroku-like simplicity on your own infrastructure. Push code, get a running app — with databases, storage, auth, and domains built in.

## Try it

**Live read-only demo:** [demo.freezenith.com](https://demo.freezenith.com)

## Quick Start

**Prerequisites:** Docker and Docker Compose (the installer checks for them but does
not install them — set them up first via [get.docker.com](https://get.docker.com)).
Around 20 GB of RAM is recommended, because the container images currently build on
the server during install. Object storage runs locally via RustFS — no external S3
account needed.

```bash
curl -fsSL https://raw.githubusercontent.com/dotechhq/zenith/main/infra/scripts/install.sh | bash
```

Or manually:

```bash
git clone https://github.com/dotechhq/zenith.git
cd zenith
cp .env.example .env
# Edit .env — set JWT_SECRET (min 32 chars) and admin credentials
docker compose up -d
```

Open [http://localhost:3000](http://localhost:3000) and log in with your admin credentials.

## Features

- **App Deployment** — Git push to deploy. Supports Dockerfiles and buildpacks.
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
- **RustFS** — Self-hosted, S3-compatible object storage for app buckets and backups. Runs locally in the same Compose stack, so no external cloud storage account is required.

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
