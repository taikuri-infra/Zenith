# Phase 6: Data Services — Databases & Storage

## Summary

Enable per-customer database provisioning (PostgreSQL via CloudNativePG) and object storage (Hetzner S3 or MinIO) so customers can create and manage data services through the Web Platform.

## Prerequisites

- Phase 5 complete (apps deploying)
- k3s cluster with sufficient storage

## Steps

### Step 6.1: Install CloudNativePG Operator

**What:** Deploy the CloudNativePG operator via Terraform Helm release.

**Build:**
- Add `helm_release.cnpg` to k8s-platform module
- CloudNativePG CRD: `Cluster` creates PostgreSQL instances
- Operator watches all namespaces

**Your manual work:** None — Terraform handles it.

**Verify:**
```bash
kubectl get pods -n cnpg-system
# cnpg-controller-manager running
```

### Step 6.2: Database Provisioning via API

**What:** API creates CloudNativePG `Cluster` CRD when user creates a database.

**Build:**
- `POST /api/v1/projects/:id/databases` → creates CNPG Cluster in user's namespace
- Database options: PostgreSQL (initially — MySQL/Redis later)
- Connection string generated and returned
- Resource limits based on tier:
  - Free: 256Mi RAM, 1Gi storage
  - Pro: 1Gi RAM, 10Gi storage

**Your manual work:** None

**Verify:**
```bash
curl -X POST https://api.stage.freezenith.com/api/v1/projects/PROJECT_ID/databases \
  -H 'Authorization: Bearer TOKEN' \
  -d '{"name":"mydb","engine":"postgresql","storage":"1Gi"}'

kubectl get clusters.postgresql.cnpg.io -n zen-my-project
# mydb cluster running
```

### Step 6.3: Storage Provisioning

**What:** Users can create S3-compatible object storage buckets.

**Build:**
- Option A: Hetzner Object Storage (API-based provisioning)
- Option B: MinIO in-cluster (simpler for staging)
- `POST /api/v1/projects/:id/storage` → creates bucket
- Access credentials returned to user
- Tier limits:
  - Free: 1 bucket, 1GB
  - Pro: 10 buckets, 50GB

**Your manual work:**
- If MinIO: nothing (deployed via Helm)
- If Hetzner S3: provide Hetzner S3 API credentials

**Verify:**
```bash
curl -X POST https://api.stage.freezenith.com/api/v1/projects/PROJECT_ID/storage \
  -H 'Authorization: Bearer TOKEN' \
  -d '{"name":"my-bucket","access":"private"}'
# Returns: endpoint, access_key, secret_key
```

### Step 6.4: Web Platform Data Services UI

**What:** Database and storage pages show real data, create/delete works.

**Build:**
- `/databases` page: list real databases, show connection strings
- `/databases/[name]`: detail page with metrics
- `/storage` page: list real buckets, show access info

**Your manual work:** None

**Verify:**
1. Go to Web Platform → Databases
2. Create a new PostgreSQL database
3. See it appear with connection string
4. Create a storage bucket
5. See it with access credentials

## Acceptance Criteria

- [ ] CloudNativePG operator running in cluster
- [ ] Users can create PostgreSQL databases via Web Platform
- [ ] Connection strings generated and accessible
- [ ] Storage buckets can be created and managed
- [ ] Tier-based resource limits enforced
- [ ] Databases survive pod restarts (persistent volumes)
- [ ] Database backups configurable (CNPG feature)
