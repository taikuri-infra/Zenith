# Galaxy - Microservice Architecture Support

> A customer can deploy 100 microservices + 1 frontend and Galaxy connects them all.

---

## The Problem

A startup has:
```
frontend (Next.js)
├── calls: api-gateway
│   ├── routes to: user-service
│   ├── routes to: order-service
│   ├── routes to: payment-service
│   ├── routes to: notification-service
│   ├── routes to: inventory-service
│   └── ... (100 services)
│
Each service needs:
├── Its own database (or shared)
├── Service-to-service communication
├── Its own environment variables
├── Its own scaling
├── Centralized logging
├── Distributed tracing
└── Health monitoring
```

Today they need: Kubernetes expertise, Helm charts, Istio/Linkerd, hours of YAML.

With Galaxy: Click, click, done.

---

## How It Works in Galaxy

### Project = Namespace = All Services Live Together

```yaml
apiVersion: galaxy.dev/v1alpha1
kind: Project
metadata:
  name: my-startup
spec:
  owner: team@startup.com
  plan: pro
  # All Applications in this project share:
  # - Same namespace
  # - Same private network (can talk via service name)
  # - Same databases
  # - Same secrets
  # - Same monitoring dashboard
```

### Service Discovery = Kubernetes Native

When you deploy 100 microservices in the same Project, they automatically find each other by name:

```
user-service    → http://user-service:8080
order-service   → http://order-service:8080
payment-service → http://payment-service:8080
```

No configuration needed. This is built into Kubernetes. Galaxy just exposes it cleanly.

### In the UI

```
Project: my-startup
│
├── Apps (100 microservices)
│   │
│   ├── frontend           ← exposed to internet (has Domain)
│   │   ├── Source: github.com/startup/frontend
│   │   ├── Domain: app.startup.com
│   │   ├── Replicas: 2
│   │   └── Env: API_URL=http://api-gateway:8080
│   │
│   ├── api-gateway         ← internal + exposed
│   │   ├── Source: github.com/startup/gateway
│   │   ├── Domain: api.startup.com
│   │   ├── Replicas: 2
│   │   └── Env: USER_SVC=http://user-service:8080
│   │         ORDER_SVC=http://order-service:8080
│   │
│   ├── user-service        ← internal only
│   │   ├── Source: github.com/startup/user-svc
│   │   ├── Replicas: 3
│   │   ├── Env: DB_URL=<from database: users-db>
│   │   └── Internal URL: http://user-service:8080
│   │
│   ├── order-service       ← internal only
│   │   ├── Source: github.com/startup/order-svc
│   │   ├── Replicas: 2
│   │   ├── Env: DB_URL=<from database: orders-db>
│   │   │       REDIS_URL=<from database: cache>
│   │   └── Internal URL: http://order-service:8080
│   │
│   ├── payment-service     ← internal only
│   │   └── ...
│   │
│   └── ... (96 more services)
│
├── Databases
│   ├── users-db        (PostgreSQL, 20GB)
│   ├── orders-db       (PostgreSQL, 50GB)
│   ├── products-db     (MongoDB, 30GB)
│   ├── cache           (Redis, 5GB)
│   └── sessions        (Redis, 2GB)
│
├── Storage
│   ├── uploads         (S3, 100GB)
│   └── backups         (S3, 50GB)
│
├── Networking
│   ├── api.startup.com    → api-gateway
│   ├── app.startup.com    → frontend
│   └── Firewall: allow 80,443 from 0.0.0.0/0
│
└── Monitoring
    ├── All 100 services in one Grafana dashboard
    ├── Request flow between services (distributed tracing)
    └── Alerts: service down, high latency, error rate
```

---

## Additional CRDs for Microservice Support

### ServiceMesh (optional, Phase 2+)

```yaml
apiVersion: galaxy.dev/v1alpha1
kind: ServiceMesh
metadata:
  name: mesh
  namespace: galaxy-my-startup
spec:
  enabled: true
  mtls: true                    # encrypt service-to-service traffic
  tracing:
    enabled: true
    samplingRate: 10            # 10% of requests traced
  retries:
    enabled: true
    maxRetries: 3
  circuitBreaker:
    enabled: true
    threshold: 50               # open circuit at 50% error rate
```

Behind the scenes: Installs Linkerd (lightweight, CNCF graduated) in the project namespace.

### MessageQueue (for async communication)

```yaml
apiVersion: galaxy.dev/v1alpha1
kind: MessageQueue
metadata:
  name: events
  namespace: galaxy-my-startup
spec:
  engine: nats                  # or rabbitmq, kafka
  storage: 10Gi
  streams:
    - name: orders
      retention: 7d
    - name: notifications
      retention: 1d
```

Behind the scenes: NATS Operator creates NATS cluster with JetStream.

### Application CRD - Enhanced for Microservices

```yaml
apiVersion: galaxy.dev/v1alpha1
kind: Application
metadata:
  name: user-service
  namespace: galaxy-my-startup
spec:
  source:
    github:
      repo: startup/user-svc
      branch: main
      path: ./                  # monorepo support

  build:
    dockerfile: ./Dockerfile
    context: ./services/user    # build context for monorepo

  replicas: 3

  resources:
    cpu: "500m"
    memory: "512Mi"

  # Port the service listens on
  port: 8080

  # Health checks
  health:
    path: /health
    port: 8080

  # Expose to internet? (default: internal only)
  expose: false                 # internal microservice

  # Connect to databases (auto-injects env vars)
  databases:
    - name: users-db
      envPrefix: DB             # creates DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_URL

  # Connect to message queues
  queues:
    - name: events
      envPrefix: NATS           # creates NATS_URL

  # Connect to object storage
  storage:
    - name: uploads
      envPrefix: S3             # creates S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY, S3_BUCKET

  # Connect to other services (for documentation/dependency tracking)
  dependencies:
    - order-service
    - notification-service

  # Environment variables
  env:
    - name: LOG_LEVEL
      value: "info"
    - name: STRIPE_KEY
      valueFrom:
        secretRef:
          name: stripe-credentials
          key: api-key

  # Auto-scaling (Phase 2)
  autoscale:
    minReplicas: 2
    maxReplicas: 10
    targetCPU: 70
```

### What `databases:` does automatically

When you set:
```yaml
databases:
  - name: users-db
    envPrefix: DB
```

Galaxy Operator automatically injects these env vars into the pod:
```
DB_HOST=users-db-rw.galaxy-my-startup.svc.cluster.local
DB_PORT=5432
DB_USER=app
DB_PASSWORD=<from secret>
DB_NAME=users
DB_URL=postgres://app:<pass>@users-db-rw:5432/users?sslmode=require
```

No manual configuration. Create a Database, reference it in Application, done.

---

## The Frontend Connection Pattern

### Single Frontend → API Gateway → Microservices

This is the most common pattern. Galaxy supports it natively:

```
Internet                         Galaxy Project (K8s namespace)
   │                            ┌───────────────────────────────────────┐
   │   app.startup.com          │                                       │
   ├──────────────────────────▶ │  frontend (Next.js, replicas: 2)     │
   │                            │    └── NEXT_PUBLIC_API=               │
   │                            │        https://api.startup.com       │
   │                            │                                       │
   │   api.startup.com          │                                       │
   ├──────────────────────────▶ │  api-gateway (replicas: 2)           │
   │                            │    ├── /users/*  → user-service       │
   │                            │    ├── /orders/* → order-service      │
   │                            │    ├── /pay/*    → payment-service    │
   │                            │    └── /notify/* → notification-svc   │
   │                            │                                       │
   │                            │  ┌─────────────┐ ┌─────────────┐     │
   │                            │  │user-service  │ │order-service│     │
   │                            │  │  :8080       │ │  :8080      │     │
   │                            │  │  DB: users-db│ │  DB: orders │     │
   │                            │  └─────────────┘ └─────────────┘     │
   │                            │  ┌─────────────┐ ┌─────────────┐     │
   │                            │  │payment-svc   │ │notif-svc    │     │
   │                            │  │  :8080       │ │  :8080      │     │
   │                            │  │  DB: payments│ │  Queue: nats│     │
   │                            │  └─────────────┘ └─────────────┘     │
   │                            │  ┌─────────────┐ ┌─────────────┐     │
   │                            │  │ users-db     │ │ orders-db   │     │
   │                            │  │ (PostgreSQL) │ │ (PostgreSQL)│     │
   │                            │  └─────────────┘ └─────────────┘     │
   │                            │  ┌─────────────┐ ┌─────────────┐     │
   │                            │  │ cache        │ │ events      │     │
   │                            │  │ (Redis)      │ │ (NATS)      │     │
   │                            │  └─────────────┘ └─────────────┘     │
   │                            └───────────────────────────────────────┘
```

### Galaxy API Gateway CRD (built-in routing)

Instead of requiring users to write their own API gateway, Galaxy provides one:

```yaml
apiVersion: galaxy.dev/v1alpha1
kind: Gateway
metadata:
  name: api
  namespace: galaxy-my-startup
spec:
  domain: api.startup.com
  routes:
    - path: /users
      service: user-service
      port: 8080
    - path: /orders
      service: order-service
      port: 8080
    - path: /payments
      service: payment-service
      port: 8080
    - path: /notifications
      service: notification-service
      port: 8080

  # Optional: rate limiting
  rateLimit:
    requests: 100
    per: minute

  # Optional: CORS
  cors:
    origins: ["https://app.startup.com"]
    methods: ["GET", "POST", "PUT", "DELETE"]

  # Optional: auth middleware
  auth:
    type: jwt
    jwksUrl: https://auth.startup.com/.well-known/jwks.json
```

Behind the scenes: Traefik IngressRoute + middleware configuration.

---

## Monorepo Support

Many microservice teams use monorepos:

```
startup-monorepo/
├── services/
│   ├── user-service/
│   │   └── Dockerfile
│   ├── order-service/
│   │   └── Dockerfile
│   └── payment-service/
│       └── Dockerfile
├── frontend/
│   └── Dockerfile
└── libs/
    └── shared/
```

Galaxy handles this with `path` in the Application CRD:

```yaml
apiVersion: galaxy.dev/v1alpha1
kind: Application
metadata:
  name: user-service
spec:
  source:
    github:
      repo: startup/monorepo
      branch: main
  build:
    dockerfile: ./services/user-service/Dockerfile
    context: .                   # root context (for shared libs)
    trigger:
      paths:                     # only rebuild when these change
        - services/user-service/**
        - libs/shared/**
```

Each service in the monorepo gets its own Application CRD. Galaxy builds only what changed.

---

## Scaling for 100 Services

### Resource Management

100 services need proper resource management:

```yaml
apiVersion: galaxy.dev/v1alpha1
kind: Project
metadata:
  name: my-startup
spec:
  plan: pro

  # Project-level resource limits
  resources:
    totalCPU: "32"              # 32 vCPU total across all services
    totalMemory: "64Gi"         # 64GB total
    totalStorage: "500Gi"       # 500GB total PVC
    maxApps: 200                # up to 200 services
    maxDatabases: 20            # up to 20 databases
```

### How many Planets for 100 services?

```
Typical microservice: 200m CPU, 256Mi RAM
100 services = 20 vCPU, 25GB RAM

Recommended:
  4x CX43 (8 vCPU, 16GB each) = 32 vCPU, 64GB RAM
  Cost: 4 x €9.49 = €37.96/mo

Or:
  2x CCX23 (4 dedicated vCPU, 16GB each) = 8 dedicated vCPU, 32GB RAM
  Cost: 2 x €24.49 = €48.98/mo

Compare to AWS EKS:
  4x t3.xlarge = 4 x $120 = $480/mo + $73 EKS fee = $553/mo

Galaxy on Hetzner: ~€38/mo vs AWS: ~$553/mo
                   14x cheaper
```

---

## Dependency Graph (built-in)

Galaxy tracks which services depend on which:

```
Frontend (Web UI shows this as an interactive graph)

  ┌──────────┐
  │ frontend │
  └────┬─────┘
       │
  ┌────▼──────┐
  │api-gateway│
  └─┬──┬──┬──┬┘
    │  │  │  │
┌───▼┐┌▼──┐┌▼────┐┌▼──────┐
│user││ordr││pay  ││notify │
│svc ││svc ││svc  ││svc    │
└─┬──┘└─┬──┘└──┬──┘└───┬───┘
  │     │      │       │
┌─▼──┐┌─▼──┐  │    ┌──▼──┐
│user││ordr │  │    │event│
│ db ││ db  │  │    │queue│
└────┘└─────┘  │    └─────┘
            ┌──▼──┐
            │cache│
            └─────┘
```

This graph is auto-generated from `dependencies` in Application CRDs and `databases`/`queues` references. The frontend shows it as an interactive, clickable diagram.

---

## Summary

Galaxy supports microservices natively because:

1. **Service discovery is free** - K8s DNS handles `http://service-name:port`
2. **Database binding is automatic** - reference a Database, env vars injected
3. **API Gateway is built-in** - Gateway CRD routes traffic to services
4. **Monorepo support** - path-based builds with change detection
5. **Dependency tracking** - visual graph of service dependencies
6. **Shared project resources** - all services in one namespace, one billing unit
7. **Per-service scaling** - each service scales independently
8. **Centralized observability** - one dashboard for all 100 services
9. **Message queues** - NATS/RabbitMQ/Kafka for async communication
10. **Service mesh (Phase 2)** - mTLS, retries, circuit breakers via Linkerd
