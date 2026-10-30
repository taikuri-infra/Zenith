# Project isolation + Compute UI — design (replaces the earlier Org idea)

Sub-project 2 of 7. No Organization level: the **Project** is the unit.

## Intent
A Project (e.g. "Site") is a self-contained group: its backend, frontend, databases,
buckets, domains, logs and monitoring. Projects are isolated from each other, each in its
own Kubernetes namespace, so WAF/Firewall (3) and logs/metrics (4) can be scoped per Project.

## Model
- `projects.namespace` (TEXT, unique): `zp-<id8>-<slug>` for new projects. Existing projects
  keep `zenith-apps` (nothing is moved or restarted).
- Resources (apps, databases, buckets, domains) already belong to a project via `project_id`;
  no new tables.
- `NamespaceFor(project)` replaces every hard-coded `zenith-apps` (deploy, env vars, compose,
  `K8sDBBackend`, `K8sLogSource`, reconciler, gateway).

## Kubernetes
- Creating a project creates its namespace (labels `zenith.dev/project`), a default-deny
  ingress NetworkPolicy (allow ingress controller + same namespace), optional ResourceQuota.
- Deleting a project deletes the namespace and everything in it (typed-name confirmation).
- Reconciler: recreates a missing project namespace first, then DBs and apps.

## UI (Compute)
- Project switcher stays the single scope; Apps/Databases/Storage/Domains show the current
  project's resources.
- App detail shows its **containers** (pods: status, restarts, image).
- Remove Queues from sidebar and delete the page. Marketplace stays hidden.

## Out of scope
Per-project WAF/Firewall (3), per-project log/metric search (4), moving resources between
projects, a level above projects.

## Testing
Unit: namespace derivation, migration backfill. Reconciler with two projects. E2E (k3d): two
projects isolated (app in A cannot reach B), delete project removes its namespace, legacy
project in `zenith-apps` keeps working.

## Decisions
Namespace `zp-<id8>-<slug>`; no quota initially; existing projects stay in `zenith-apps`.
