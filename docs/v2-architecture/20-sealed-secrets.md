# 20 — Sealed Secrets

> **Purpose:** Understand how secrets are safely stored in Git for GitOps workflows using Sealed Secrets.
> **Audience:** Any developer who needs to manage secrets, add new ones, or understand the encryption flow.
> **Last Updated:** 2026-03-03
> **Related:** [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (installation steps), [15-argocd-gitops.md](./15-argocd-gitops.md) (ArgoCD syncs SealedSecrets from Git), [19-velero-backup.md](./19-velero-backup.md) (Velero backs up encryption keys)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose It](#2-why-we-chose-it)
3. [Architecture Diagram](#3-architecture-diagram)
4. [How Sealed Secrets Work](#4-how-sealed-secrets-work)
5. [Creating a Sealed Secret](#5-creating-a-sealed-secret)
6. [Key Management](#6-key-management)
7. [Configuration Reference](#7-configuration-reference)
8. [Troubleshooting](#8-troubleshooting)
9. [Upgrade Path](#9-upgrade-path)

---

## 1. Overview

**The Problem:** GitOps (ArgoCD) requires everything to be in Git. But Kubernetes Secrets contain sensitive data (passwords, tokens, API keys). You can't commit plain Secrets to Git.

**The Solution:** Sealed Secrets encrypts secrets client-side so they can be safely committed to Git. Only the Sealed Secrets controller running in the cluster can decrypt them.

```
The workflow:

  1. You have a Secret (plain text)
  2. kubeseal encrypts it with the cluster's public key → SealedSecret
  3. Commit SealedSecret to Git (safe — it's encrypted!)
  4. ArgoCD syncs SealedSecret to cluster
  5. Sealed Secrets controller decrypts it → creates regular Secret
  6. Pods read the Secret as normal
```

---

## 2. Why We Chose It

| Feature | Sealed Secrets | External Secrets Operator | SOPS + age | Vault |
|---------|---------------|--------------------------|-----------|-------|
| Complexity | Very low | Medium | Low | High |
| External dependency | None (self-contained) | Needs external store | None | Vault server |
| GitOps native | Yes (CRD-based) | Yes (CRD-based) | File-based | External |
| Key management | Auto (controller manages) | External | Manual | Vault |
| Namespace scoping | Yes | Yes | File-based | Policy-based |
| Cost | Free | Free | Free | $$$ |

**Decision:** Sealed Secrets is the simplest option for our use case. We don't need a full secrets management platform (Vault) — we just need to safely store a few dozen secrets in Git.

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SEALED SECRETS ARCHITECTURE                               │
│                                                                             │
│  DEVELOPER LAPTOP                          KUBERNETES CLUSTER               │
│  ┌─────────────────────────────┐          ┌──────────────────────────────┐ │
│  │                             │          │                              │ │
│  │  1. Create Secret YAML      │          │  sealed-secrets namespace    │ │
│  │  ┌───────────────────────┐  │          │  ┌────────────────────────┐ │ │
│  │  │ apiVersion: v1        │  │          │  │ Sealed Secrets         │ │ │
│  │  │ kind: Secret          │  │          │  │ Controller             │ │ │
│  │  │ data:                 │  │          │  │                        │ │ │
│  │  │   password: cGFzcw==  │  │          │  │ - Holds private key    │ │ │
│  │  └───────────┬───────────┘  │          │  │ - Decrypts Sealed      │ │ │
│  │              │               │          │  │   Secrets → Secrets    │ │ │
│  │              ▼               │          │  │ - Serves public key    │ │ │
│  │  2. kubeseal encrypt         │          │  │   via API endpoint     │ │ │
│  │  ┌───────────────────────┐  │  fetch   │  │                        │ │ │
│  │  │ $ kubeseal             │──┼──pubkey──┼─▶│ Resources: 25m-100m   │ │ │
│  │  │   --controller-name    │  │          │  │ CPU, 64Mi-128Mi RAM   │ │ │
│  │  │   sealed-secrets       │  │          │  └────────────┬───────────┘ │ │
│  │  │   --controller-ns      │  │          │               │             │ │
│  │  │   sealed-secrets       │  │          │               │ decrypt     │ │
│  │  │   < secret.yaml        │  │          │               ▼             │ │
│  │  │   > sealed.yaml        │  │          │  Target namespace          │ │
│  │  └───────────┬───────────┘  │          │  ┌────────────────────────┐ │ │
│  │              │               │          │  │ SealedSecret (CRD)    │ │ │
│  │              ▼               │          │  │ (encrypted, in Git)    │ │ │
│  │  3. Commit to Git            │  ArgoCD  │  │         │              │ │ │
│  │  ┌───────────────────────┐  │  sync    │  │         ▼              │ │ │
│  │  │ sealed-secret.yaml    │──┼─────────▶│  │ Secret (plain text)    │ │ │
│  │  │ (safe to commit!)     │  │          │  │ (created by controller)│ │ │
│  │  └───────────────────────┘  │          │  │ (pods can read this)   │ │ │
│  └─────────────────────────────┘          │  └────────────────────────┘ │ │
│                                            └──────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. How Sealed Secrets Work

```
┌─────────────────────────────────────────────────────────────────────────┐
│              ENCRYPTION / DECRYPTION FLOW                                │
│                                                                          │
│  ENCRYPTION (developer laptop → kubeseal CLI):                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                                                                    │ │
│  │  Input: Secret YAML (plain text)                                   │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ apiVersion: v1                                               │  │ │
│  │  │ kind: Secret                                                 │  │ │
│  │  │ metadata:                                                    │  │ │
│  │  │   name: my-secret                                            │  │ │
│  │  │   namespace: zenith-staging                                  │  │ │
│  │  │ data:                                                        │  │ │
│  │  │   DB_PASSWORD: c3VwZXJzZWNyZXQ=   (base64 of "supersecret") │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  │                         │                                          │ │
│  │                         ▼                                          │ │
│  │  kubeseal:                                                         │ │
│  │    1. Fetches public key from controller                          │ │
│  │    2. Encrypts each data field with RSA-OAEP                     │ │
│  │    3. Embeds namespace + name in encryption                       │ │
│  │       (so secret can ONLY be decrypted for that namespace/name)   │ │
│  │                         │                                          │ │
│  │                         ▼                                          │ │
│  │  Output: SealedSecret YAML (encrypted, safe for Git)              │ │
│  │  ┌──────────────────────────────────────────────────────────────┐  │ │
│  │  │ apiVersion: bitnami.com/v1alpha1                             │  │ │
│  │  │ kind: SealedSecret                                           │  │ │
│  │  │ metadata:                                                    │  │ │
│  │  │   name: my-secret                                            │  │ │
│  │  │   namespace: zenith-staging                                  │  │ │
│  │  │ spec:                                                        │  │ │
│  │  │   encryptedData:                                             │  │ │
│  │  │     DB_PASSWORD: AgBy3i4OJD...long encrypted blob...         │  │ │
│  │  └──────────────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  DECRYPTION (controller in cluster):                                     │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                                                                    │ │
│  │  1. ArgoCD creates SealedSecret CRD in cluster                    │ │
│  │  2. Controller watches for SealedSecret CRDs                       │ │
│  │  3. Controller decrypts using private key (stored in cluster only) │ │
│  │  4. Controller creates corresponding Secret:                       │ │
│  │     ┌──────────────────────────────────────────────────────────┐   │ │
│  │     │ apiVersion: v1                                           │   │ │
│  │     │ kind: Secret                                             │   │ │
│  │     │ metadata:                                                │   │ │
│  │     │   name: my-secret                                        │   │ │
│  │     │   namespace: zenith-staging                              │   │ │
│  │     │   ownerReferences: [SealedSecret/my-secret]              │   │ │
│  │     │ data:                                                    │   │ │
│  │     │   DB_PASSWORD: c3VwZXJzZWNyZXQ=  (decrypted!)           │   │ │
│  │     └──────────────────────────────────────────────────────────┘   │ │
│  │  5. Pods mount the Secret as normal (env var or volume)           │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Creating a Sealed Secret

### Step-by-step

```bash
# 1. Create a plain Secret YAML (don't apply it!)
kubectl create secret generic my-db-creds \
  --from-literal=username=admin \
  --from-literal=password=supersecret \
  --namespace=zenith-staging \
  --dry-run=client -o yaml > secret.yaml

# 2. Seal it using the cluster's public key
kubeseal \
  --controller-name=sealed-secrets \
  --controller-namespace=sealed-secrets \
  --format=yaml \
  < secret.yaml > sealed-secret.yaml

# 3. Delete the plain Secret (NEVER commit it)
rm secret.yaml

# 4. Commit the SealedSecret to Git
git add sealed-secret.yaml
git commit -m "Add sealed secret for DB credentials"
git push

# 5. ArgoCD syncs → controller decrypts → Secret created!
```

### Verifying

```bash
# Check SealedSecret exists
kubectl get sealedsecret -n zenith-staging

# Check corresponding Secret was created
kubectl get secret my-db-creds -n zenith-staging

# Verify the secret data
kubectl get secret my-db-creds -n zenith-staging -o jsonpath='{.data.password}' | base64 -d
# Output: supersecret
```

---

## 6. Key Management

```
┌─────────────────────────────────────────────────────────────────────────┐
│              KEY MANAGEMENT                                              │
│                                                                          │
│  KEY PAIR:                                                               │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ Private key: stored ONLY in the cluster                            │ │
│  │   Secret: sealed-secrets-key in sealed-secrets namespace           │ │
│  │   NEVER exported to Git or anywhere outside the cluster            │ │
│  │                                                                    │ │
│  │ Public key: available to anyone (safe to share)                    │ │
│  │   Fetched by kubeseal CLI from controller endpoint                │ │
│  │   Can be exported: kubeseal --fetch-cert > pub-cert.pem            │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  KEY ROTATION:                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ Controller generates new key pair every 30 days (default)          │ │
│  │ Old keys are kept (so old SealedSecrets can still be decrypted)    │ │
│  │                                                                    │ │
│  │ Timeline:                                                          │ │
│  │   Day 0:   Key A (active) — new SealedSecrets encrypted with A    │ │
│  │   Day 30:  Key B (active) — new SealedSecrets encrypted with B    │ │
│  │            Key A (retained) — old SealedSecrets still work         │ │
│  │   Day 60:  Key C (active)                                         │ │
│  │            Key A, B (retained)                                     │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  DISASTER RECOVERY:                                                      │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ If the private key is lost (cluster destroyed), you CANNOT         │ │
│  │ decrypt existing SealedSecrets.                                    │ │
│  │                                                                    │ │
│  │ Solution: Velero backs up the sealed-secrets-key Secret            │ │
│  │ On restore: restore Velero backup → keys are restored →            │ │
│  │             SealedSecrets can be decrypted again                   │ │
│  │                                                                    │ │
│  │ BACKUP THE KEY MANUALLY (belt + suspenders):                       │ │
│  │ $ kubectl get secret -n sealed-secrets -l sealedsecrets.bitnami.   │ │
│  │   com/sealed-secrets-key -o yaml > sealed-secrets-backup.yaml      │ │
│  │ Store this file in a secure location (NOT Git!)                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Configuration Reference

### Terraform File

**File:** `infra/terraform/modules/k8s-platform/sealed_secrets.tf`

| Setting | Value |
|---------|-------|
| Namespace | sealed-secrets |
| Chart | bitnami-labs/sealed-secrets |
| Resources requests | 25m CPU, 64Mi RAM |
| Resources limits | 100m CPU, 128Mi RAM |
| Depends on | cert-manager |

---

## 8. Troubleshooting

### SealedSecret not creating Secret

```bash
# 1. Check SealedSecret status
kubectl get sealedsecret -n <namespace>
kubectl describe sealedsecret <name> -n <namespace>
# Look for events/conditions

# 2. Check controller logs
kubectl logs -n sealed-secrets deploy/sealed-secrets --tail=50

# 3. Common issues:
#    - Wrong namespace: SealedSecret was sealed for a different namespace
#    - Wrong name: SealedSecret was sealed for a different name
#    - Key mismatch: cluster was rebuilt without restoring the key
```

### "error: unable to decrypt" in controller logs

```bash
# The private key doesn't match the encryption
# This happens when:
#   1. Cluster was rebuilt (new key pair)
#   2. SealedSecrets were sealed with the old cluster's public key

# Fix: re-seal all secrets with the new cluster's public key
kubeseal --fetch-cert > new-pub-cert.pem
kubeseal --cert new-pub-cert.pem < secret.yaml > sealed-secret.yaml
# Commit and push the re-sealed secrets
```

### kubeseal can't connect to controller

```bash
# 1. Check controller is running
kubectl get pods -n sealed-secrets

# 2. If using kubeseal from local machine, need port-forward or kubeconfig
kubectl port-forward -n sealed-secrets svc/sealed-secrets 8080:8080

# 3. Or use offline mode with the certificate
kubeseal --fetch-cert > pub-cert.pem
kubeseal --cert pub-cert.pem < secret.yaml > sealed.yaml
```

---

## 9. Upgrade Path

### Upgrading Sealed Secrets

```bash
terraform plan -target=helm_release.sealed_secrets
terraform apply -target=helm_release.sealed_secrets

# Verify: existing SealedSecrets should still work (keys are preserved)
kubectl get sealedsecret -A
kubectl logs -n sealed-secrets deploy/sealed-secrets --tail=20
```

### Backup key before upgrade (safety)

```bash
kubectl get secret -n sealed-secrets \
  -l sealedsecrets.bitnami.com/sealed-secrets-key \
  -o yaml > sealed-secrets-key-backup.yaml
# Store securely (password manager, encrypted S3, etc.)
```
