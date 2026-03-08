# Runbook: High CPU/Memory Usage

**Alert:** `ZenithAppHighMemory`
**Severity:** Warning

## Symptoms
- Memory usage > 90% of limit
- CPU throttling
- Slow response times

## Diagnosis

```bash
# 1. Check node-level resources
kubectl top nodes

# 2. Check pod resource usage across zenith-apps
kubectl top pods -n zenith-apps --sort-by=memory

# 3. Check specific app pods
kubectl top pods -n zenith-apps -l zenith.dev/app=<APP_NAME>

# 4. Check resource limits
kubectl get pods -n zenith-apps -o custom-columns='NAME:.metadata.name,CPU_REQ:.spec.containers[0].resources.requests.cpu,CPU_LIM:.spec.containers[0].resources.limits.cpu,MEM_REQ:.spec.containers[0].resources.requests.memory,MEM_LIM:.spec.containers[0].resources.limits.memory'

# 5. Check HPA/KEDA status
kubectl get hpa -n zenith-apps
kubectl get httpscaledobjects -n zenith-apps
```

## Remediation

### Single Pod High Memory
```bash
# Restart the pod (k8s will recreate)
kubectl delete pod <POD_NAME> -n zenith-apps
```

### Cluster-Wide High Memory
```bash
# Check for memory leak candidates (high RSS, growing over time)
kubectl top pods -n zenith-apps --sort-by=memory | head -10

# Check node allocatable vs actual
kubectl describe nodes | grep -A 5 "Allocated resources"
```

### CPU Throttling
```bash
# Check if CPU is being throttled
kubectl get --raw /apis/metrics.k8s.io/v1beta1/namespaces/zenith-apps/pods | jq '.items[] | {name: .metadata.name, cpu: .containers[0].usage.cpu}'
```

## Prevention
- Set appropriate resource requests/limits per plan tier
- Enable HPA for Pro+ apps
- Monitor trends in Grafana Platform Health dashboard
