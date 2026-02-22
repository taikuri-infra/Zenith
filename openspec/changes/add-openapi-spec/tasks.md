## 1. OpenAPI Specification

- [ ] 1.1 Create `services/api/openapi.yaml` with OpenAPI 3.0 header, info, servers
- [ ] 1.2 Define auth security schemes (bearerAuth JWT, apiKeyAuth X-API-Key)
- [ ] 1.3 Document public endpoints: /health, /ready, /api/v1/version, auth (login, register, refresh)
- [ ] 1.4 Document webhook endpoint: /api/v1/webhooks/github (HMAC-SHA256)
- [ ] 1.5 Document protected endpoints: projects CRUD, legacy apps CRUD, V2 apps CRUD, databases, storage
- [ ] 1.6 Document deploy engine endpoints: deployments, rollback, env vars, secrets, releases, logs SSE
- [ ] 1.7 Document admin endpoints: dashboard, clusters, tenants, modules, audit, updates, infrastructure, state, settings
- [ ] 1.8 Document Phase 6.5 endpoints: MFA, SSO, sessions, API keys, webhooks, SCIM, roles, IP whitelist, compliance, branding, preview, domain, audit export
- [ ] 1.9 Define all request/response schemas from existing DTOs and entities
- [ ] 1.10 Add Swagger UI route at `/api/v1/docs` (embed spec + serve UI)

## 2. Validation
- [ ] 2.1 Validate spec with `swagger-cli validate`
- [ ] 2.2 Test Swagger UI renders correctly
- [ ] 2.3 Verify all existing endpoints are covered
