# Runbook: Storage Issues

**Severity:** Warning/Critical

## Hetzner Volume Full

```bash
# 1. Check PV usage
kubectl get pv --sort-by='.status.capacity.storage'

# 2. Check PVC usage in specific namespace
kubectl exec -n <NAMESPACE> <POD> -- df -h

# 3. Identify large files
kubectl exec -n <NAMESPACE> <POD> -- du -sh /* 2>/dev/null | sort -rh | head -10
```

### Fix: Expand Volume
```bash
# Edit PVC to request more storage (Hetzner volumes support online resize)
kubectl patch pvc <PVC_NAME> -n <NAMESPACE> -p '{"spec":{"resources":{"requests":{"storage":"50Gi"}}}}'
```

## S3 (Hetzner Object Storage) Access Denied

```bash
# 1. Verify S3 credentials
kubectl get secret -n zenith-platform -o jsonpath='{.data}' | base64 -d

# 2. Test S3 access
kubectl run s3-test --rm -it --image=amazon/aws-cli -n zenith-platform -- \
  s3 ls --endpoint-url=https://fsn1.your-objectstorage.com s3://zenith-platform-storage/

# 3. Check bucket policy
# Hetzner Object Storage doesn't support bucket policies — check IAM credentials
```

## Prometheus/Loki Storage Full

```bash
# Check Prometheus PV usage
kubectl exec -n monitoring prometheus-kube-prometheus-stack-prometheus-0 -- df -h /prometheus

# Check Loki PV usage
kubectl exec -n monitoring loki-0 -- df -h /var/loki

# Fix: reduce retention or expand volume
# Prometheus: spec.retention in kube-prometheus-stack values
# Loki: compactor.retention_period in loki config
```

## Harbor Registry Storage
```bash
# Check Harbor storage usage
kubectl exec -n zenith-platform <HARBOR_REGISTRY_POD> -- df -h /storage

# Garbage collect unreferenced blobs
kubectl exec -n zenith-platform <HARBOR_REGISTRY_POD> -- registry garbage-collect /etc/docker/registry/config.yml
```
