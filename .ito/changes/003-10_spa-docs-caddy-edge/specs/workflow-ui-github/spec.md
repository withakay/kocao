## MODIFIED Requirements

### Requirement: Session and Run Workflow Console
The system SHALL provide a web console for creating sessions, starting runs, and viewing run lifecycle state across workflow routes, and SHALL serve this console from the in-cluster Caddy edge using the built SPA assets.

#### Scenario: Cluster endpoint serves real SPA workflow console
- **WHEN** an operator opens the deployed root endpoint
- **THEN** the React workflow console loads (not a placeholder static page) and can call `/api/v1/*` through the same edge endpoint

## ADDED Requirements

### Requirement: Versioned API Docs Endpoints
The edge SHALL expose API documentation endpoints under versioned paths: Scalar at `/api/v1/scalar` and OpenAPI at `/api/v1/openapi.json`.

#### Scenario: User opens versioned Scalar endpoint
- **WHEN** a user navigates to `/api/v1/scalar`
- **THEN** Scalar loads and fetches schema from `/api/v1/openapi.json`

#### Scenario: Legacy docs routes remain compatible
- **WHEN** a user or script requests `/scalar` or `/openapi.json`
- **THEN** the edge redirects to `/api/v1/scalar` or `/api/v1/openapi.json`
