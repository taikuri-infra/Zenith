# Backstage Integration (Backend Feature Doc)

## 1. Overview

Exposes Zenith CRDs as Backstage catalog entities via REST API. Converts Projects, Apps, Databases, StorageBuckets, and Domains to Backstage-compatible `Component`, `System`, `Resource`, and `API` entities.

## 2. UI/UX Flow

N/A — API-only, consumed by Backstage's catalog ingestion.

## 3. Data Flow

```
Zenith CRDs (K8s) → BackstageHandler → Backstage Entity JSON → Backstage Catalog
```

## 4. Components

| Component | File | Description |
|-----------|------|-------------|
| `BackstageHandler` | `handlers/backstage.go` | HTTP handler with 2 endpoints |
| `BackstageEntity` | `handlers/backstage.go` | Backstage catalog entity struct |
| `BackstageMetadata` | `handlers/backstage.go` | Entity metadata (name, namespace, labels, tags) |

## 5. Services/API

| Method | Route | Description |
|--------|-------|-------------|
| `GET` | `/api/v1/backstage/catalog` | All Zenith resources as Backstage entities |
| `GET` | `/api/v1/backstage/catalog/:kind` | Filter by kind (Component, Resource, API, System) |

Both routes require JWT auth (protected group).

## 6. Hooks

N/A — backend only.

## 7. State Logic

Stateless — reads CRDs from K8s on each request.

## 8. Edge Cases

- Empty cluster → returns `{"items": [], "total": 0}`
- Unknown kind filter → returns empty filtered list
- K8s client error → returns 500 with error message

## 9. Security Considerations

- Requires JWT authentication
- Uses same K8s client permissions as the API server
- No write operations (read-only)

## 10. Testing Strategy

```bash
GO111MODULE=on go test ./internal/handlers/ -run TestBackstage
```

Existing tests in `backstage_test.go` cover catalog listing and kind filtering.

## 11. Future Improvements

- Add pagination for large clusters
- Cache catalog with TTL to reduce K8s API calls
- Support `/backstage/catalog.yaml` for file-based catalog ingestion
