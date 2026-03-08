# Runbook: Build Failures

**Alert:** `ZenithBuildFailureSpike`, `ZenithBuildQueueBacklog`
**Severity:** Warning

## Diagnosis

```bash
# 1. List recent builds
kubectl get jobs -n zenith-builds --sort-by='.metadata.creationTimestamp' | tail -20

# 2. Check failed builds
kubectl get jobs -n zenith-builds --field-selector=status.failed=1

# 3. Check specific build logs
kubectl logs job/<JOB_NAME> -n zenith-builds

# 4. Check build pod events
kubectl describe job <JOB_NAME> -n zenith-builds
```

## Common Causes

### Registry Push Failure
- **Symptom:** "unauthorized" or "denied" in Kaniko logs
- **Fix:** Verify registry auth secret
```bash
kubectl get secret kaniko-registry-auth -n zenith-builds -o jsonpath='{.data.\.dockerconfigjson}' | base64 -d | jq .
```

### Out of Disk
- **Symptom:** "no space left on device" in build logs
- **Fix:** Clean up old build artifacts, increase PV size
```bash
kubectl get pv | grep zenith-builds
```

### Image Pull Failure in Build
- **Symptom:** "pulling image" timeout
- **Fix:** Check internet access from build pods, DNS resolution
```bash
kubectl run test-dns --rm -it --image=busybox -n zenith-builds -- nslookup docker.io
```

### Build Queue Backlog
```bash
# Check how many builds are running vs pending
kubectl get jobs -n zenith-builds -o custom-columns='NAME:.metadata.name,ACTIVE:.status.active,SUCCEEDED:.status.succeeded,FAILED:.status.failed'

# Check if build concurrency limit is hit
# (configured in API: MAX_CONCURRENT_DEPLOYS env var)
```

## Cleanup

```bash
# Delete completed jobs older than 1 hour
kubectl delete jobs -n zenith-builds --field-selector=status.successful=1 --field-selector='metadata.creationTimestamp<1h'

# Delete all failed jobs (after investigation)
kubectl delete jobs -n zenith-builds --field-selector=status.failed=1
```
