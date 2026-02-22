# Change: Add Stripe Billing (Phase 6)

## Why
Revenue requires self-service billing. Users need to upgrade/downgrade plans, enter payment info, and see invoices — all without contacting sales. Stripe handles international payments; Fairbroker handles IRR/Toman.

## What Changes
- Stripe integration: products, prices, subscriptions, customer portal
- Plan upgrade/downgrade flow with proration
- Usage-based metering for overages (CPU, storage beyond plan ceiling)
- Invoice generation and history
- Payment method management (card, SEPA)
- Billing page in Web Platform
- Webhook handler for Stripe events (payment succeeded/failed, subscription updated/cancelled)
- Admin dashboard: revenue metrics, MRR, churn

## Impact
- Affected specs: project-management (plan changes), web-platform (billing page), admin-dashboard (revenue stats)
- Affected code: `services/api/` (new billing handlers, Stripe client), `apps/web/` (billing page), `apps/mission-control/` (revenue dashboard)
- New dependencies: Stripe Go SDK (`github.com/stripe/stripe-go/v78`)
- Sensitive: payment data, PCI compliance considerations
