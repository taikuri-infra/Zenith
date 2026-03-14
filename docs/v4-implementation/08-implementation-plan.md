# 08 — 10/10 Implementation Plan

> **Goal:** Bring every section of Zenith from its current score to 10/10
> **Priority:** P0 (do first) → P1 (do second) → P2 (do third) → P3 (nice to have)
> **Format:** Same as V3 phases — checkboxes, validation commands, file references

---

## Score Summary

| # | Section | Current | Target | Priority | Effort |
|---|---------|---------|--------|----------|--------|
| 1 | Billing & Payments | 6/10 | 10/10 | **P0** | 2 weeks |
| 2 | Landing & Docs | 6.5/10 | 10/10 | **P0** | 1 week |
| 3 | Monitoring & Observability | 7/10 | 10/10 | **P1** | 2 weeks |
| 4 | Frontend (Web) | 7/10 | 10/10 | **P1** | 2 weeks |
| 5 | Team & RBAC | 7/10 | 10/10 | **P1** | 1 week |
| 6 | Support & Tickets | 7/10 | 10/10 | **P1** | 1 week |
| 7 | Frontend (Admin/MC) | 7.5/10 | 10/10 | **P2** | 1 week |
| 8 | Storage (S3) | 7.5/10 | 10/10 | **P2** | 1 week |
| 9 | App Deployment | 8/10 | 10/10 | **P2** | 1 week |
| 10 | CI/CD | 8/10 | 10/10 | **P2** | 1 week |
| 11 | Database & Migrations | 8.5/10 | 10/10 | **P3** | 3 days |
| 12 | API Gateway | 8.5/10 | 10/10 | **P3** | 3 days |
| 13 | Infrastructure | 8.5/10 | 10/10 | **P3** | 3 days |
| 14 | Auth & Security | 9/10 | 10/10 | **P3** | 2 days |

**Total estimated effort: ~12 weeks for 1 developer**

---

## P0-1: Stripe Billing Integration (6/10 → 10/10)

> **Why P0:** Without payment, there is no business. This is the #1 blocker for revenue.

### What Exists
- Plan entities and limits (`entities/plan.go`, `entities/billing.go`)
- Stub Stripe client (`adapters/stripeclient/`)
- Checkout/portal/cancel handlers (`handlers/billing.go`)
- Plan enforcement in services (`services/plan.go`)

### What's Missing

- [ ] **P0-1-01** Complete Stripe product/price setup in Stripe Dashboard
  - Create products: Free, Pro, Team, Business
  - Create prices: monthly + annual (20% discount)
  - Create add-on prices: Gold Support, Extra Compute, etc.
  - Store price IDs in config (`STRIPE_PRO_PRICE_ID`, etc.)

- [ ] **P0-1-02** Implement real Stripe checkout flow
  - **File:** `adapters/stripeclient/stripe.go`
  - Use `stripe.CheckoutSession.New()` with correct price IDs
  - Support monthly/annual billing toggle
  - Pass customer email + metadata (user_id, plan_tier)
  - Redirect to Stripe Hosted Checkout → success/cancel URL

- [ ] **P0-1-03** Implement Stripe webhook handler
  - **File:** `handlers/stripe_webhook.go`
  - Events to handle:
    - `checkout.session.completed` → activate subscription
    - `customer.subscription.updated` → plan change
    - `customer.subscription.deleted` → downgrade to Free
    - `invoice.payment_failed` → notify user, grace period
    - `invoice.paid` → create invoice record
  - Verify webhook signature (STRIPE_WEBHOOK_SECRET)

- [ ] **P0-1-04** Implement subscription lifecycle
  - **File:** `services/billing.go`
  - `CreateCheckoutSession(userID, tier, annual)` → Stripe URL
  - `HandleCheckoutComplete(sessionID)` → update user plan + limits
  - `CancelSubscription(userID)` → Stripe cancel + schedule downgrade
  - `GetBillingPortalURL(userID)` → Stripe customer portal

- [ ] **P0-1-05** Implement invoice generation
  - **File:** `services/billing.go`
  - Fetch invoices from Stripe API (`stripe.Invoice.List()`)
  - Store locally for fast access
  - PDF download via Stripe hosted URL

- [ ] **P0-1-06** Implement usage metering (for future pay-as-you-go)
  - **File:** `services/billing.go`
  - Track: build minutes, storage GB, bandwidth GB
  - Report to Stripe Metering API monthly
  - Show usage on billing page

- [ ] **P0-1-07** Implement trial period
  - 14-day free trial on Pro plan (no credit card required)
  - Auto-downgrade to Free after trial ends
  - Trial banner on dashboard

- [ ] **P0-1-08** Frontend: Complete billing page
  - **File:** `apps/web/src/app/billing/page.tsx`
  - Current plan + usage meters
  - Upgrade/downgrade buttons (→ Stripe Checkout)
  - Invoice history table (date, amount, PDF link)
  - Payment method management (→ Stripe Portal)
  - Trial countdown (if applicable)

- [ ] **P0-1-09** Frontend: Plan upgrade modal in sidebar
  - When user hits plan limit, show upgrade nudge
  - "You've used 5/5 apps. Upgrade to Team for 20 apps."

- [ ] **P0-1-10** Add APISIX route for Stripe webhook
  - **File:** `infra/helm/zenith-api/templates/apisix-route.yaml`
  - Add `/api/v1/billing/webhook` to public routes (no JWT needed)
  - Stripe sends POST requests directly

### Validation
```bash
# Stripe CLI for local testing
stripe listen --forward-to localhost:8080/api/v1/billing/webhook
stripe trigger checkout.session.completed
# Verify: user plan updated, invoice created
```

### Files to Modify
| File | Change |
|------|--------|
| `adapters/stripeclient/stripe.go` | Real Stripe API calls |
| `services/billing.go` | Checkout, webhook, invoice, metering |
| `handlers/billing.go` | Checkout, portal endpoints |
| `handlers/stripe_webhook.go` | Webhook event handling |
| `config/config.go` | Stripe price ID env vars |
| `cmd/server/main.go` | Wire webhook route |
| `apps/web/src/app/billing/page.tsx` | Complete billing UI |
| `apps/web/src/lib/api.ts` | Billing API methods |
| `infra/helm/zenith-api/templates/apisix-route.yaml` | Webhook route |

---

## P0-2: Landing Page & Documentation (6.5/10 → 10/10)

> **Why P0:** Without docs, nobody signs up. Without SEO, nobody finds us.

### What's Missing

- [ ] **P0-2-01** SEO optimization for landing page
  - **File:** `apps/landing/src/app/layout.tsx`
  - Add: title, description, og:image, twitter:card, structured data
  - Create `sitemap.xml` and `robots.txt`
  - Add canonical URLs

- [ ] **P0-2-02** Blog section
  - **File:** `apps/landing/src/app/blog/page.tsx` (NEW)
  - MDX-based blog posts (use `next-mdx-remote`)
  - Categories: tutorials, comparisons, changelog
  - RSS feed

- [ ] **P0-2-03** Changelog page
  - **File:** `apps/landing/src/app/changelog/page.tsx` (NEW)
  - Auto-generated from GitHub Releases (Release Please)
  - Show: version, date, changes, breaking changes

- [ ] **P0-2-04** Documentation site (API reference)
  - **File:** `apps/landing/src/app/docs/` (expand existing)
  - Getting Started guide (5 min quickstart)
  - API reference (auto-generated from Swagger)
  - SDK examples (curl, JavaScript, Python, Go)
  - Guide: "Deploy your first app"
  - Guide: "Set up a database"
  - Guide: "Configure API Gateway"
  - Guide: "Add authentication"

- [ ] **P0-2-05** Comparison pages
  - **Files:** `apps/landing/src/app/compare/[competitor]/page.tsx` (NEW)
  - Zenith vs Vercel (price, features, EU data)
  - Zenith vs Railway (price, databases, gateway)
  - Zenith vs Supabase (self-hosted, pricing)
  - SEO-optimized for "[competitor] alternative"

- [ ] **P0-2-06** Social proof
  - **File:** `apps/landing/src/components/testimonials.tsx` (NEW)
  - Customer logos (when available)
  - Testimonial quotes
  - GitHub stars badge
  - "Trusted by X developers" counter

- [ ] **P0-2-07** CLI documentation
  - **File:** `apps/landing/src/app/docs/cli/page.tsx` (NEW)
  - `zenith login`, `zenith deploy`, `zenith logs`, etc.
  - Installation instructions
  - Shell autocompletion

### Validation
```bash
# Check SEO
npx next-sitemap  # Generate sitemap
curl https://stage.freezenith.com/sitemap.xml

# Check meta tags
curl -s https://stage.freezenith.com | grep '<meta'

# Lighthouse score
npx lighthouse https://stage.freezenith.com --only-categories=seo
```

---

## P1-1: Monitoring & Observability (7/10 → 10/10)

> **Why P1:** Enterprise customers expect this. Without it, we can't sell Business tier.

### What's Missing

- [ ] **P1-1-01** Distributed tracing integration
  - **File:** `services/api/internal/telemetry/tracer.go`
  - Initialize OTel TracerProvider
  - Add trace context propagation to all HTTP handlers
  - Add spans to key service methods (deploy, database provision, gateway sync)
  - Forward traces to Tempo via OTel Collector

- [ ] **P1-1-02** Log aggregation via Loki
  - **File:** `services/monitoring.go`
  - Implement `GetLogs()` using Loki HTTP API (`/loki/api/v1/query_range`)
  - Filter by: namespace, pod, level, time range, search text
  - Stream logs via SSE (Server-Sent Events)

- [ ] **P1-1-03** Custom alert rules per customer
  - **New file:** `services/alerts.go` (expand existing)
  - CRUD for PrometheusRule CRDs (scoped to customer namespace)
  - Templates: "Alert when CPU > X%", "Alert when error rate > Y%"
  - Notification channels: email, webhook, Slack

- [ ] **P1-1-04** Uptime monitoring per app
  - **New file:** `services/uptime.go` (NEW)
  - Background goroutine: HTTP probe every 60s for each app
  - Store uptime % (24h, 7d, 30d)
  - Show on app detail page + dashboard

- [ ] **P1-1-05** Error tracking (Sentry-like)
  - Capture application errors from Loki logs
  - Group by error signature
  - Show: error count, first/last occurrence, affected apps
  - Frontend page: `/errors`

- [ ] **P1-1-06** Frontend: Enhanced monitoring pages
  - **File:** `apps/web/src/app/monitoring/page.tsx`
  - Real-time charts (CPU, memory, network, request rate)
  - Log viewer with search/filter
  - Trace viewer (service map, span timeline)
  - Uptime status page
  - Alert configuration UI

### Files to Modify
| File | Change |
|------|--------|
| `telemetry/tracer.go` | OTel trace initialization |
| `services/monitoring.go` | Loki log queries, uptime probes |
| `handlers/monitoring.go` | New endpoints (logs, traces, uptime) |
| `apps/web/src/app/monitoring/page.tsx` | Charts, log viewer, traces |
| `apps/web/src/lib/api.ts` | Monitoring API methods |

---

## P1-2: Frontend Quality (7/10 → 10/10)

> **Why P1:** Buggy frontend = no trust. Users judge quality by UI.

### What's Missing

- [ ] **P1-2-01** Test coverage (target: 80%)
  - Add tests for: useApi, useAuth, useMutation hooks
  - Add tests for: Shell, StatusBadge, Modal components
  - Add tests for: key page renders (dashboard, apps, databases)
  - Setup: Vitest + React Testing Library (already in devDeps)

- [ ] **P1-2-02** Global error boundary
  - **File:** `apps/web/src/app/error.tsx` (NEW)
  - Catch unhandled errors → show friendly error page
  - "Something went wrong" + retry button + support link
  - Log error to console (future: send to error tracking)

- [ ] **P1-2-03** Loading skeletons for ALL pages
  - **File:** `apps/web/src/app/*/loading.tsx` (NEW for each route)
  - Skeleton matching the page layout
  - Prevents layout shift on load

- [ ] **P1-2-04** Accessibility audit (a11y)
  - Run `axe-core` on all pages
  - Fix: aria labels, keyboard navigation, focus management
  - Color contrast check (especially on dark theme)
  - Screen reader testing

- [ ] **P1-2-05** "Coming Soon" pages → Real implementation
  - **Backstage** (`/backstage`) → K8s resource viewer (namespaces, pods, services)
  - **Compliance** (`/compliance`) → SOC2/GDPR checklist, DPA download
  - Any other "Coming Soon" sections

- [ ] **P1-2-06** Internationalization (i18n) — Phase 1
  - Extract all strings to locale files
  - Support: English (default), Finnish (fi), Persian (fa)
  - Use `next-intl` or `react-i18next`

- [ ] **P1-2-07** Responsive design audit
  - Test all pages on: mobile (375px), tablet (768px), desktop (1440px)
  - Fix any overflow, truncation, or layout issues
  - Collapsible sidebar on mobile

### Validation
```bash
# Run tests
cd apps/web && pnpm test

# Coverage report
cd apps/web && pnpm test -- --coverage

# Lint
cd apps/web && npx next lint --quiet

# a11y check (manual)
# Install axe browser extension → scan each page
```

---

## P1-3: Team & RBAC (7/10 → 10/10)

### What's Missing

- [ ] **P1-3-01** Granular permissions (resource-level)
  - Currently: role-based (owner/admin/dev/viewer)
  - Need: "developer X can only edit App Y" (resource-level ACL)
  - **File:** `entities/role.go` — add `ResourcePermission` struct
  - **File:** `services/team.go` — check resource permissions in service methods

- [ ] **P1-3-02** Per-seat billing enforcement
  - Track active seats in subscription
  - Prevent invite if seats full (Team: min 3, Business: min 3)
  - Show seat count on billing page

- [ ] **P1-3-03** Activity feed per team
  - **New file:** `services/activity.go` (NEW)
  - Track: who deployed what, who changed settings, who invited whom
  - Show timeline on dashboard
  - Filterable by member, action type, date range

- [ ] **P1-3-04** Frontend: Enhanced team page
  - **File:** `apps/web/src/app/iam/page.tsx`
  - Pending invitations with resend/cancel
  - Permission matrix (who can access what)
  - Activity feed tab

---

## P1-4: Support & Tickets (7/10 → 10/10)

### What's Missing

- [ ] **P1-4-01** SLA tracking
  - **File:** `services/support.go`
  - Track: first response time, resolution time
  - Gold Support: 10 min SLA — alert if approaching
  - Show SLA status on ticket detail
  - Dashboard widget for admin: "2 tickets approaching SLA breach"

- [ ] **P1-4-02** Real-time notifications for tickets
  - When admin replies → email + in-app notification to customer
  - When customer replies → email + in-app notification to admin
  - Use NATS event bus for pub/sub

- [ ] **P1-4-03** Knowledge base
  - **File:** `apps/landing/src/app/docs/kb/page.tsx` (NEW)
  - Common questions + answers
  - Link from support page: "Check our KB before creating a ticket"
  - Searchable

- [ ] **P1-4-04** Canned responses for admin
  - **File:** `apps/mission-control/src/app/support/[id]/page.tsx`
  - Pre-written reply templates
  - "Your database has been restored", "Please update your image tag", etc.

---

## P2-1: Admin/MC Quality (7.5/10 → 10/10)

### What's Missing

- [ ] **P2-1-01** Real-time dashboard updates (WebSocket)
  - Currently: data loads on page mount, manual refresh
  - Need: WebSocket push for new events (new customer, new ticket, alert firing)

- [ ] **P2-1-02** Bulk operations
  - Select multiple customers → bulk suspend/activate
  - Select multiple tickets → bulk assign/close

- [ ] **P2-1-03** Export/Import
  - Export customer list as CSV
  - Export audit logs as JSON
  - Export billing data as CSV

---

## P2-2: Storage S3 (7.5/10 → 10/10)

### What's Missing

- [ ] **P2-2-01** Real-time quota enforcement
  - Before upload: check `current_size + file_size <= max_size`
  - Reject with clear error: "Storage limit reached. Upgrade to Pro for 10GB."
  - Background job: sync actual S3 usage with database

- [ ] **P2-2-02** File preview
  - Images: inline preview in file browser
  - Text/JSON/CSV: syntax-highlighted preview
  - PDF: first-page thumbnail

- [ ] **P2-2-03** CDN integration
  - Presigned URLs with Cloudflare CDN caching
  - Cache headers on public bucket objects

- [ ] **P2-2-04** File versioning
  - S3 versioning on pro+ buckets
  - Show version history in file browser
  - Restore previous version

---

## P2-3: App Deployment (8/10 → 10/10)

### What's Missing

- [ ] **P2-3-01** Rollback UI
  - **File:** `apps/web/src/app/apps/[name]/page.tsx`
  - Deployment history list (version, timestamp, status, git SHA)
  - "Rollback to this version" button
  - **File:** `handlers/deployments.go` — `RollbackDeployment` endpoint
  - **File:** `services/deploy/deployer.go` — `Rollback()` method (update K8s Deployment image)

- [ ] **P2-3-02** Deploy history page
  - Timeline of all deployments
  - Status (success, failed, in-progress)
  - Duration, build log link, deploy log link

- [ ] **P2-3-03** Build log streaming
  - SSE endpoint for real-time Kaniko build output
  - Show in deploy wizard as build progresses
  - Save complete log after build finishes

- [ ] **P2-3-04** Custom Dockerfile path
  - Support: `Dockerfile` not in repo root
  - Input field: "Dockerfile path" (default: `./Dockerfile`)
  - Pass to Kaniko: `--dockerfile=<path>`

---

## P2-4: CI/CD Improvements (8/10 → 10/10)

### What's Missing

- [ ] **P2-4-01** E2E tests in CI
  - Run smoke tests automatically on PR (not just manual)
  - Block merge if tests fail
  - **File:** `.github/workflows/ci.yml` — add smoke test job

- [ ] **P2-4-02** Preview environments per PR
  - On PR: deploy to temporary namespace (zenith-preview-<pr-number>)
  - Comment on PR with preview URL
  - Auto-cleanup on PR close
  - **File:** `.github/workflows/preview.yml` (NEW)

- [ ] **P2-4-03** Canary deployments
  - Progressive rollout: 10% → 50% → 100%
  - Automatic rollback on error rate spike
  - Use: Argo Rollouts or Flagger

---

## P3-1: Database & Migrations (8.5/10 → 10/10)

### What's Missing

- [ ] **P3-1-01** Read replicas for analytics queries
  - CNPG supports replicas natively
  - Route analytics queries to replica (read-only connection string)

- [ ] **P3-1-02** Migration CLI tool
  - `lich migration rollback` — safely rollback last migration
  - `lich migration status` — show current version, pending migrations

- [ ] **P3-1-03** Row-level security documentation
  - Document when RLS is needed vs service-layer checks
  - Current approach: service-layer ownership checks (sufficient for now)

---

## P3-2: API Gateway (8.5/10 → 10/10)

### What's Missing

- [ ] **P3-2-01** Domain verification (DNS TXT check)
  - Before adding custom domain: verify ownership via DNS TXT record
  - Generate: `_zenith-verify.example.com TXT "v=zenith1 gw=<gateway_id>"`
  - Check before issuing certificate

- [ ] **P3-2-02** Per-gateway rate limits
  - Currently: global rate limit (500 req/60s)
  - Need: configurable per gateway (customer sets their own limit)
  - **File:** `services/gateway.go` — add rate_limit field to gateway

- [ ] **P3-2-03** Gateway access logs
  - Stream APISIX access logs per gateway
  - Show in gateway analytics section

---

## P3-3: Infrastructure (8.5/10 → 10/10)

### What's Missing

- [ ] **P3-3-01** DR automation
  - Script: automated restore drill (monthly)
  - Restore CNPG backup → verify data → destroy test cluster

- [ ] **P3-3-02** Cost monitoring
  - Fetch Hetzner billing via API
  - Show monthly cost on admin dashboard
  - Cost per customer calculation

- [ ] **P3-3-03** Multi-cluster readiness
  - Document: how to add second cluster
  - Terraform module for multi-cluster
  - DNS failover configuration

---

## P3-4: Auth & Security (9/10 → 10/10)

### What's Missing

- [ ] **P3-4-01** Account lockout after N failed attempts
  - After 5 failed logins: lock account for 15 minutes
  - **File:** `services/auth.go` — track failed attempts in Redis
  - Unlock: wait 15min or admin unlock

- [ ] **P3-4-02** Auth audit events
  - Log all auth events: login success/fail, MFA enable/disable, password change
  - Store in audit_events table
  - Show on user's security page

- [ ] **P3-4-03** Session revocation UI
  - **File:** `apps/web/src/app/settings/page.tsx`
  - List active sessions (device, IP, last active)
  - "Revoke" button per session
  - "Revoke all other sessions" button

---

## Implementation Order (Recommended)

### Month 1: Revenue (P0)
```
Week 1-2: P0-1 Stripe Billing (the only thing that makes money)
Week 3:   P0-2 Landing + Docs (so people can find us and sign up)
Week 4:   P1-2 Frontend Quality (so the product feels professional)
```

### Month 2: Enterprise Features (P1)
```
Week 5-6: P1-1 Monitoring & Observability
Week 7:   P1-3 Team & RBAC + P1-4 Support
Week 8:   P2-3 App Deployment (rollback UI, deploy history)
```

### Month 3: Polish (P2-P3)
```
Week 9:   P2-1 Admin MC + P2-2 Storage
Week 10:  P2-4 CI/CD improvements
Week 11:  P3 tasks (database, gateway, infra, auth)
Week 12:  Testing, polishing, documentation updates
```

---

## Verification Commands

```bash
# Backend: verify all code compiles
cd services/api && GO111MODULE=on go vet ./internal/...

# Frontend: lint all apps
cd apps/web && npx next lint --quiet
cd apps/mission-control && npx next lint --quiet
cd apps/landing && npx next lint --quiet

# Helm: validate all charts
helm lint infra/helm/zenith-api/
helm lint infra/helm/zenith-web/

# Smoke tests: full system verification
cd infra/scripts && ./smoke-test-customer.sh
cd infra/scripts && ./smoke-test-owner.sh

# Terraform: plan changes
cd infra/terraform/staging-k8s && terraform plan
```

---

**Next → [09 — Quick Reference](./09-quick-reference.md)**
