## 1. Ansible Foundation
- [x] 1.1 Create `infra/ansible/` directory structure (ansible.cfg, requirements.yml, inventory/, group_vars/, vault/, playbooks/, roles/)
- [x] 1.2 Write `ansible.cfg` (inventory path, roles path, vault settings, SSH pipelining, retry files disabled)
- [x] 1.3 Write `requirements.yml` (kubernetes.core, community.general, community.crypto collections)
- [x] 1.4 Write `inventory/production.yml` (ghasi server, current domains, current namespaces)
- [x] 1.5 Write `inventory/staging.yml` (staging server or same server with staging namespace)
- [x] 1.6 Write `group_vars/all.yml` (shared defaults: image names, k3s paths, resource limits)
- [x] 1.7 Write `group_vars/production.yml` (production overrides: domains, replicas, resource limits)
- [x] 1.8 Write `group_vars/staging.yml` (staging overrides: staging domains, lower resources)
- [ ] 1.9 Create vault files with `ansible-vault create` for staging and production secrets (JWT secret, DB password, admin password, Cloudflare token)

## 2. Base Roles (Server + K8s)
- [x] 2.1 `roles/common/` — apt update, install Docker, install pip packages, configure firewall (ufw), swap off
- [x] 2.2 `roles/k3s/` — install/upgrade k3s, wait for node Ready, configure kubeconfig
- [x] 2.3 `roles/cert-manager/` — Helm install cert-manager, create letsencrypt-prod ClusterIssuer, wait for webhook ready
- [x] 2.4 `roles/traefik-config/` — Traefik middlewares (redirect-to-https), TLS options

## 3. Zenith App Deployment Roles
- [x] 3.1 `roles/zenith-build/` — Docker build 6 images (landing, mc, web, api, mc-demo, web-demo) with parameterized tags
- [x] 3.2 `roles/zenith-import/` — docker save | k3s ctr images import for each image
- [x] 3.3 `roles/zenith-namespaces/` — create namespaces (zenith-platform, zenith-embermind, customer namespaces), create K8s secrets from vault vars
- [x] 3.4 `roles/zenith-api/` — templated API deployment + service (from infra/k8s/api.yaml), wait for rollout
- [x] 3.5 `roles/zenith-landing/` — templated landing deployment + service
- [x] 3.6 `roles/zenith-mc/` — templated MC deployment + service (real mode for customers, demo mode for platform)
- [x] 3.7 `roles/zenith-web/` — templated Web deployment + service (real mode + demo mode)
- [x] 3.8 `roles/zenith-ingress/` — templated IngressRoutes + Certificates (from infra/k8s/ingress.yaml, infra/k8s/certificates.yaml), supports dynamic customer domains

## 4. Infrastructure Roles
- [x] 4.1 `roles/postgres/` — PostgreSQL StatefulSet + Service (from infra/k8s/postgres.yaml), wait for ready, verify DB connectivity
- [x] 4.2 `roles/keda/` — Helm install KEDA + HTTP Add-on (from infra/k8s/keda/), apply cold-start service + error middleware
- [x] 4.3 `roles/monitoring/` — Helm install kube-prometheus-stack + Loki + Promtail (from infra/helm/monitoring/ values)
- [x] 4.4 `roles/dns/` — Run Terraform apply for Cloudflare DNS (or direct Cloudflare API via community.general.cloudflare_dns)

## 5. Playbooks
- [x] 5.1 `playbooks/site.yml` — full deployment: common → k3s → cert-manager → postgres → namespaces → build → import → deploy apps → ingress → keda → monitoring → dns
- [x] 5.2 `playbooks/infra.yml` — infrastructure only: common → k3s → cert-manager → postgres → keda → monitoring
- [x] 5.3 `playbooks/apps.yml` — apps only: build → import → deploy apps → ingress
- [x] 5.4 `playbooks/build.yml` — build images only (no deploy)
- [x] 5.5 `playbooks/teardown.yml` — remove all Zenith resources (with confirmation prompt)

## 6. Testing & Validation
- [ ] 6.1 Run full `site.yml` on staging — verify all pods Running, all endpoints responding
- [ ] 6.2 Run `apps.yml` on staging — verify incremental deploy works (no infra changes)
- [ ] 6.3 Run full `site.yml` on production — verify existing services unaffected, same endpoints work
- [ ] 6.4 Run idempotency test — execute `site.yml` twice, verify no changes on second run
- [x] 6.5 Document usage in project README or `infra/ansible/README.md`

## 7. Cleanup
- [ ] 7.1 Mark `infra/scripts/deploy.sh` as deprecated (add comment header pointing to Ansible)
- [ ] 7.2 Mark `infra/scripts/cloudflare-dns.sh` as deprecated
- [ ] 7.3 Mark `infra/k8s/keda/install.sh` as deprecated
- [ ] 7.4 Update `MEMORY.md` redeploy command to use Ansible
