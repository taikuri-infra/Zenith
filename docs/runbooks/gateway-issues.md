# Runbook: Gateway Issues (APISIX)

**Severity:** Warning/Critical

## APISIX Not Routing

```bash
# 1. Check APISIX pod status
kubectl get pods -n zenith-platform -l app.kubernetes.io/name=apisix

# 2. Check APISIX logs
kubectl logs -n zenith-platform deployment/apisix -c apisix --tail=100

# 3. List configured routes
kubectl exec -n zenith-platform deployment/apisix -c apisix -- \
  curl -s http://localhost:9180/apisix/admin/routes -H "X-API-KEY: $(kubectl get secret apisix-admin -n zenith-platform -o jsonpath='{.data.key}' | base64 -d)" | jq '.list | length'

# 4. Check Traefik → APISIX connectivity
kubectl exec -n zenith-platform deployment/apisix -c apisix -- wget -qO- http://localhost:9080/health
```

## Rate Limiting Too Aggressive

```bash
# Check which routes have rate limiting
kubectl exec -n zenith-platform deployment/apisix -c apisix -- \
  curl -s http://localhost:9180/apisix/admin/routes -H "X-API-KEY: <ADMIN_KEY>" | \
  jq '.list[] | select(.value.plugins.limit_req) | {id: .value.id, uri: .value.uri, rate: .value.plugins.limit_req.rate}'

# Temporarily increase rate limit for a route
curl -X PATCH http://localhost:9180/apisix/admin/routes/<ROUTE_ID> \
  -H "X-API-KEY: <ADMIN_KEY>" \
  -d '{"plugins":{"limit-req":{"rate":100,"burst":50}}}'
```

## CORS Issues

```bash
# Check CORS plugin on routes
kubectl exec -n zenith-platform deployment/apisix -c apisix -- \
  curl -s http://localhost:9180/apisix/admin/routes -H "X-API-KEY: <ADMIN_KEY>" | \
  jq '.list[] | select(.value.plugins.cors) | {id: .value.id, uri: .value.uri, origins: .value.plugins.cors.allow_origins}'
```

## SSL/TLS Certificate Issues on Gateway Routes

```bash
# Check SSL certificates loaded in APISIX
kubectl exec -n zenith-platform deployment/apisix -c apisix -- \
  curl -s http://localhost:9180/apisix/admin/ssls -H "X-API-KEY: <ADMIN_KEY>" | jq '.list | length'
```

## Restart APISIX

```bash
kubectl rollout restart deployment apisix -n zenith-platform
kubectl rollout status deployment apisix -n zenith-platform
```
