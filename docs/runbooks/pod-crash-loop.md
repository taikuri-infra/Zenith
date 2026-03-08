# Runbook: Pod Crash Loop

**Alert:** `ZenithAppCrashLooping`
**Severity:** Warning

## Symptoms
- Pod restarts > 3 in 15 minutes
- App returning 502/503 errors
- KEDA cold-start page showing instead of app

## Diagnosis

```bash
# 1. Identify the crashing pod
kubectl get pods -n zenith-apps -l zenith.dev/app=<APP_NAME> --sort-by='.status.containerStatuses[0].restartCount'

# 2. Check pod events
kubectl describe pod <POD_NAME> -n zenith-apps

# 3. Check container logs (current)
kubectl logs <POD_NAME> -n zenith-apps --tail=100

# 4. Check container logs (previous crash)
kubectl logs <POD_NAME> -n zenith-apps --previous --tail=100

# 5. Check resource limits vs actual usage
kubectl top pod <POD_NAME> -n zenith-apps
```

## Common Causes & Fixes

### OOM Killed
- **Symptom:** `reason: OOMKilled` in describe output
- **Fix:** Increase memory limit in app configuration or optimize app memory usage
```bash
kubectl get pod <POD_NAME> -n zenith-apps -o jsonpath='{.spec.containers[0].resources}'
```

### Liveness Probe Failure
- **Symptom:** `Liveness probe failed` in events
- **Fix:** Check if app health endpoint is responding, increase probe timeout
```bash
kubectl exec <POD_NAME> -n zenith-apps -- wget -qO- http://localhost:<PORT>/health
```

### Image Pull Error
- **Symptom:** `ErrImagePull` or `ImagePullBackOff`
- **Fix:** Verify image exists in Harbor, check registry auth secrets
```bash
kubectl get secret app-registry-auth -n zenith-apps -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d
```

### Application Error
- **Symptom:** CrashLoopBackOff with exit code 1
- **Fix:** Check application logs for startup errors (missing env vars, failed DB connection)

## Escalation
If the issue persists after 30 minutes, check the CNPG database and APISIX gateway status.
