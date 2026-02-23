# Phase 4: Customer Onboarding — Projects & Namespaces

## Summary

Enable the core SaaS flow: a customer signs up, creates a project, gets a namespace with resource quotas based on their tier (Free/Pro). Mission Control can see and manage all customers.

## Prerequisites

- Phase 3 complete (auth works)
- API server with PostgreSQL (persistent)

## Steps

### Step 4.1: Project Creation Creates Namespace

**What:** When a user creates a project via API, a Kubernetes namespace is created with resource quotas.

**Build:**
- Wire `ProjectReconciler` in operator OR handle in API directly
- Namespace naming: `zen-{project-id}` or `zen-{project-name}`
- Apply `ResourceQuota` based on plan:
  - Free: 256Mi RAM, 250m CPU, 1 pod max
  - Pro: 4Gi RAM, 2 CPU, 10 pods max
- Apply `CiliumNetworkPolicy` for namespace isolation

**Your manual work:** None

**Verify:**
```bash
# Create project via API
curl -X POST https://api.stage.freezenith.com/api/v1/projects \
  -H 'Authorization: Bearer TOKEN' \
  -d '{"name":"my-project","plan":"free"}'

# Check namespace created
kubectl get ns zen-my-project
kubectl get resourcequota -n zen-my-project
```

### Step 4.2: MC Shows Customers & Projects

**What:** Mission Control admin dashboard shows real customer data.

**Build:**
- MC `/tenants` page fetches from `/api/v1/admin/tenants`
- Shows: project name, plan, resource usage, status
- MC dashboard stats reflect real counts

**Your manual work:** None

**Verify:**
1. Log in to MC as admin
2. Go to Tenants page
3. See the project created in Step 4.1

### Step 4.3: Web Platform Project Scoping

**What:** After login, Web Platform is scoped to the user's project.

**Build:**
- Project selector if user has multiple projects
- All API calls scoped to `projectId`
- Dashboard shows project-specific stats

**Your manual work:** None

**Verify:**
1. Login to Web Platform as the user who created the project
2. See project dashboard with correct resource limits

### Step 4.4: Customer DNS Subdomain

**What:** Each project gets a subdomain for accessing their apps.

**Build:**
- Pattern: `*.{project-name}.stage.freezenith.com`
- Wildcard DNS record OR per-app DNS via Cloudflare API
- cert-manager handles TLS

**Your manual work:**
- Add wildcard DNS: `*.apps.stage.freezenith.com → 77.42.88.149`

**Verify:**
```bash
# After deploying an app named "myapp" in project "demo"
curl https://myapp.apps.stage.freezenith.com
```

## Acceptance Criteria

- [ ] User creates project → namespace + ResourceQuota created in k8s
- [ ] Free tier resource limits enforced
- [ ] Pro tier resource limits enforced
- [ ] MC shows all customers and their resource usage
- [ ] Web Platform scoped to user's project
- [ ] Cilium network policy prevents cross-namespace traffic
- [ ] Customer subdomain resolves and serves apps
