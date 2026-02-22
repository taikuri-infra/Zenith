## ADDED Requirements

### Requirement: OpenAPI Documentation Endpoint
The system SHALL serve an interactive API documentation UI at `/api/v1/docs` powered by Swagger UI, rendering the OpenAPI 3.0 spec for all endpoints.

#### Scenario: Access API docs
- **WHEN** a user navigates to `/api/v1/docs`
- **THEN** an interactive Swagger UI is displayed with all endpoint documentation

### Requirement: OpenAPI Specification File
The system SHALL maintain an OpenAPI 3.0 specification file covering all public, protected, admin, and webhook endpoints with request/response schemas and auth security schemes.

#### Scenario: Valid spec
- **WHEN** the OpenAPI spec is validated
- **THEN** it passes OpenAPI 3.0 validation with zero errors
