## ADDED Requirements

### Requirement: Hostile Private Network Posture
The system SHALL assume an attacker may have access to the private network and SHALL apply defense-in-depth controls.

#### Scenario: Default deployment posture
- **WHEN** the control-plane components are deployed in-cluster
- **THEN** they use least-privilege RBAC, restrictive egress by default, and authenticated control-plane API access

### Requirement: Authenticated Control-Plane API
The system SHALL require a valid bearer token for all `/api/v1/*` endpoints except health checks and OpenAPI.

#### Scenario: Missing token is rejected
- **WHEN** a client calls `GET /api/v1/workspace-sessions` without an `Authorization: Bearer ...` header
- **THEN** the API responds with HTTP 401

### Requirement: Attach Session Safety
The system SHALL harden attach websocket endpoints against abuse (origin checks, message size limits, timeouts).

#### Scenario: Oversized websocket messages are rejected
- **WHEN** a client sends an oversized websocket message to the attach endpoint
- **THEN** the server closes the connection and does not exhaust memory

### Requirement: Auditability
The system SHALL record security-relevant actions (authorization decisions, control changes, attach issuance/usage, egress overrides) as append-only audit events.

#### Scenario: Authorization decision is auditable
- **WHEN** an authenticated client attempts an action requiring scopes
- **THEN** an audit event is recorded with outcome `allowed` or `denied`

### Requirement: Secrets Handling
The system SHALL avoid exposing secrets in URLs and SHALL prefer file-mounted secrets for runtime components.

#### Scenario: No secrets in URLs
- **WHEN** the UI or API constructs URLs for browser navigation or websocket connections
- **THEN** secrets (tokens, passwords, API keys) are not placed in the URL query string
