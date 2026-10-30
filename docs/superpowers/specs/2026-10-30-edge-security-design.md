# Edge and security — design (sub-project 3 of 7)

Depends on: project = namespace (sub-project 2). Feeds: observability (4).

## Intent
Every Project (namespace) gets its own gateway routes, WAF and firewall rules, and users can
sign in through one SSO. Self-host only, k3s only, operators/Helm only, minimal install:
every component here is optional and installed by `install-k3s.sh` flags.

## Findings in the repo today
- Traefik (k3s default) is the edge; apps are published through Traefik IngressRoutes.
- Code already has an APISIX path (`generateIngressRouteViaApisix*`, `GatewayService`), unused on k3s.
- `waf.go`, `network_policy.go`, `sso.go`, `keycloakclient` exist as API/UI but do nothing on k3s.
- Each project namespace already gets a `project-isolation` NetworkPolicy (sub-project 2).

## Design
1. **Gateway (APISIX)** — optional (`ZENITH_GATEWAY=apisix`): Helm-installs APISIX +
   ingress-controller. Apps with the gateway enabled get an `ApisixRoute` in their project
   namespace instead of a Traefik IngressRoute; everything else stays on Traefik. Traefik keeps
   serving the platform UI/API and cert HTTP-01 challenges.
2. **Firewall (per namespace)** — rules in the UI compile to standard Kubernetes
   `NetworkPolicy` (works on flannel/kube-router and Cilium). If Cilium is the CNI the same
   rules are emitted as `CiliumNetworkPolicy`, which adds FQDN egress and L7 rules.
   Cilium itself is NOT forced on existing clusters (changing the CNI means a reinstall);
   `ZENITH_CNI=cilium` is available for fresh installs.
3. **WAF (per project)** — enforced at the gateway with APISIX's Coraza (ModSecurity-compatible)
   plugin, rule sets (OWASP CRS toggles + custom rules) stored per project and rendered into
   the project's `ApisixRoute`/`PluginConfig`. Needs the gateway; without it the UI says so.
4. **SSO (Keycloak)** — optional (`ZENITH_SSO=keycloak`): Keycloak Operator in `zenith-system`,
   one realm `zenith`, OIDC client for the dashboard. Login page gains "Sign in with SSO";
   local password login remains. "Native" = Zenith creates the realm/client itself via
   `keycloakclient` at install; no manual Keycloak setup.

## Out of scope
Service mesh, mTLS between pods, rate limiting beyond what APISIX plugins give, external IdPs
(users add them in Keycloak themselves).

## Testing
- Unit: policy compilation (rule → NetworkPolicy/CiliumNetworkPolicy), route/WAF rendering.
- E2E (k3d): with `ZENITH_GATEWAY=apisix`, a request matching a WAF rule is blocked (403) and
  a normal one passes; a firewall rule blocks pod→pod across projects; SSO login via a test
  Keycloak user. Heavy components run only in a separate E2E job.

## Decisions (approved 2026-10-30)
1. Firewall compiles to NetworkPolicy; CiliumNetworkPolicy only when Cilium is present. Cilium is not required.
2. APISIX runs next to Traefik as an optional component; it does not replace Traefik.
3. Keycloak is optional at install; SSO is off by default.
