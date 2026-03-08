# Runbook: Disaster Recovery

**Severity:** Critical

## Prerequisites
- Access to Hetzner Cloud console
- Access to S3 backup bucket (Hetzner Object Storage)
- kubectl configured for the target cluster
- CNPG backup credentials

## Full Cluster Recovery

### 1. Provision New Cluster
```bash
cd infra/terraform/staging
terraform apply -var="environment=staging"
```

### 2. Install Core Platform
```bash
cd infra/terraform/staging-k8s
terraform apply
```

### 3. Restore CNPG Database
```bash
# 1. Check available backups
kubectl get backups -n zenith-platform

# 2. Create a recovery cluster from the latest backup
cat <<EOF | kubectl apply -f -
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: zenith-db-restore
  namespace: zenith-platform
spec:
  instances: 2
  storage:
    size: 20Gi
    storageClass: hcloud-volumes
  bootstrap:
    recovery:
      source: zenith-db
  externalClusters:
    - name: zenith-db
      barmanObjectStore:
        destinationPath: "s3://zenith-backups/cnpg/"
        endpointURL: "https://fsn1.your-objectstorage.com"
        s3Credentials:
          accessKeyId:
            name: cnpg-s3-creds
            key: ACCESS_KEY_ID
          secretAccessKey:
            name: cnpg-s3-creds
            key: SECRET_ACCESS_KEY
EOF

# 3. Wait for recovery to complete
kubectl get cluster zenith-db-restore -n zenith-platform -w

# 4. Point API to recovered database
kubectl set env deployment/zenith-api -n zenith-platform \
  DATABASE_URL="postgres://zenith:PASSWORD@zenith-db-restore-rw:5432/zenith?sslmode=require"
```

### 4. Restore Velero Resources
```bash
# List available Velero backups
velero backup get

# Restore specific namespaces
velero restore create --from-backup <BACKUP_NAME> \
  --include-namespaces zenith-apps,zenith-builds

# Verify restore
velero restore describe <RESTORE_NAME>
```

### 5. Verify Recovery
```bash
# Check all pods are running
kubectl get pods --all-namespaces | grep -v Running | grep -v Completed

# Test API health
curl https://api.stage.freezenith.com/health

# Test a sample app deployment
curl https://<APP>.apps.stage.freezenith.com/

# Verify database data
kubectl exec -n zenith-platform <CNPG_POD> -- psql -U zenith -c "SELECT count(*) FROM users;"
```

### 6. DNS Cutover
If recovering to a new IP:
```bash
# Update Cloudflare DNS records
# A record: api.stage.freezenith.com → <NEW_IP>
# A record: *.apps.stage.freezenith.com → <NEW_IP>
```

## Point-in-Time Recovery (PITR)
```bash
# Recover to a specific timestamp
cat <<EOF | kubectl apply -f -
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: zenith-db-pitr
  namespace: zenith-platform
spec:
  instances: 1
  storage:
    size: 20Gi
  bootstrap:
    recovery:
      source: zenith-db
      recoveryTarget:
        targetTime: "2026-03-08T12:00:00Z"
  externalClusters:
    - name: zenith-db
      barmanObjectStore:
        destinationPath: "s3://zenith-backups/cnpg/"
        endpointURL: "https://fsn1.your-objectstorage.com"
        s3Credentials:
          accessKeyId:
            name: cnpg-s3-creds
            key: ACCESS_KEY_ID
          secretAccessKey:
            name: cnpg-s3-creds
            key: SECRET_ACCESS_KEY
EOF
```

## Recovery Time Objectives
- **RTO (Recovery Time Objective):** < 30 minutes for database, < 1 hour for full platform
- **RPO (Recovery Point Objective):** < 5 minutes (continuous WAL archival)
- **DR Test Frequency:** Every 10 days (automated smoke test)
