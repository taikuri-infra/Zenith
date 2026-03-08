# Zenith — Getting Started Guide

Welcome to Zenith! This guide walks you through deploying your first application.

## 1. Create Your Account

1. Go to [app.freezenith.com](https://app.freezenith.com)
2. Sign up with email or Google
3. Verify your email address
4. You start on the **Free plan** (1 app, 1 database)

## 2. Install the CLI (Optional)

```bash
curl -fsSL https://get.freezenith.com | sh
zenith login
```

## 3. Create Your First App

### Via Web Dashboard

1. Click **"New App"** on the dashboard
2. Choose deploy source:
   - **Git**: Connect your GitHub repo
   - **Image**: Use a Docker image
3. Configure settings (port, environment variables)
4. Click **Deploy**

### Via CLI

```bash
# From a Git repo
zenith apps create my-api --source git --repo https://github.com/you/my-api

# From a Docker image
zenith apps create my-api --source image --image myregistry/my-api:latest --port 3000
```

### Via Terraform

```hcl
resource "zenith_app" "my_api" {
  name          = "my-api"
  deploy_source = "git"
  repo_url      = "https://github.com/you/my-api"
  port          = 3000
}
```

## 4. Add a Database

Zenith supports PostgreSQL, MySQL, Redis, MongoDB, RabbitMQ, and Kafka.

### Via Dashboard

Go to **Databases** → **Create Database** → Select engine → Done.

### Via CLI

```bash
zenith db create my-db --engine postgresql
```

Connection details are shown after creation and available in the dashboard.

## 5. Set Environment Variables

```bash
zenith apps env set my-api DATABASE_URL="postgresql://..."
zenith apps env set my-api API_KEY="sk-..."
```

Or set them in the dashboard under **App → Environment**.

## 6. Custom Domain

```bash
zenith domains add my-api api.mycompany.com
```

Then create a CNAME DNS record pointing `api.mycompany.com` to your app's Zenith subdomain. TLS certificates are provisioned automatically.

## 7. API Gateway (Pro+)

Create an API gateway to aggregate multiple services under one endpoint:

```bash
# Create gateway
zenith gateways create main-gw

# Add routes
zenith gateways route add main-gw \
  --name api \
  --path "/api/*" \
  --methods GET,POST,PUT,DELETE \
  --app my-api \
  --auth jwt
```

## 8. Monitoring

View logs, metrics, and pod health in the **Monitoring** section:

- **Metrics**: CPU, memory, request rate, error rate, P95 latency
- **Logs**: Real-time log streaming with level and search filters
- **Pods**: Pod status, restarts, resource usage

## 9. Scaling

```bash
# Scale manually
zenith apps scale my-api --replicas 3

# Free tier apps scale to zero after 15 minutes of inactivity
# Pro+ apps are always-on
```

## Supported Frameworks

Zenith auto-detects your framework and configures the build:

| Framework | Detection | Build |
|-----------|-----------|-------|
| Next.js | `next.config.*` | `npm run build` → standalone |
| Express | `express` in package.json | `npm start` |
| Go | `go.mod` | `go build` |
| Python/Django | `manage.py` | `gunicorn` |
| Flask | `flask` in requirements.txt | `gunicorn` |
| Rails | `Gemfile` with rails | `rails server` |
| Static | `index.html` only | nginx |
| Dockerfile | `Dockerfile` present | Custom build |

## Plan Comparison

| Feature | Free | Pro (€29/mo) | Team (€199/mo) |
|---------|------|------|------|
| Apps | 1 | 5 | 20 |
| Databases | 1 (500MB) | 3 (5GB each) | 10 (20GB each) |
| Custom Domain | - | Yes | Yes |
| Always-On | - | Yes | Yes |
| API Gateway | - | Yes | Yes |
| RBAC / Team Members | - | - | Yes |
| SSO | - | - | Yes |
| Storage Buckets | - | 3 | 10 |

## Getting Help

- **Dashboard**: Click the help icon in the bottom-right
- **Support Tickets**: Go to **Settings → Support** (Pro+)
- **Email**: support@freezenith.com
- **Status**: status.freezenith.com
