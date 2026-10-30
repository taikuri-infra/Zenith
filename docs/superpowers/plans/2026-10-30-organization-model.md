# Project isolation Implementation Plan

**Goal:** each Project gets its own namespace. **Spec:** docs/superpowers/specs/2026-10-30-organization-model-design.md
Native execution, one PR per slice, CI green before merge.

1. **Data** — migration: `projects.namespace` (backfill `zenith-apps`); `Project.Namespace`; derive for new projects.
2. **Kubernetes** — create/delete namespace + NetworkPolicy on project create/delete; replace hard-coded `zenith-apps` everywhere with the project's namespace; chart RBAC for namespaces; reconciler.
3. **UI** — containers view on App detail; remove Queues.
4. **CLI + E2E** — backup/restore across project namespaces; two-project isolation E2E.
