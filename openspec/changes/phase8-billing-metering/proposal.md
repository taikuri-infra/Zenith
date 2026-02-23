# Phase 8: Billing & Metering — Stripe + Usage Tracking

## Summary

Integrate Stripe billing, track resource usage per customer, enforce tier limits, and enable self-service plan upgrades.

## Prerequisites

- Phase 7 complete (observability — we need metrics for metering)
- Stripe account set up

## Steps

### Step 8.1: Stripe Integration

**What:** Connect Stripe for subscription management.

**Build:**
- Add Stripe Go SDK to API
- Products/Prices: Free (€0), Pro (€29/mo)
- Checkout session for new subscriptions
- Customer portal for managing subscriptions
- Webhook handler for: `invoice.paid`, `invoice.payment_failed`, `customer.subscription.updated/deleted`

**Your manual work:**
1. Create Stripe account (if not exists)
2. Create Products + Prices in Stripe dashboard
3. Add secrets: `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_PRICE_PRO`
4. Configure webhook endpoint in Stripe: `https://api.stage.freezenith.com/api/v1/webhooks/stripe`

**Verify:**
```bash
# Checkout session
curl -X POST https://api.stage.freezenith.com/api/v1/billing/checkout \
  -H 'Authorization: Bearer TOKEN' \
  -d '{"plan":"pro"}'
# Returns: Stripe checkout URL
```

### Step 8.2: Resource Metering

**What:** Track CPU, RAM, storage, pod count per project.

**Build:**
- Periodic job (every 5 min) queries Prometheus for per-namespace metrics
- Stores in `resource_usage` table: project_id, timestamp, cpu_used, ram_used, storage_used, pod_count
- API endpoint: `GET /api/v1/projects/:id/usage`

**Your manual work:** None

**Verify:**
```bash
curl https://api.stage.freezenith.com/api/v1/projects/PROJECT_ID/usage \
  -H 'Authorization: Bearer TOKEN'
# Returns: current usage vs limits
```

### Step 8.3: Tier Limit Enforcement

**What:** Enforce resource limits — block actions that exceed tier.

**Build:**
- Before creating app/database/storage, check current usage vs tier limits
- Return 402/403 with "Upgrade to Pro" message when limit hit
- ResourceQuota in k8s as hard enforcement (backup)

**Your manual work:** None

**Verify:**
- Free user tries to create 2nd pod → blocked with upgrade prompt

### Step 8.4: Web Platform Billing Page

**What:** Billing page shows real plan, usage, payment method.

**Build:**
- `/billing` page shows: current plan, resource usage bars, payment method
- "Upgrade" button → Stripe checkout
- "Manage subscription" → Stripe customer portal
- Invoice history

**Your manual work:** None

**Verify:**
1. Login to Web Platform → Billing
2. See current plan and usage
3. Click Upgrade → Stripe checkout page
4. Complete payment → plan upgraded → limits increased

## Acceptance Criteria

- [ ] Stripe subscriptions work (checkout, portal, webhooks)
- [ ] Resource usage tracked per project (CPU, RAM, storage, pods)
- [ ] Tier limits enforced (Free users blocked at limits)
- [ ] Billing page shows real data
- [ ] Plan upgrade increases resource limits automatically
- [ ] Invoice history accessible
- [ ] Failed payments trigger grace period / suspension
