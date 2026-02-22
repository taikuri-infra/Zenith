## 1. KEDA Infrastructure

- [ ] 1.1 Install KEDA operator via Helm on workload cluster
- [ ] 1.2 Install KEDA HTTP Add-on for HTTP-based scaling
- [ ] 1.3 Configure interceptor proxy for HTTP request routing

## 2. Backend Integration

- [ ] 2.1 Add `sleeping` status to app status enum
- [ ] 2.2 Generate `ScaledObject` CRD for free-tier apps (minReplicas=0, maxReplicas=1, cooldown=900s)
- [ ] 2.3 Generate `HTTPScaledObject` for HTTP trigger routing
- [ ] 2.4 Skip KEDA resources for Pro/Team/Enterprise plans
- [ ] 2.5 App status endpoint reflects sleeping/waking state
- [ ] 2.6 Wake latency monitoring (target: <5s)

## 3. Frontend

- [ ] 3.1 Deploy page: sleep indicator (moon icon) on sleeping apps
- [ ] 3.2 App detail: show "Sleeping — wakes on first request" status
- [ ] 3.3 Dashboard: sleeping vs running app count in stats

## 4. Testing

- [ ] 4.1 Unit tests for ScaledObject generation
- [ ] 4.2 Integration test: app scales to zero after inactivity
- [ ] 4.3 Integration test: app wakes on HTTP request within 5s
- [ ] 4.4 Verify Pro/Team apps never sleep
