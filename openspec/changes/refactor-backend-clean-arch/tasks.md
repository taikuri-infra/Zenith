## 1. Phase B — Extract Ports and Adapters

- [ ] 1.1 Create `internal/ports/` — move interfaces from `store/interfaces.go`
- [ ] 1.2 Create `internal/adapters/postgres/` — move `store/postgres_*.go` files
- [ ] 1.3 Create `internal/adapters/memory/` — move `store/memory_*.go` files
- [ ] 1.4 Keep `store/` as re-export wrapper during transition
- [ ] 1.5 Update `main.go` imports to use new paths
- [ ] 1.6 Run all tests — must pass with zero changes

## 2. Phase C — Introduce Services Layer

- [ ] 2.1 Create `internal/services/auth_service.go` — extract business logic from `handlers/auth.go`
- [ ] 2.2 Create `internal/services/customer_service.go` — extract from `handlers/customer.go`
- [ ] 2.3 Create `internal/services/app_service.go` — extract from `handlers/apps_v2.go`
- [ ] 2.4 Create `internal/services/deploy_service.go` — extract from `handlers/deploy.go`
- [ ] 2.5 Create `internal/services/metering_service.go` — extract from `handlers/metering.go`
- [ ] 2.6 Create `internal/services/admin_service.go` — extract from `handlers/admin.go`
- [ ] 2.7 Update handlers to call services instead of stores directly
- [ ] 2.8 Inject services into handlers via constructor
- [ ] 2.9 Run all tests — must pass

## 3. Phase D — Validators and Cleanup

- [ ] 3.1 Create `internal/validators/` — extract inline validation from handlers
- [ ] 3.2 Remove `store/` backward-compat wrappers
- [ ] 3.3 Remove `models/` package (fully replaced by `entities/` + `dto/`)
- [ ] 3.4 Final dependency audit: no layer violations (handlers must not import adapters)
- [ ] 3.5 Run all tests — must pass
