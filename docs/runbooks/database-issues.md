# Runbook: Database Issues

**Alerts:** `ZenithDatabaseHighConnections`, `ZenithDatabaseReplicationLag`
**Severity:** Warning/Critical

## Connection Pool Exhaustion

### Diagnosis
```bash
# 1. Check CNPG cluster status
kubectl get clusters -n zenith-platform

# 2. Check active connections
kubectl exec -n zenith-platform <CNPG_PRIMARY_POD> -- psql -U postgres -c "SELECT datname, state, count(*) FROM pg_stat_activity GROUP BY datname, state ORDER BY count DESC;"

# 3. Check max_connections setting
kubectl exec -n zenith-platform <CNPG_PRIMARY_POD> -- psql -U postgres -c "SHOW max_connections;"

# 4. Check for long-running queries
kubectl exec -n zenith-platform <CNPG_PRIMARY_POD> -- psql -U postgres -c "SELECT pid, now()-query_start AS duration, state, query FROM pg_stat_activity WHERE state != 'idle' ORDER BY duration DESC LIMIT 10;"
```

### Fix
```bash
# Kill idle connections older than 10 minutes
kubectl exec -n zenith-platform <CNPG_PRIMARY_POD> -- psql -U postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle' AND query_start < now() - interval '10 minutes';"

# Restart API pods to reset connection pool
kubectl rollout restart deployment zenith-api -n zenith-platform
```

## Replication Lag

### Diagnosis
```bash
# 1. Check replication status
kubectl exec -n zenith-platform <CNPG_PRIMARY_POD> -- psql -U postgres -c "SELECT client_addr, state, sent_lsn, write_lsn, flush_lsn, replay_lsn FROM pg_stat_replication;"

# 2. Check CNPG cluster conditions
kubectl get cluster <CLUSTER_NAME> -n zenith-platform -o jsonpath='{.status.conditions}' | jq .

# 3. Check replica pod logs
kubectl logs <CNPG_REPLICA_POD> -n zenith-platform --tail=50
```

### Fix
- If replica is behind: check disk I/O, network between nodes
- If WAL files accumulating: check `wal_keep_size` setting
- Nuclear option: delete replica pod, let CNPG recreate from backup

## OOM on Database Pod

```bash
# Check if OOM killed
kubectl describe pod <CNPG_POD> -n zenith-platform | grep -A2 "Last State"

# Fix: increase shared_buffers / work_mem limits in CNPG cluster spec
kubectl edit cluster <CLUSTER_NAME> -n zenith-platform
```

## Full Backup Verification
```bash
# List CNPG backups
kubectl get backups -n zenith-platform

# Verify latest backup is recent (< 24h)
kubectl get backups -n zenith-platform -o jsonpath='{.items[-1].status.startedAt}'
```
