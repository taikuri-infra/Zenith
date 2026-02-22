# K8s Client (Backend Module Doc)

## 1. Purpose

Provides a unified Kubernetes interface for the Zenith API. Supports two modes:
- **Memory** (`K8S_MODE=memory`): In-memory fake client for development/testing
- **Real** (`K8S_MODE=real`): Production client using `client-go` v0.35.1

## 2. Entities

| Struct | File | Description |
|--------|------|-------------|
| `CRDObject` | `client.go` | Generic K8s CRD (apiVersion, kind, metadata, spec, status) |
| `ObjectMeta` | `client.go` | Standard name/namespace/labels/annotations |
| `JobObject` | `client.go` | Kubernetes batch/v1 Job with status fields |

## 3. Services (Use Cases)

N/A — this is an infrastructure adapter, not a domain service.

## 4. Ports

```go
// k8s.Client interface (client.go)
type Client interface {
    CreateCRD(ctx, obj) error
    GetCRD(ctx, kind, namespace, name) (*CRDObject, error)
    UpdateCRD(ctx, obj) error
    DeleteCRD(ctx, kind, namespace, name) error
    ListCRDs(ctx, kind, namespace) ([]*CRDObject, error)
    CreateJob(ctx, job) error
    GetJob(ctx, namespace, name) (*JobObject, error)
    DeleteJob(ctx, namespace, name) error
    GetPodLogs(ctx, namespace, podSelector, logCh) error
}
```

## 5. Adapters

| Adapter | File | Description |
|---------|------|-------------|
| `MemoryClient` | `client.go` | In-memory map-based, fake execution. Jobs succeed instantly. Logs emit 9 fake Kaniko lines. |
| `RealClient` | `real_client.go` | Uses `client-go` dynamic client for CRDs, typed client for Jobs/Pods. Auto-detects in-cluster vs kubeconfig. |

## 6. API Endpoints

The K8s client is not directly exposed via API. It's used internally by:
- `handlers/project.go` — CRUD projects (CRDs)
- `handlers/app.go` — CRUD apps (CRDs)
- `handlers/database.go` — CRUD databases (CRDs)
- `handlers/storage.go` — CRUD storage buckets (CRDs)
- `handlers/backstage.go` — Catalog aggregation (reads all CRDs)
- `deploy/builder.go` — Creates Kaniko build Jobs
- `deploy/deployer.go` — Creates Deployment/Service/IngressRoute CRDs

## 7. Validation Rules

- `NewRealClient()` fails fast if neither in-cluster config nor kubeconfig is available
- CRD kind is pluralized via `strings.ToLower(kind) + "s"` (covers Zenith CRDs)
- Job deletion uses `DeletePropagationForeground` to clean up pods

## 8. Security Model

- In-cluster: uses ServiceAccount token mounted by K8s (RBAC required)
- Local: reads `~/.kube/config` or `KUBECONFIG` env var
- No credentials stored in code or config files

## 9. Testing Strategy

```bash
GO111MODULE=on go test ./internal/k8s/...
```

- `MemoryClient` tested with full CRUD + Job lifecycle
- `RealClient` requires a live cluster (integration test, not in CI)

## 10. Future Improvements

- Add retry/backoff for transient K8s API errors
- Watch support for real-time CRD status updates
- Namespace-scoped RBAC validation on startup
