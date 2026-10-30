# Organization model + Compute UI — design

Sub-project 2 of 7. Depends on: Zenith on k3s (done). Feeds: edge/security (3), observability (4).

## Intent
Inside a Project, users group resources into **Organizations**. An Org is an isolation
boundary: its own Kubernetes namespace, so WAF/Firewall (sub-project 3) and logs/metrics
(sub-project 4) can be scoped per Org. Self-host only, k3s only, no SaaS tiers.

## Model
Project → Org → { App, Database, Bucket, Domain }.

- `orgs(id, project_id, name, slug, namespace, created_at)`; unique `(project_id, slug)`.
- `namespace` is stored on the row: `zo-<project8>-<slug>` for new Orgs. The existing
  `zenith-apps` namespace is kept as the namespace of each project's migrated `default` Org
  (no redeploy of running apps).
- `org_id` (NOT NULL after backfill) added to `apps`, `databases`, `user_buckets`, `domains`.
- Migration: create a `default` Org per existing project, backfill `org_id`, then set NOT NULL.

## Kubernetes
- Creating an Org creates its namespace with labels `zenith.dev/project`, `zenith.dev/org`,
  a default-deny ingress NetworkPolicy (allow from ingress controller + same namespace),
  a ResourceQuota (defaults in chart values, editable per Org later).
- `NamespaceFor(org)` replaces every hard-coded `zenith-apps` in deploy, env vars, compose,
  DB backend (`K8sDBBackend`), log source, reconciler, gateway.
- Deleting an Org deletes the namespace (cascades apps, DB CRs, PVCs). Requires typed-name
  confirmation; refused if it still holds a managed DB unless `?force=true`.
- Reconciler iterates Orgs; recreates missing namespaces first.

## API
- `GET/POST /projects/:pid/orgs`, `GET/PATCH/DELETE /orgs/:id`.
- Create endpoints for apps/databases/buckets/domains accept `org_id`; if omitted they use
  the project's `default` Org (keeps CLI/E2E/older clients working).
- List endpoints accept `?org_id=`; ownership checks go through Org → Project → user.
- `zen` CLI: `--org` flag, defaults to `default`. Backup/restore cover all Org namespaces.

## UI (Compute)
- Sidebar: Project switcher stays; add an Org switcher under it ("All orgs" + each Org).
- Apps page: list → App detail shows its **containers** (pods/containers, status, restarts,
  image) from the Deployment's pods.
- Databases, Storage (buckets), Domains: grouped by Org, filter by the switcher.
- Remove Queues from the sidebar and delete the page. Marketplace stays hidden.
- Org settings: rename, quota, delete.

## Out of scope (later sub-projects)
Per-Org WAF/Firewall rules (3), per-Org log/metric search (4), RBAC between Orgs (members
stay at Project level for now), moving resources between Orgs.

## Testing
- Unit: migration backfill, `NamespaceFor`, ownership checks, default-Org fallback.
- Handler tests with the memory repos; reconciler test with two Orgs.
- E2E (k3d): create Org → deploy app + DB in it → verify namespace, NetworkPolicy isolation
  (app in Org A cannot reach Org B), delete Org removes namespace; legacy default Org still works.

## Decisions
1. **Namespace name** `zo-<project8>-<slug>`. (Approved)
2. **Default Org** auto-created per project, used when `org_id` is omitted. (Approved)
3. **Quota defaults** — none (unlimited) at first; editable per Org later. (Approved)
