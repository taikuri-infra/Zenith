# Organization model Implementation Plan

**Goal:** Project → Org → resources, each Org its own namespace.
**Spec:** docs/superpowers/specs/2026-10-30-organization-model-design.md
Execution: native, in slices, one PR each, CI green before merge.

## Slices
1. **Data + API** — migration 053 (`orgs`, `org_id` on apps/user_databases/user_buckets/domains, default Org backfill, namespace `zenith-apps` for migrated Orgs); `entities.Org`; `ports.OrgRepository` (postgres + memory); `OrgHandler` CRUD under `/projects/:pid/orgs` and `/orgs/:id`; `org_id` accepted on create endpoints, default Org fallback; `?org_id=` list filter; tests.
2. **Kubernetes** — `OrgService` creates/deletes the namespace (labels, default-deny NetworkPolicy, optional ResourceQuota); replace hard-coded `zenith-apps` in deploy, env vars, compose, `K8sDBBackend`, `K8sLogSource`, reconciler with the Org's namespace; RBAC for namespaces in the chart; reconciler iterates Orgs.
3. **UI** — Org switcher, Apps → containers view (pods API), resources grouped by Org, remove Queues page + nav entry, Org settings.
4. **CLI + E2E** — `--org` flag, backup/restore over all Org namespaces, k3d E2E: two Orgs isolated, delete removes namespace, legacy default Org keeps working.

## Global constraints
Default Org used when `org_id` omitted; existing deployments must not restart; no new runtime besides Kubernetes; tests run in CI (dev droplet is shared — avoid heavy local builds).
