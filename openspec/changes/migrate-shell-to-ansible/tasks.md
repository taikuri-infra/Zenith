## 1. Ansible Foundation
- [ ] 1.1 Create `ansible/` directory structure (ansible.cfg, requirements.yml, inventory/, group_vars/, vault/, playbooks/, roles/)
- [ ] 1.2 Write `ansible.cfg` (inventory path, roles path, vault settings, SSH pipelining, retry files disabled)
- [ ] 1.3 Write `requirements.yml` (kubernetes.core, community.general, community.crypto collections)
- [ ] 1.4 Write `inventory/production.yml` (ghasi server, current domains, current namespaces)
- [ ] 1.5 Write `inventory/staging.yml` (staging server or same server with staging namespace)
- [ ] 1.6 Write `group_vars/all.yml` (shared defaults: image names, k3s paths, resource limits)
- [ ] 1.7 Write `group_vars/production.yml` (production overrides: domains, replicas, resource limits)
- [ ] 1.8 Write `group_vars/staging.yml` (staging overrides: staging domains, lower resources)
- [ ] 1.9 Create vault files with `ansible-vault create` for staging and production secrets (JWT secret, DB password, admin password, Cloudflare token)

## 2. Base Roles (Server + K8s)
- [ ] 2.1 `roles/common/` — apt update, install Docker, install pip packages, configure firewall (ufw), swap off
- [ ] 2.2 `roles/k3s/` — install/upgrade k3s, wait for node Ready, configure kubeconfig
- [ ] 2.3 `roles/cert-manager/` — Helm install cert-manager, create letsencrypt-prod ClusterIssuer, wait for webhook ready
- [ ] 2.4 `roles/traefik-config/` — Traefik middlewares (redirect-to-https), TLS options

## 3. Zenith App Deployment Roles
- [ ] 3.1 `roles/zenith-build/` — Docker build 6 images (landing, mc, web, api, mc-demo, web-demo) with parameterized tags
- [ ] 3.2 `roles/zenith-import/` — docker save | k3s ctr images import for each image
- [ ] 3.3 `roles/zenith-namespaces/` — create namespaces (zenith-platform, zenith-embermind, customer namespaces), create K8s secrets from vault vars
- [ ] 3.4 `roles/zenith-api/` — templated API deployment + service (from k8s/api.yaml), wait for rollout
- [ ] 3.5 `roles/zenith-landing/` — templated landing deployment + service
- [ ] 3.6 `roles/zenith-mc/` — templated MC deployment + service (real mode for customers, demo mode for platform)
- [ ] 3.7 `roles/zenith-web/` — templated Web deployment + service (real mode + demo mode)
- [ ] 3.8 `roles/zenith-ingress/` — templated IngressRoutes + Certificates (from k8s/ingress.yaml, k8s/certificates.yaml), supports dynamic customer domains

## 4. Infrastructure Roles
- [ ] 4.1 `roles/postgres/` — PostgreSQL StatefulSet + Service (from k8s/postgres.yaml), wait for ready, verify DB connectivity
- [ ] 4.2 `roles/keda/` — Helm install KEDA + HTTP Add-on (from k8s/keda/), apply cold-start service + error middleware
- [ ] 4.3 `roles/monitoring/` — Helm install kube-prometheus-stack + Loki + Promtail (from helm/monitoring/ values)
- [ ] 4.4 `roles/dns/` — Run Terraform apply for Cloudflare DNS (or direct Cloudflare API via community.general.cloudflare_dns)

## 5. Playbooks
- [ ] 5.1 `playbooks/site.yml` — full deployment: common → k3s → cert-manager → traefik → postgres → namespaces → build → import → deploy apps → ingress → keda → monitoring → dns
- [ ] 5.2 `playbooks/infra.yml` — infrastructure only: common → k3s → cert-manager → traefik → postgres → keda → monitoring
- [ ] 5.3 `playbooks/apps.yml` — apps only: build → import → deploy apps → ingress
- [ ] 5.4 `playbooks/build.yml` — build images only (no deploy)
- [ ] 5.5 `playbooks/teardown.yml` — remove all Zenith resources (with confirmation prompt)

## 6. Testing & Validation
- [ ] 6.1 Run full `site.yml` on staging — verify all pods Running, all endpoints responding
- [ ] 6.2 Run `apps.yml` on staging — verify incremental deploy works (no infra changes)
- [ ] 6.3 Run full `site.yml` on production — verify existing services unaffected, same endpoints work
- [ ] 6.4 Run idempotency test — execute `site.yml` twice, verify no changes on second run
- [ ] 6.5 Document usage in project README or `ansible/README.md`

## 7. Cleanup
- [ ] 7.1 Mark `scripts/deploy.sh` as deprecated (add comment header pointing to Ansible)
- [ ] 7.2 Mark `scripts/cloudflare-dns.sh` as deprecated
- [ ] 7.3 Mark `k8s/keda/install.sh` as deprecated
- [ ] 7.4 Update `MEMORY.md` redeploy command to use Ansible
