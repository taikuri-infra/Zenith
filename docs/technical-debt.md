# Technical Debt Tracker

Items tracked here are known issues or shortcuts that need addressing.

## Active Debt

### 1. Web Demo App — Outdated Image (zenith-web-demo)
- **Current**: `zenith-web:0.2.0`
- **Latest**: `zenith-web:0.4.2`
- **Impact**: Demo site shows old UI without new pages (support, audit, SSO, WAF, monitoring, alerts, etc.)
- **Fix**: Rebuild demo image or update demo values to use latest web image
- **Priority**: Low — demo is separate from staging/prod

### 2. Stripe Billing — Not Configured
- **Status**: `stripe_enabled: false` in staging
- **Needed**: `STRIPE_BILLING_ENABLED`, `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, price IDs for Pro/Team/Business
- **Impact**: Plan upgrades are only possible via direct DB update
- **Priority**: Medium — needed before real users

### 3. Manual Docker Builds
- **Status**: Images are built manually on staging server via rsync + `docker build`
- **Impact**: Slow, error-prone, no reproducibility
- **Fix**: Complete GitHub Actions CI/CD pipeline to auto-build on push to staging
- **Priority**: High — blocking developer velocity

### 4. Real Monitoring Data
- **Status**: Monitoring and logs pages show mock/demo data
- **Needed**: Connect Prometheus + Loki APIs (plan exists in `.claude/plans/`)
- **Impact**: Users see fake data on monitoring dashboard
- **Priority**: Medium

### 5. No SSH Key on Staging Server
- **Status**: Staging server (zen-stage) has no GitHub SSH key
- **Impact**: `git pull` fails, must use rsync to copy code
- **Fix**: Add deploy key or GitHub Actions runner
- **Priority**: Low — resolved by CI/CD pipeline

## Resolved

_None yet — items move here when fixed._
