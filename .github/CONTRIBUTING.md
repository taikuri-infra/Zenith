# Contributing to Zenith

Thanks for your interest in contributing to Zenith! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.25+
- Node.js 20+
- pnpm 10+
- Docker and Docker Compose

### Getting Started

```bash
# Clone the repo
git clone https://github.com/dotechhq/zenith.git
cd zenith

# Start dependencies
docker compose up postgres -d

# Run the API
cd services/api
export DATABASE_URL="postgres://zenith:zenith@localhost:5432/zenith?sslmode=disable"
export JWT_SECRET="dev-secret-change-me-in-production-32ch"
export ADMIN_EMAIL="admin@localhost"
export ADMIN_PASSWORD="changeme"
go run ./cmd/server

# In another terminal, run the web dashboard
cd apps/web
pnpm install
NEXT_PUBLIC_API_URL=http://localhost:8080 pnpm dev
```

## Making Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests:
   ```bash
   cd services/api && go test ./... && go vet ./...
   cd apps/web && pnpm build
   ```
5. Commit your changes (`git commit -m 'Add my feature'`)
6. Push to your fork (`git push origin feature/my-feature`)
7. Open a Pull Request

## Code Style

- **Go**: Standard `gofmt` formatting. Run `go vet ./...` before committing.
- **TypeScript/React**: Follow existing patterns. Run `pnpm build` to check for errors.

## Project Structure

```
zenith/
  services/api/     # Go API server
  apps/web/         # Next.js web dashboard
  packages/ui/      # Shared UI components
  infra/            # Infrastructure (Terraform, Ansible, Helm, K8s manifests, scripts)
```

## Reporting Issues

Use [GitHub Issues](https://github.com/dotechhq/zenith/issues) with the provided templates.

## License

By contributing to Zenith, you agree that your contributions will be licensed under the AGPLv3 License.
