# Change: Add OpenAPI 3.0 Specification

## Why
The API has 50+ endpoints across public, protected, admin, and webhook routes but no machine-readable documentation. An OpenAPI spec enables auto-generated client SDKs, Swagger UI, API testing, and onboarding for new developers.

## What Changes
- Generate/write OpenAPI 3.0 spec covering all existing endpoints
- Add Swagger UI endpoint at `/api/v1/docs`
- Add request/response schemas matching existing DTOs
- Document auth schemes (JWT Bearer, API Key, Webhook HMAC)

## Impact
- Affected specs: observability (new docs endpoint)
- Affected code: `services/api/` (new static file or generated spec), possibly `main.go` for Swagger UI route
- No breaking changes — purely additive
