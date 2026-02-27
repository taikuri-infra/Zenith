# Migration Plan: V1 to V2

> **Status:** Planning
> **Risk Level:** High (production is live)
> **Strategy:** Blue-Green — build V2 alongside V1, switch when ready

---

## Current State (V1)

### Production (161.35.82.211 — DigitalOcean)
- Manual deployment via `scripts/deploy.sh`
- Raw K8s manifests in `infra/k8s/`
- No CI/CD, no GitOps, no backup
- Kong DB-less (API only)
- No Keycloak (JWT managed in zenith-api)
- Flannel CNI (no NetworkPolicy)
- One shared PostgreSQL (CNPG, one DB for everything)

### Staging (77.42.88.149 — Hetzner)
- Terraform + Helm managed
- Harbor registry running
- Kong, KEDA, monitoring installed
- CNPG operator with one shared cluster
- Monolithic Helm chart (recently split into 5 charts)

### What's Live
| URL | Service | Status |
|-----|---------|--------|
| freezenith.com | Landing page | Live |
| api.freezenith.com | Go API | Live |
| demo-ms.freezenith.com | MC Demo | Live |
| demo-cloud.freezenith.com | Web Demo | Live |
| ms.embermind.app | Customer MC | Live |
| cloud.embermind.app | Customer Web | Live (HTTP 500 bug) |

---

## Migration Strategy

**We do NOT migrate V1 in-place.** Instead:

1. Build V2 on a NEW Hetzner server (staging-v2)
2. Test everything on staging-v2
3. When ready, point DNS from production to staging-v2
4. Decommission old servers

This is safer because:
- V1 production stays running during entire migration
- If V2 has issues, we can point DNS back to V1
- No risk of breaking production during migration
- Clean start, no legacy state to deal with

---

## Migration Phases

### Phase M1: New Staging Server (Week 1-2)

```
Action: Create new Hetzner server for V2 staging
```

1. Create new Hetzner server (cx42 or bigger for V2 components)
2. Run Phase 1 Terraform (new server + Cloudflare DNS for v2.stage.freezenith.com)
3. Run Phase 2 Ansible (k3s + Cilium + hcloud-csi)
4. Run Phase 3 Terraform (all V2 infra components)
5. Verify: all infra running (cert-manager, APISIX, Keycloak, CNPG, ArgoCD, etc.)

**Validation checklist:**
- [ ] k3s running with Cilium CNI
- [ ] cert-manager issuing test certificate
- [ ] CNPG operator watching namespaces
- [ ] Keycloak accessible, admin realm working
- [ ] APISIX routing test requests
- [ ] external-dns creating DNS records
- [ ] ArgoCD UI accessible
- [ ] Temporal UI accessible
- [ ] Harbor accessible (or reuse existing)
- [ ] Monitoring dashboards loading
- [ ] Hubble UI showing network flows

### Phase M2: Deploy Zenith Apps (Week 2-3)

```
Action: Deploy zenith-api, zenith-landing, zenith-admin via ArgoCD
```

1. Push Helm charts to Harbor
2. ArgoCD syncs application charts
3. Test: landing page accessible
4. Test: API health endpoint responding
5. Test: Keycloak login flow working
6. Test: APISIX JWT verification working

**Validation checklist:**
- [ ] Landing page loads at v2.stage.freezenith.com
- [ ] API responds at api.v2.stage.freezenith.com
- [ ] Keycloak login/register works
- [ ] APISIX correctly routes protected/public routes
- [ ] ArgoCD shows all apps synced and healthy

### Phase M3: Customer Provisioning (Week 3-4)

```
Action: Test Temporal customer provisioning workflow
```

1. Trigger provision-customer workflow via API
2. Verify: Keycloak realm created
3. Verify: Database created in shared CNPG cluster
4. Verify: S3 bucket created
5. Verify: K8s namespace with all resources
6. Verify: DNS record created
7. Verify: TLS certificate issued
8. Verify: Customer frontend accessible
9. Verify: Customer backend API through APISIX with JWT

**Validation checklist:**
- [ ] Full provisioning workflow completes without errors
- [ ] Customer can log in via Keycloak
- [ ] Customer frontend loads
- [ ] Customer API calls work through APISIX
- [ ] Network isolation: customer A cannot reach customer B namespace
- [ ] ResourceQuota enforced
- [ ] Backup CronJobs running

### Phase M4: Data Migration (Week 4-5)

```
Action: Migrate existing customer data from V1 to V2
```

1. pg_dump from V1 production PostgreSQL
2. Create customer DB in V2 CNPG cluster
3. pg_restore into V2
4. Migrate Keycloak realm (or recreate users)
5. Migrate S3 data (if any)
6. Verify: customer can access their data on V2

**For embermind.app (current customer):**
- pg_dump embermind DB from production
- Create zenith-embermind namespace on V2
- pg_restore into V2 CNPG cluster
- Create Keycloak realm for embermind
- Update DNS: ms.embermind.app → V2 server
- Update DNS: cloud.embermind.app → V2 server
- Test: customer can log in and see their data

### Phase M5: DNS Cutover (Week 5-6)

```
Action: Point production DNS to V2 server
```

1. Update Cloudflare DNS: freezenith.com → V2 server IP
2. Update Cloudflare DNS: api.freezenith.com → V2 server IP
3. Update Cloudflare DNS: ms.embermind.app → V2 server IP
4. Update Cloudflare DNS: cloud.embermind.app → V2 server IP
5. Enable Cloudflare proxy (WAF + DDoS protection)
6. Monitor: check all endpoints are responding
7. Keep V1 server running for 2 weeks (rollback safety)

**Rollback plan:**
If V2 has issues after cutover:
- Point DNS back to V1 server IP (< 5 minute recovery)
- V1 data may be slightly behind (since cutover)
- Investigate and fix V2 issues
- Re-attempt cutover

### Phase M6: Cleanup (Week 7)

```
Action: Decommission old infrastructure
```

1. Verify V2 has been running smoothly for 2+ weeks
2. Take final backup of V1 server
3. Delete V1 DigitalOcean server
4. Delete V1 Hetzner staging server (if separate from V2)
5. Clean up old DNS records
6. Archive V1 Terraform state
7. Update all documentation to reflect V2

---

## Risk Matrix

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| V2 infra component fails | High | Medium | Keep V1 running, DNS rollback in 5min |
| Data loss during migration | Critical | Low | pg_dump before and after, verify data |
| Customer downtime during cutover | Medium | Medium | Low-TTL DNS, cutover during low traffic |
| Keycloak realm misconfiguration | High | Medium | Test thoroughly on staging first |
| APISIX routing misconfiguration | High | Medium | Compare V1 Kong routes with V2 APISIX |
| Cilium NetworkPolicy blocks legit traffic | Medium | Medium | Start with audit mode, then enforce |
| cert-manager DNS-01 fails | Medium | Low | Test with staging cert first |

---

## Timeline Summary

```
Week 1-2:  Phase M1 — New server + infra bootstrap
Week 2-3:  Phase M2 — Deploy Zenith apps
Week 3-4:  Phase M3 — Test customer provisioning
Week 4-5:  Phase M4 — Migrate data
Week 5-6:  Phase M5 — DNS cutover
Week 7:    Phase M6 — Cleanup
```

Total: ~6-7 weeks for full migration with safety margins.
