## 1. Stripe Setup

- [ ] 1.1 Create Stripe products and prices for Free/Pro/Team plans
- [ ] 1.2 Add Stripe Go SDK to API dependencies
- [ ] 1.3 Add `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET` env vars to config
- [ ] 1.4 Create `internal/billing/stripe_client.go` wrapper

## 2. Subscription Management

- [ ] 2.1 Create Stripe customer on user registration
- [ ] 2.2 `POST /api/v1/billing/checkout` — create Stripe Checkout session for plan upgrade
- [ ] 2.3 `POST /api/v1/billing/portal` — create Stripe Customer Portal session
- [ ] 2.4 `GET /api/v1/billing/subscription` — get current subscription status
- [ ] 2.5 Plan upgrade/downgrade with proration handling
- [ ] 2.6 `stripe_customer_id` field on user entity

## 3. Webhook Handler

- [ ] 3.1 `POST /api/v1/webhooks/stripe` — receive Stripe events
- [ ] 3.2 Handle `checkout.session.completed` — activate plan
- [ ] 3.3 Handle `invoice.payment_succeeded` — record payment
- [ ] 3.4 Handle `invoice.payment_failed` — notify user, grace period
- [ ] 3.5 Handle `customer.subscription.updated` — sync plan changes
- [ ] 3.6 Handle `customer.subscription.deleted` — downgrade to free

## 4. Invoices and Usage

- [ ] 4.1 `GET /api/v1/billing/invoices` — list invoices from Stripe
- [ ] 4.2 Usage-based metering: report CPU/storage overages to Stripe
- [ ] 4.3 Invoice PDF download

## 5. Frontend

- [ ] 5.1 Billing page: current plan, usage, upgrade/downgrade buttons
- [ ] 5.2 Stripe Checkout redirect flow
- [ ] 5.3 Payment method display and management
- [ ] 5.4 Invoice history table
- [ ] 5.5 Plan comparison modal on upgrade
- [ ] 5.6 Admin dashboard: MRR, revenue, churn metrics
- [ ] 5.7 Demo mode mocks for billing

## 6. Testing

- [ ] 6.1 Unit tests for billing handlers
- [ ] 6.2 Stripe webhook signature verification tests
- [ ] 6.3 Plan transition tests (free→pro, pro→team, downgrade)
