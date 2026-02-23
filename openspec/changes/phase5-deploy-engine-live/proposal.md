# Phase 5: Deploy Engine Live — Users Deploy Apps from Git

## Summary

Enable the full app deployment flow on staging: user connects a Git repo, pushes code, Zenith builds and deploys it automatically with Kaniko.

## Prerequisites

- Phase 4 complete (projects + namespaces)
- Harbor registry accessible from k3s (for Kaniko to push built images)

## Steps

### Step 5.1: Kaniko Build Pipeline on Staging

**What:** The deploy engine runs Kaniko jobs in k3s to build Docker images from user repos.

**Build:**
- API config: `REGISTRY=registry.stage.freezenith.com/zenith-stage`
- Kaniko pushes built images to Harbor
- Create k8s ServiceAccount + Secret for Kaniko to auth with Harbor
- `BUILD_WORKDIR` configured on persistent storage

**Your manual work:**
1. Create Harbor robot account `kaniko-push` with push access to `zenith-stage`
2. Create k8s secret with Harbor credentials for Kaniko

**Verify:**
```bash
# Create an app from a git repo
curl -X POST https://api.stage.freezenith.com/api/v1/apps \
  -H 'Authorization: Bearer TOKEN' \
  -d '{"name":"test-app","repo_url":"https://github.com/user/repo","branch":"main"}'

# Watch build logs
curl https://api.stage.freezenith.com/api/v1/apps/{id}/deployments/{did}/logs
# SSE stream of build output
```

### Step 5.2: K8s Resource Deployment

**What:** After build succeeds, Deployer creates Deployment + Service + IngressRoute in the user's namespace.

**Build:**
- `DeployApp()` creates resources in `zen-{project}` namespace
- IngressRoute with TLS for `{app-name}.apps.stage.freezenith.com`
- Health checks configured
- Resource limits based on plan tier

**Your manual work:** None

**Verify:**
```bash
kubectl get deploy -n zen-my-project
# test-app deployment running

curl https://test-app.apps.stage.freezenith.com
# App response
```

### Step 5.3: GitHub Webhook Integration

**What:** Push to GitHub triggers automatic rebuild and deploy.

**Build:**
- Webhook endpoint: `POST /api/v1/webhooks/github`
- HMAC-SHA256 signature verification
- `findAppsByRepo()` matches push to registered apps
- Triggers `pipeline.TriggerBuild()`

**Your manual work:**
1. Configure GitHub webhook on a test repo:
   - URL: `https://api.stage.freezenith.com/api/v1/webhooks/github`
   - Secret: (from `GITHUB_WEBHOOK_SECRET` env var)
   - Events: Push

**Verify:**
1. Push code to the test repo
2. Watch MC or API logs — build triggered
3. New deployment appears in deployment history
4. App updated with new code

### Step 5.4: Web Platform Deploy UI

**What:** Web Platform `/deploy` page shows real apps, real builds, real logs.

**Build:**
- Deploy page fetches from real `appsDeploy` API
- "Deploy from Git" modal creates real apps
- Build log viewer streams real SSE logs
- Deployment history shows real deployments with rollback

**Your manual work:** None

**Verify:**
1. Go to Web Platform → Deploy
2. Click "Deploy from Git" → enter repo URL
3. Watch build log in real-time
4. See app appear in the deploy grid with green status

## Acceptance Criteria

- [ ] User deploys app from Git repo via Web Platform
- [ ] Kaniko builds Docker image inside k3s
- [ ] Built image pushed to Harbor
- [ ] App accessible at `{name}.apps.stage.freezenith.com`
- [ ] GitHub webhook triggers auto-deploy on push
- [ ] Build logs stream in real-time via SSE
- [ ] Deployment history with rollback works
- [ ] Resource limits enforced per tier
