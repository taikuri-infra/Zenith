# Edge and security 3a — Firewall Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The existing Firewall page/API stops being an in-memory mock: rules are stored in Postgres and compiled into a real Kubernetes policy in the app's project namespace.

**Architecture:** Rules stay per app (the existing UI and routes, `/apps/:appId/network-policies`). A pure compiler turns an app's enabled rules into one policy object named `zenith-fw-<appID[:8]>` that selects the app's pods by `zenith.dev/app-id`. Standard `networking.k8s.io/v1 NetworkPolicy` is the default; when the cluster has the Cilium CRD the same rules compile to `cilium.io/v2 CiliumNetworkPolicy`, which also supports deny rules and FQDN egress. A `FirewallApplier` re-applies the policy after every rule change and deletes it when no enabled rules remain.

**Tech Stack:** Go (Fiber, pgx), Kubernetes NetworkPolicy / CiliumNetworkPolicy, Next.js page `apps/web/src/app/firewall/page.tsx` (existing), k3d E2E (`cli/e2e`).

**Spec:** `docs/superpowers/specs/2026-10-30-edge-security-design.md` (Design §2, Decision 1)

Follow-up plans (not this one): 3b gateway + WAF (APISIX, Coraza), 3c SSO (Keycloak). Each ships standalone.

## Global Constraints

- k3s/Kubernetes only; no new operator or Helm chart in this plan.
- Cilium is not required. Without Cilium: `deny` rules and `fqdns` are rejected with HTTP 400 and the message `deny rules and FQDN egress need Cilium`.
- Policies live in the app's project namespace (`App.ProjectID` → `deploy.ProjectNamespaces.For`), never hard-coded `zenith-apps`.
- Policy name: `zenith-fw-` + first 8 chars of app ID. Pod selector: `zenith.dev/app-id: <app.ID>`.
- Egress rules always also allow DNS: UDP and TCP port 53 to namespace `kube-system`.
- Self-host: no plan gating (`requireBusiness` only applies when `cfg.Mode == "saas"`).
- Ownership: every endpoint returns 404 unless `app.UserID == user_id`.
- Migration number: `054`.

## Review Focus

1. Rule with an invalid CIDR (`10.0.0.0/33`, `abc`) — expect 400, nothing stored or applied.
2. Disabling the last enabled rule — expect the K8s policy to be deleted, so traffic returns to the project-isolation default.
3. Port outside 1–65535 or protocol other than TCP/UDP — expect 400.
4. API restarts — rules come back from Postgres and `ReconcileAll` re-applies them.
5. Another user's app ID in the URL — expect 404 and no K8s call.

---

### Task 1: Persist firewall rules

**Files:**
- Create: `services/api/internal/adapters/postgres/migrations/054_firewall_rules.up.sql`, `054_firewall_rules.down.sql`
- Create: `services/api/internal/adapters/postgres/postgres_firewall.go`
- Create: `services/api/internal/adapters/memory/memory_firewall.go`
- Modify: `services/api/internal/ports/repositories.go` (add interface)
- Test: `services/api/internal/adapters/memory/memory_firewall_test.go`

**Interfaces:**
- Produces:
  ```go
  type FirewallRuleRepository interface {
      ListByApp(ctx context.Context, appID string) ([]entities.NetworkPolicyRule, error)
      ListAll(ctx context.Context) ([]entities.NetworkPolicyRule, error)
      Get(ctx context.Context, id string) (*entities.NetworkPolicyRule, error)
      Create(ctx context.Context, r *entities.NetworkPolicyRule) error
      Update(ctx context.Context, r *entities.NetworkPolicyRule) error
      Delete(ctx context.Context, id string) error
  }
  ```
  Constructors: `postgres.NewFirewallRuleRepository(pool *pgxpool.Pool)`, `memory.NewFirewallRuleRepository()`.

- [ ] **Step 1:** Migration up: table `firewall_rules` (`id UUID PK`, `user_id UUID NOT NULL`, `app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE`, `name TEXT NOT NULL`, `direction TEXT NOT NULL CHECK (direction IN ('ingress','egress'))`, `action TEXT NOT NULL CHECK (action IN ('allow','deny'))`, `enabled BOOLEAN NOT NULL DEFAULT true`, `priority INT NOT NULL DEFAULT 0`, `config JSONB NOT NULL DEFAULT '{}'`, `created_at`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`), index on `app_id`. Down: `DROP TABLE IF EXISTS firewall_rules`. Check the `apps.id` type in an earlier migration and match it.
- [ ] **Step 2:** Test `TestMemoryFirewallRepo`: create two rules for app A and one for app B; `ListByApp(A)` has 2 sorted by priority then created_at; `Update` changes `Enabled`; `Delete` removes; `Get` of a missing id returns an error.
- [ ] **Step 3:** Implement memory and postgres repos (config marshalled as JSON). Run `go test ./internal/adapters/memory/ -run Firewall` → PASS; `go build ./...` → OK.
- [ ] **Step 4:** Commit `feat(api): store firewall rules in postgres`.

### Task 2: Compile rules into a policy

**Files:**
- Create: `services/api/internal/services/firewall/compile.go`
- Test: `services/api/internal/services/firewall/compile_test.go`

**Interfaces:**
- Produces:
  ```go
  func PolicyName(appID string) string                                  // "zenith-fw-" + appID[:8]
  func Validate(r entities.NetworkPolicyRule, cilium bool) error         // input checks below
  func Compile(app *entities.App, ns string, rules []entities.NetworkPolicyRule, cilium bool) (*k8sclient.CRDObject, error) // nil when no enabled rules
  ```

`Validate`: CIDRs parse with `net.ParseCIDR`; ports 1–65535; protocol TCP or UDP (empty = TCP); namespace names match `^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$`; when `!cilium`, `Action == deny` or `len(FQDNs) > 0` returns the error `deny rules and FQDN egress need Cilium`.

`Compile` (enabled rules only, sorted by priority):
- NetworkPolicy (`networking.k8s.io/v1`): `podSelector.matchLabels{"zenith.dev/app-id": app.ID}`. Each ingress allow rule becomes one `ingress[]` entry: `from` = ipBlock per CIDR + namespaceSelector (`kubernetes.io/metadata.name`) per namespace + podSelector for `PodLabels`; `ports` from `Ports`. Egress works the same way with `to`, plus a DNS entry (kube-system, 53/UDP and 53/TCP) added once. `policyTypes` lists only the directions present.
- CiliumNetworkPolicy (`cilium.io/v2`): `endpointSelector` instead of `podSelector`. Allow rules map to `ingress`/`egress` (`fromCIDR`/`toCIDR`, `fromEndpoints`/`toEndpoints` with `k8s:io.kubernetes.pod.namespace`, `toPorts`, `toFQDNs` with `matchPattern`). Deny rules map to `ingressDeny`/`egressDeny`. When FQDNs are present, the DNS egress entry carries `rules.dns: [{matchPattern: "*"}]` so Cilium can see the lookups.

- [ ] **Step 1:** Tests:
  - `TestCompileNoEnabledRulesReturnsNil`
  - `TestCompileIngressCIDRAndPort` (exact `ingress[0].from[0].ipBlock.cidr == "10.0.0.0/8"`, `ports[0].port == 443`, `policyTypes == ["Ingress"]`)
  - `TestCompileEgressAddsDNS`
  - `TestCompileNamespaceAllow`
  - `TestCompileCiliumDenyAndFQDN` (`ingressDeny` present, `egress[*].toFQDNs[0].matchPattern == "api.stripe.com"`)
  - `TestValidate` table: bad CIDR, port 0, port 70000, protocol "ICMP", deny without Cilium, FQDN without Cilium, bad namespace name → error; good rule → nil
  - `TestPolicyName`
- [ ] **Step 2:** Run `go test ./internal/services/firewall/` → FAIL (undefined).
- [ ] **Step 3:** Implement. Run tests → PASS.
- [ ] **Step 4:** Commit `feat(api): compile firewall rules to NetworkPolicy or CiliumNetworkPolicy`.

### Task 3: Apply policies to the cluster

**Files:**
- Create: `services/api/internal/services/firewall/applier.go`
- Test: `services/api/internal/services/firewall/applier_test.go` (use `k8sclient` memory/fake client used by other deploy tests)

**Interfaces:**
- Consumes: Task 1 repo, Task 2 `Compile`/`PolicyName`.
- Produces:
  ```go
  type Applier struct { /* repo, apps ports.AppRepository, k8s k8sclient.Client, nsFor func(ctx, projectID) string */ }
  func NewApplier(repo ports.FirewallRuleRepository, apps ports.AppRepository, k8s k8sclient.Client, nsFor func(context.Context, string) string) *Applier
  func (a *Applier) Cilium(ctx context.Context) bool          // ListCRDsWithVersion("cilium.io/v2","CiliumNetworkPolicy","kube-system") succeeds; cached after first call
  func (a *Applier) Apply(ctx context.Context, appID string) error   // compile + create-or-update; delete (both kinds) when Compile returns nil
  func (a *Applier) ReconcileAll(ctx context.Context)                // Apply for every app that has rules; log errors, never fail startup
  ```

- [ ] **Step 1:** Tests:
  - `TestApplyCreatesPolicyInProjectNamespace`: nsFor returns `zp-ab12cd34-site`, so the CRD is created there with the right name.
  - `TestApplyDeletesWhenAllDisabled`
  - `TestApplyUpdatesExisting`: a second Apply with a changed port results in an update, not a duplicate.
- [ ] **Step 2:** Implement with `GetCRDWithVersion` → `UpdateCRD`/`CreateCRD`, `DeleteCRDWithVersion` (ignore not-found). Tests → PASS.
- [ ] **Step 3:** Commit `feat(api): apply firewall policies in the app's project namespace`.

### Task 4: Handler uses the repo and applier

**Files:**
- Modify: `services/api/internal/handlers/network_policy.go` (drop the in-memory map)
- Modify: `services/api/cmd/server/main.go` (around line 1534: construct repo + applier; `SetSelfHosted(cfg.Mode != "saas")`; under `cfg.K8sMode == "real"` call `go applier.ReconcileAll(ctx)` after startup)
- Test: `services/api/internal/handlers/network_policy_test.go`

**Interfaces:**
- Consumes: Tasks 1–3.
- Produces: `NewNetworkPolicyHandler(appRepo, planRepo, repo ports.FirewallRuleRepository)`, `SetApplier(a *firewall.Applier)`, `SetSelfHosted(bool)`. Routes are unchanged.

Behaviour: Create and Update call `firewall.Validate(rule, applier.Cilium(ctx))` → 400 with the error text on failure. After a successful store, `applier.Apply(ctx, appID)`; if apply fails, respond 502 `firewall rule saved but not applied: <err>` (the rule stays stored and ReconcileAll retries). Update and Delete check that `rule.AppID == :appId`, otherwise 404. When the applier is nil (memory mode), skip applying.

- [ ] **Step 1:** Tests (memory repos, nil applier, self-hosted):
  - `TestFirewallCreateInvalidCIDR400`
  - `TestFirewallCreateDenyWithoutCilium400`
  - `TestFirewallOtherUsersApp404`
  - `TestFirewallRuleFromOtherApp404` (update rule X via app B's URL)
  - `TestFirewallCRUDRoundTrip`
  - `TestFirewallNoPlanGateWhenSelfHosted`
- [ ] **Step 2:** Implement. `go test ./internal/handlers/ -run Firewall` → PASS; `go build ./...` OK.
- [ ] **Step 3:** Commit `feat(api): firewall rules are stored and enforced`.

### Task 5: Firewall page shows what is enforced

**Files:**
- Modify: `apps/web/src/app/firewall/page.tsx`
- Test: `apps/web/src/app/firewall/__tests__/page.test.tsx` (create; follow `apps/web/src/app/apps/[name]/__tests__/containers-card.test.tsx` for mocking style)

Behaviour: show the API's 400 message inline in the form (for example the Cilium message) instead of a generic error. Show 502 "saved but not applied" as a warning toast. Remove any "Business plan" upsell text when the app runs self-hosted (use the existing self-host flag the sidebar uses for `cloudOnly`).

- [ ] **Step 1:** Tests:
  - `shows server validation message`: the mock API rejects with 400 and the message `deny rules and FQDN egress need Cilium`, and the text is visible.
  - `no plan upsell in self-host`.
- [ ] **Step 2:** Implement; run `npx vitest run src/app/firewall` → PASS; `npx tsc --noEmit` clean.
- [ ] **Step 3:** Commit `feat(web): firewall page reports enforcement errors`.

### Task 6: E2E — a rule opens one project to another

**Files:**
- Create: `cli/e2e/02c_firewall_test.go` (build tag `e2e`; reuse helpers in `cli/e2e/helpers_test.go` and the projects from `02b_isolation_test.go`)

Scenario (k3d, flannel, no Cilium):
1. Two projects P1 and P2, each with one app. A curl pod in P2 cannot reach P1's app service (project-isolation).
2. Create an ingress allow rule on P1's app with `namespaces: [<P2 ns>]`. Assert the NetworkPolicy `zenith-fw-<id8>` exists in P1's namespace.
3. The curl pod in P2 now gets HTTP 200.
4. Disable the rule. The policy is gone and the curl fails again.
5. Creating a deny rule returns 400.

- [ ] **Step 1:** Write the test. Run locally only if the host is idle; otherwise rely on the CI E2E job.
- [ ] **Step 2:** Commit `test(e2e): firewall rules open and close cross-project traffic`.
