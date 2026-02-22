## ADDED Requirements

### Requirement: Self-Service Plan Changes
The system SHALL allow users to upgrade or downgrade their project plan via Stripe Checkout. Plan changes SHALL be prorated and take effect immediately.

#### Scenario: Upgrade from Free to Pro
- **WHEN** a user initiates upgrade to Pro plan
- **THEN** a Stripe Checkout session is created and the user is redirected to complete payment

#### Scenario: Downgrade plan
- **WHEN** a user downgrades from Pro to Free
- **THEN** the plan changes at the end of the current billing period and resources exceeding free limits are flagged

### Requirement: Stripe Customer Mapping
The system SHALL create a Stripe customer record on user registration and store the `stripe_customer_id` on the user entity.

#### Scenario: New user gets Stripe customer
- **WHEN** a user registers
- **THEN** a Stripe customer is created and linked to the user record

### Requirement: Stripe Webhook Processing
The system SHALL process Stripe webhook events for payment lifecycle: checkout completion, payment success/failure, subscription updates, and cancellations.

#### Scenario: Payment failed
- **WHEN** a `invoice.payment_failed` event is received
- **THEN** the user is notified and a grace period begins before plan downgrade
