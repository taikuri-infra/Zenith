# Change: Add KEDA Scale-to-Zero (Phase 4)

## Why
The free tier's economics depend on sleep mode: apps scale to zero after 15 minutes of inactivity, reducing per-user cost from ~EUR 12/mo to ~EUR 0.70/mo. This makes the free tier viable with 1,000+ users on shared infrastructure. KEDA + HTTP Add-on enable this transparently.

## What Changes
- Install KEDA + KEDA HTTP Add-on on the workload cluster
- Free-tier apps get `ScaledObject` CRDs with HTTP trigger (0-1 replicas)
- Pro/Team/Enterprise apps remain always-on (no KEDA)
- App status gains `sleeping` state (replicas=0, wakes on HTTP request)
- Wake latency target: 2-5 seconds (cold start)
- Dashboard shows sleep status with wake indicator

## Impact
- Affected specs: app-management (new sleeping status), deploy-engine (ScaledObject creation), web-platform (sleep indicator)
- Affected code: `services/api/internal/deploy/`, K8s resource generation, `apps/web/`
- New dependencies: KEDA operator, KEDA HTTP Add-on
- Plan-gated: only free tier apps sleep
