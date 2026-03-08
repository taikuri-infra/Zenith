# Runbook: API Degradation

**Alerts:** `ZenithAPIDown`, `ZenithAPIHighErrorRate`, `ZenithAPIHighLatency`
**Severity:** Critical (down) / Warning (degraded)

## API Down

```bash
# 1. Check API pod status
kubectl get pods -n zenith-platform -l app=zenith-api

# 2. Check API pod logs
kubectl logs -n zenith-platform deployment/zenith-api --tail=100

# 3. Check if service endpoint exists
kubectl get endpoints zenith-api -n zenith-platform

# 4. Restart API
kubectl rollout restart deployment zenith-api -n zenith-platform
kubectl rollout status deployment zenith-api -n zenith-platform
```

## High Error Rate

```bash
# 1. Check API logs for 5xx errors
kubectl logs -n zenith-platform deployment/zenith-api --tail=200 | grep -E '"status":(5[0-9]{2})'

# 2. Check database connectivity
kubectl exec -n zenith-platform deployment/zenith-api -- wget -qO- http://localhost:8080/health

# 3. Check APISIX gateway logs
kubectl logs -n zenith-platform deployment/apisix -c apisix --tail=100

# 4. Check if rate limiting is too aggressive
kubectl exec -n zenith-platform deployment/apisix -c apisix -- curl -s http://localhost:9180/apisix/admin/routes | jq '.list[].value.plugins.limit_req'
```

## High Latency

```bash
# 1. Check database query performance
kubectl exec -n zenith-platform <CNPG_PRIMARY> -- psql -U postgres -c "SELECT pid, now()-query_start AS duration, query FROM pg_stat_activity WHERE state='active' AND query_start < now()-interval '1 second' ORDER BY duration DESC LIMIT 5;"

# 2. Check API pod CPU/memory
kubectl top pods -n zenith-platform -l app=zenith-api

# 3. Check network latency between API and DB
kubectl exec -n zenith-platform deployment/zenith-api -- time wget -qO- http://<CNPG_SERVICE>:5432 2>&1 || true

# 4. Check if garbage collection is causing pauses (Go)
kubectl logs -n zenith-platform deployment/zenith-api --tail=500 | grep -i "gc\|pause"
```

## Quick Recovery

```bash
# Scale up API replicas temporarily
kubectl scale deployment zenith-api -n zenith-platform --replicas=3

# After investigation, scale back
kubectl scale deployment zenith-api -n zenith-platform --replicas=2
```
