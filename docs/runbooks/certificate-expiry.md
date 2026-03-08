# Runbook: Certificate Expiry

**Alert:** cert-manager Certificate not ready
**Severity:** Critical (if < 7 days to expiry)

## Diagnosis

```bash
# 1. List all certificates and their status
kubectl get certificates --all-namespaces

# 2. Check certificate details (expiry, ready status)
kubectl describe certificate <CERT_NAME> -n <NAMESPACE>

# 3. Check cert-manager logs for renewal errors
kubectl logs -n cert-manager deployment/cert-manager --tail=100 | grep -i error

# 4. Check CertificateRequest status
kubectl get certificaterequests --all-namespaces

# 5. Check ACME challenges
kubectl get challenges --all-namespaces
```

## Common Issues

### HTTP-01 Challenge Failing
```bash
# Check challenge pod/ingress exists
kubectl get challenges -A
kubectl describe challenge <CHALLENGE_NAME> -n <NAMESPACE>

# Common cause: Cloudflare proxy blocking challenge
# Fix: Ensure DNS record is DNS-only (not proxied) during renewal
```

### ClusterIssuer Not Ready
```bash
kubectl get clusterissuer letsencrypt-prod -o yaml
# Check: account registered, ACME server reachable
```

### Rate Limit Hit
- Let's Encrypt rate limits: 50 certs per domain per week
- Wait and retry, or use staging issuer for testing

## Manual Renewal
```bash
# Force certificate renewal by deleting the secret
kubectl delete secret <TLS_SECRET_NAME> -n <NAMESPACE>
# cert-manager will automatically request a new certificate

# Verify renewal
kubectl get certificate <CERT_NAME> -n <NAMESPACE> -w
```

## Wildcard Certificates
```bash
# Wildcard certs require DNS-01 challenge
# Check Cloudflare API token is valid
kubectl get secret cloudflare-api-token -n cert-manager -o jsonpath='{.data.api-token}' | base64 -d
```
