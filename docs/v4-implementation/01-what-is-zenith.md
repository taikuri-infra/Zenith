# 01 — What is Zenith?

> **Read time:** 30 minutes
> **Prerequisite:** None — start here
> **Next:** [02 — Architecture Deep Dive](./02-architecture.md)

---

## The One-Sentence Pitch

**Zenith is a Kubernetes-native PaaS (Platform-as-a-Service) built on Hetzner Cloud that gives developers everything they need to deploy, manage, and scale applications — 3-5x cheaper than AWS/Vercel/Railway.**

Think of it as: **Supabase + Railway + Vercel**, but European, affordable, and self-hostable.

---

## What We Provide

| Feature | Customer Sees | What's Under the Hood |
|---------|--------------|----------------------|
| Deploy Apps | "Push code, get URL" | Kaniko build → Harbor → K8s Deployment → IngressRoute → TLS |
| Managed Database | "One-click PostgreSQL" | CNPG Operator → WAL archiving → S3 backup → auto-failover |
| Object Storage | "S3 buckets" | Hetzner Object Storage → presigned URLs → quota enforcement |
| API Gateway | "Route & protect APIs" | APISIX → JWT verify → rate-limit → CORS → custom domains |
| Authentication | "Add login to your app" | Keycloak realms → OAuth/OIDC/SAML → user pools → MFA |
| Monitoring | "See metrics & logs" | Prometheus + Loki + Tempo → Grafana dashboards → alerts |
| Container Registry | "Push Docker images" | Harbor → Trivy scan → Kyverno admission → deploy |
| Team Management | "Invite teammates" | RBAC (owner/admin/developer/viewer) → project isolation |
| Custom Domains | "Use your own domain" | cert-manager → Let's Encrypt → Certificate CRD → IngressRoute |
| Message Queues | "Async messaging" | RabbitMQ Operator / Strimzi Kafka (customer CRDs) |

**Rule:** Customers NEVER see infrastructure names. They see "API Gateway", not "APISIX". They see "Managed Database", not "CNPG". This is intentional — we abstract complexity.

---

## Who Uses Zenith?

### Customer Types

```
┌──────────────────────────────────────────────────────────────┐
│                                                               │
│  SOLO DEVELOPER (Free/Pro)                                    │
│  "I want to deploy my side project with zero DevOps"          │
│  Uses: Apps, Database, Storage, maybe Gateway                 │
│  Pays: €0-29/month                                            │
│                                                               │
│  STARTUP TEAM (Team/Business)                                 │
│  "We need infrastructure but can't afford a DevOps engineer"  │
│  Uses: Everything — apps, DBs, auth, monitoring, RBAC         │
│  Pays: €149/seat × 3-10 seats = €447-1,490/month             │
│                                                               │
│  GROWING COMPANY (Enterprise)                                 │
│  "We need dedicated infrastructure with SLA guarantees"       │
│  Uses: Dedicated cluster, SSO, compliance, audit              │
│  Pays: €2,000+/month (custom)                                 │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### Our Operators (Zenith Team)

We use **Mission Control** to manage the platform:
- Monitor customer health, cluster status, billing
- Handle support tickets
- Manage plans, modules, infrastructure
- View audit logs, security posture

---

## Pricing Strategy — The McDonald's Decoy Effect

> **Philosophy:** We are Audi. Premium quality, accessible pricing.

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│   FREE   │    │   PRO    │    │   TEAM   │    │ BUSINESS │    │ENTERPRISE│
│   €0/mo  │    │  €29/mo  │    │ €99/seat │    │€149/seat │    │ Custom   │
│          │    │          │    │  min 3   │    │  min 3   │    │ €2000+   │
│          │    │          │    │          │    │          │    │          │
│  (test)  │    │ (worth   │    │ (pricey) │    │ (BEST    │    │  (sales  │
│          │    │  it)     │    │ DECOY ←  │    │  DEAL) ← │    │  call)   │
└──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘
```

**Why this works:**
- **Team** (€99/seat) seems expensive for shared infra + no SSO
- **Business** (€149/seat) is only €50 more but gets **dedicated** infra + SSO + audit + compliance
- 90% choose Business because it's obviously the best value

### Tier Comparison

| Feature | Free | Pro | Team | Business | Enterprise |
|---------|------|-----|------|----------|------------|
| Price | €0 | €29/mo | €99/seat (min 3) | €149/seat (min 3) | Custom |
| Infrastructure | Shared | Shared | Shared | **Dedicated** | Dedicated |
| Apps | 1 | 5 | 20 | Unlimited | Unlimited |
| Databases | 1 | 3 | 10 | Unlimited | Unlimited |
| Storage | 1GB | 10GB | 100GB | Unlimited | Unlimited |
| Always-On | No (15min sleep) | Yes | Yes | Yes | Yes |
| Custom Domain | No | Yes | Yes | Yes | Yes |
| Registry | No | Yes | Yes | Yes | Yes |
| RBAC | No | No | Yes | Yes | Yes |
| SSO (SAML/OIDC) | No | No | No | **Yes** | Yes |
| Audit Logs | No | No | No | **Yes** | Yes |
| Compliance | No | No | No | **Yes** | Yes |
| SLA | — | — | — | 99.5% | 99.9% |

### Support Tiers

| Level | Response | Cost | Who |
|-------|----------|------|-----|
| Community | Best-effort | €0 | Free |
| Standard | 48h email | Included | Pro+ |
| Priority | 12h ticket | Included | Business |
| Gold | 10 min | €699/mo | Pro+ add-on |
| Platinum | 5 min + dedicated engineer | €1,499/mo | Business+ |

### Cost Advantage (Why We Can Be Cheap)

```
Hetzner CX41 (4 vCPU, 16GB RAM): €15.90/month
AWS t3.xlarge (4 vCPU, 16GB RAM): €140/month  ← 9x more expensive
```

With a single CX41 running 20 free-tier customers:
- Our cost: €15.90/month / 20 = **€0.80 per customer**
- Even at €29/month Pro pricing, our margin is **97%**

Team of 3 engineers at €500-1,000/person (Iran-based): **€1,500-3,000/month**

Break-even: ~10 Pro customers or ~3 Business seats.

---

## Business Context (Important for Decision-Making)

| Fact | Implication |
|------|-------------|
| Company is a Finnish Oy (EU entity) | GDPR compliant, Stripe-ready, EU-based billing |
| Infrastructure is 100% Hetzner (Germany) | 3-5x cheaper than AWS, European data residency |
| Team cost is €500-1,000/person | Can operate at very low revenue levels |
| Iran conflict blocks public launches | Manual outreach only (LinkedIn DMs, communities) |
| 32K LinkedIn followers | Primary marketing channel |
| ProductHunt launch postponed | Waiting for geopolitical situation |
| Startup-first strategy | Heavy discounts, brand > profit |

---

## The Three Frontend Apps

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                  │
│  LANDING PAGE          WEB DASHBOARD         MISSION CONTROL    │
│  apps/landing/         apps/web/              apps/mission-ctrl/ │
│  Port 3200             Port 3000              Port 3100          │
│                                                                  │
│  WHO: Public           WHO: Customers         WHO: Operators     │
│  AUTH: None            AUTH: JWT (user)        AUTH: JWT (admin)  │
│                                                                  │
│  Pages:                Pages:                 Pages:             │
│  • Hero + Features     • Dashboard            • Command Center   │
│  • Pricing             • Apps (deploy)         • Customers        │
│  • Docs                • Databases             • Clusters         │
│                        • Storage               • Services         │
│                        • Gateway               • Modules          │
│                        • Auth Pools            • Security         │
│                        • Monitoring            • Logs/Traces      │
│                        • Registry              • Backups          │
│                        • Team/RBAC             • GitOps           │
│                        • Billing               • CRM/Analytics    │
│                        • Support               • Support          │
│                        • Settings              • Settings         │
│                                                                  │
│  ALL → zenith-api (Go/Fiber, port 8080)                          │
│  Via: APISIX gateway (JWT, rate-limit, CORS)                     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Customer Journey

```
Step 1: DISCOVER
  └── Visit freezenith.com → See pricing → Compare with Vercel/Railway

Step 2: SIGN UP
  └── Click "Start Free" → Register → Verify email → Choose plan

Step 3: FIRST DEPLOY
  └── Dashboard → Create App → Connect GitHub or push image
  └── Kaniko builds → Harbor stores → K8s deploys → DNS + TLS auto
  └── App live at: my-app.apps.freezenith.com (under 5 minutes)

Step 4: ADD SERVICES
  └── Create PostgreSQL database (one click)
  └── Create S3 bucket (one click)
  └── Set up API Gateway routes
  └── Configure auth pools (SSO/OAuth)

Step 5: GROW
  └── Add team members
  └── Set up monitoring & alerts
  └── Add custom domain
  └── Upgrade plan for more resources

Step 6: SCALE (Business+)
  └── Dedicated infrastructure
  └── SSO/SAML for team
  └── Compliance dashboard
  └── SLA guarantees
```

---

**Next → [02 — Architecture Deep Dive](./02-architecture.md)**
