# Zenith Deployment Plan - freezenith.com + Mission Control + Customer Test

## Server Info
- **Host**: ghasi (SSH config)
- **IP**: 161.35.82.211
- **OS**: Ubuntu 25.04, 4 vCPU, 8GB RAM, 155GB disk
- **K8s**: k3s v1.34.3 with Traefik 3.5.1 ingress
- **cert-manager**: installed, `letsencrypt-prod` ClusterIssuer ready
- **Repo on server**: `/opt/zenith` (needs `git pull` to get latest)
- **Docker images exist**: `zenith-web`, `zenith-mc`

## Prerequisites - What I Need From You

### 1. Cloudflare API Token
Create a **Custom API Token** at https://dash.cloudflare.com/profile/api-tokens with:

| Permission | Access |
|-----------|--------|
| Zone - DNS - Edit | For freezenith.com AND embermind.com |
| Zone - Zone - Read | All zones (to list/find zone IDs) |

**Zone Resources**: Include > Specific zone > `freezenith.com` AND `embermind.com`

This token lets me:
- Create DNS A records pointing to 161.35.82.211
- Create CNAME records for subdomains
- Verify domain ownership for SSL certificates

### 2. Confirm domains are on Cloudflare
- [ ] freezenith.com nameservers point to Cloudflare
- [ ] embermind.com nameservers point to Cloudflare

### 3. Cloudflare SSL mode
Set both domains to **Full (strict)** SSL mode in Cloudflare dashboard, OR **disable Cloudflare proxy** (grey cloud) so cert-manager can issue Let's Encrypt certs directly.

---

## Phase 1: Server Cleanup & Preparation

### Task 1.1: Clean Docker images
```
docker image prune -a --filter "until=48h"  # Remove dangling images
docker system prune -f                       # Clean build cache
```
**Expected**: Free ~30-40GB of disk space

### Task 1.2: Update Zenith repo on server
```
cd /opt/zenith && git pull origin main
```

### Task 1.3: Install build dependencies
```
cd /opt/zenith && pnpm install
```

### Test Case 1:
- [ ] `df -h /` shows <70% disk usage
- [ ] `/opt/zenith` has latest code (commit 8d1584b or newer)
- [ ] `pnpm install` completes without errors

---

## Phase 2: Design & Build freezenith.com Landing Page

### Task 2.1: Redesign landing page
Complete redesign of `apps/landing/` - make it world-class:

**Pages:**
- `/` - Hero + Features + How it Works + Pricing + CTA
- `/docs` - Documentation hub
- `/pricing` - Detailed pricing comparison

**Design requirements:**
- Dark theme, emerald accent (#10B981)
- Animated terminal showing `zen install` flow
- Interactive pricing calculator
- Architecture diagram (visual)
- Comparison table vs AWS/GCP/Heroku
- Testimonials section (placeholder)
- Mobile responsive
- Performance: <2s load, 90+ Lighthouse score

### Task 2.2: Build Docker image for landing page
```dockerfile
# apps/landing/Dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY . .
RUN pnpm install && pnpm build
FROM node:20-alpine
COPY --from=builder /app/.next .next
COPY --from=builder /app/public public
COPY --from=builder /app/package.json .
CMD ["pnpm", "start"]
```

### Task 2.3: Build & push image on server
```
cd /opt/zenith
docker build -t zenith-landing:latest -f apps/landing/Dockerfile .
```

### Test Case 2:
- [ ] `docker build` succeeds
- [ ] `docker run -p 3200:3000 zenith-landing` shows landing page
- [ ] All 3 pages load correctly (/, /docs, /pricing)
- [ ] Mobile responsive (test at 375px width)
- [ ] No console errors

---

## Phase 3: Deploy freezenith.com to k3s

### Task 3.1: Create Cloudflare DNS records (automated)
```
freezenith.com      A    161.35.82.211  (proxied: OFF for cert-manager)
www.freezenith.com  A    161.35.82.211  (proxied: OFF)
```

### Task 3.2: Create K8s namespace and resources
```yaml
# k8s/zenith-landing.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: zenith-platform
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: landing
  namespace: zenith-platform
spec:
  replicas: 1
  selector:
    matchLabels:
      app: zenith-landing
  template:
    spec:
      containers:
      - name: landing
        image: zenith-landing:latest
        imagePullPolicy: Never  # local image
        ports:
        - containerPort: 3000
---
apiVersion: v1
kind: Service
metadata:
  name: landing
  namespace: zenith-platform
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: freezenith-tls
  namespace: zenith-platform
spec:
  secretName: freezenith-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - freezenith.com
  - www.freezenith.com
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: freezenith-landing
  namespace: zenith-platform
spec:
  entryPoints: [websecure]
  routes:
  - match: Host(`freezenith.com`) || Host(`www.freezenith.com`)
    kind: Rule
    services:
    - name: landing
      port: 3000
  tls:
    secretName: freezenith-tls
```

### Test Case 3:
- [ ] `nslookup freezenith.com` resolves to 161.35.82.211
- [ ] `kubectl get pods -n zenith-platform` shows landing pod Running
- [ ] `kubectl get certificate -n zenith-platform` shows freezenith-tls Ready=True
- [ ] `curl -I https://freezenith.com` returns 200
- [ ] `curl -I https://www.freezenith.com` returns 200 (or redirect)
- [ ] Browser: https://freezenith.com loads beautiful landing page
- [ ] SSL certificate is valid (Let's Encrypt)

---

## Phase 4: Deploy Mission Control

### Task 4.1: Create DNS records
```
mission.freezenith.com  A  161.35.82.211
api.freezenith.com      A  161.35.82.211
```

### Task 4.2: Build Mission Control Docker image
```
docker build -t zenith-mc:latest -f apps/mission-control/Dockerfile .
```

### Task 4.3: Build API server Docker image
```
docker build -t zenith-api:latest -f services/api/Dockerfile .
```

### Task 4.4: Deploy Mission Control + API
```yaml
# Deployment: mission-control (port 3100)
# Deployment: zenith-api (port 8080)
# Service: mission-control, zenith-api
# IngressRoute: mission.freezenith.com -> mission-control
# IngressRoute: api.freezenith.com -> zenith-api
# Certificate: mission.freezenith.com, api.freezenith.com
```

### Test Case 4:
- [ ] `curl -I https://mission.freezenith.com` returns 200
- [ ] `curl -I https://api.freezenith.com/health` returns 200
- [ ] Browser: Mission Control dashboard loads
- [ ] Mission Control can reach API (no CORS errors)
- [ ] SSL certificates valid for both subdomains

---

## Phase 5: Customer Test - embermind.com

### Task 5.1: Create Cloudflare DNS for embermind.com
```
embermind.com       A  161.35.82.211
*.embermind.com     A  161.35.82.211  (wildcard for app subdomains)
```

### Task 5.2: Create tenant namespace
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: zenith-embermind
  labels:
    zenith.dev/tenant: embermind
    zenith.dev/domain: embermind.com
```

### Task 5.3: Deploy Zenith Web Platform for embermind
```yaml
# Deployment: zenith-web (port 3000)
# IngressRoute: embermind.com -> zenith-web
# Certificate: embermind.com
```

### Task 5.4: Test full platform workflow
Through Mission Control or API:
1. Create a Project for embermind
2. Deploy a sample app (nginx hello-world)
3. Create a PostgreSQL database
4. Assign domain app.embermind.com to the app
5. Verify everything works end-to-end

### Test Case 5:
- [ ] `curl -I https://embermind.com` returns 200 (Zenith Web Platform)
- [ ] Can log into embermind.com web platform
- [ ] Can create a project via the platform
- [ ] Can deploy a sample app
- [ ] Can create a PostgreSQL database
- [ ] `curl https://app.embermind.com` shows the deployed app
- [ ] Database connection string works
- [ ] Mission Control shows embermind as a tenant

---

## Phase 6: End-to-End Smoke Tests

### Automated test script: `infra/scripts/e2e-test.sh`
```bash
#!/bin/bash
set -e

echo "=== Zenith E2E Tests ==="

# Test 1: Landing page
echo "[1/8] Testing freezenith.com..."
curl -sf https://freezenith.com | grep -q "Zenith" && echo "PASS" || echo "FAIL"

# Test 2: Mission Control
echo "[2/8] Testing mission.freezenith.com..."
curl -sf https://mission.freezenith.com | grep -q "Mission" && echo "PASS" || echo "FAIL"

# Test 3: API health
echo "[3/8] Testing api.freezenith.com..."
curl -sf https://api.freezenith.com/health | grep -q "ok" && echo "PASS" || echo "FAIL"

# Test 4: Customer platform
echo "[4/8] Testing embermind.com..."
curl -sf https://embermind.com | grep -q "Zenith" && echo "PASS" || echo "FAIL"

# Test 5: SSL certificates
echo "[5/8] Checking SSL certificates..."
echo | openssl s_client -connect freezenith.com:443 2>/dev/null | grep -q "CN=freezenith.com" && echo "PASS" || echo "FAIL"

# Test 6: API create project
echo "[6/8] Creating test project..."
curl -sf -X POST https://api.freezenith.com/api/v1/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"e2e-test"}' && echo "PASS" || echo "FAIL"

# Test 7: API create database
echo "[7/8] Creating test database..."
curl -sf -X POST https://api.freezenith.com/api/v1/projects/e2e-test/databases \
  -H "Content-Type: application/json" \
  -d '{"name":"testdb","engine":"postgresql","version":"16","storage":"1Gi"}' && echo "PASS" || echo "FAIL"

# Test 8: DNS resolution
echo "[8/8] Checking DNS resolution..."
dig +short freezenith.com | grep -q "161.35.82.211" && echo "PASS" || echo "FAIL"

echo "=== Done ==="
```

### Test Case 6:
- [ ] `infra/scripts/e2e-test.sh` passes all 8 tests
- [ ] All changes committed to git
- [ ] Server state matches code (no manual changes)

---

## File Structure (what gets created/modified)

```
/opt/zenith/
├── apps/
│   ├── landing/
│   │   ├── Dockerfile          # NEW
│   │   └── src/                # REDESIGNED
│   ├── mission-control/
│   │   └── Dockerfile          # NEW
│   └── web/
│       └── Dockerfile          # NEW
├── services/
│   └── api/
│       └── Dockerfile          # NEW
├── infra/
│   ├── k8s/                    # K8s manifests
│   │   ├── namespace.yaml
│   │   ├── landing.yaml
│   │   ├── mission-control.yaml
│   │   ├── api.yaml
│   │   ├── certificates.yaml
│   │   └── tenant-embermind.yaml
│   └── scripts/                # Utility scripts
│       ├── deploy.sh           # Full deployment script
│       ├── e2e-test.sh         # E2E test runner
│       └── cloudflare-dns.sh   # DNS record management
└── DEPLOYMENT-PLAN.md          # THIS FILE
```

---

## Rollback Plan

If anything breaks:
```bash
# Roll back deployments
kubectl rollout undo deployment/landing -n zenith-platform
kubectl rollout undo deployment/mission-control -n zenith-platform

# Delete tenant
kubectl delete namespace zenith-embermind

# Remove DNS records (via Cloudflare API)
./infra/scripts/cloudflare-dns.sh delete freezenith.com
```

---

## Summary

| Phase | What | Domain | Status |
|-------|------|--------|--------|
| 1 | Server cleanup | - | TODO |
| 2 | Landing page design + build | - | TODO |
| 3 | Deploy landing | freezenith.com | TODO |
| 4 | Deploy Mission Control + API | mission.freezenith.com, api.freezenith.com | TODO |
| 5 | Customer test | embermind.com | TODO |
| 6 | E2E smoke tests | all | TODO |

**Total DNS records needed:**
- `freezenith.com` → 161.35.82.211
- `www.freezenith.com` → 161.35.82.211
- `mission.freezenith.com` → 161.35.82.211
- `api.freezenith.com` → 161.35.82.211
- `embermind.com` → 161.35.82.211
- `*.embermind.com` → 161.35.82.211
