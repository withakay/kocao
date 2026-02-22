## ADDED Requirements

### Requirement: Request Size Limits
The system SHALL enforce a maximum request body size for JSON API endpoints.

#### Scenario: Oversized request is rejected
- **WHEN** a client sends a JSON request body larger than the configured limit
- **THEN** the API responds with HTTP 413 or HTTP 400 and does not exhaust memory

### Requirement: Defensive JSON Parsing
The system SHALL reject unknown fields for JSON request bodies.

#### Scenario: Unknown fields are rejected
- **WHEN** a client sends a request body containing unknown JSON fields
- **THEN** the API responds with HTTP 400

### Requirement: Server Timeouts
The system SHALL configure HTTP server timeouts appropriate for a control-plane service.

#### Scenario: Slow client does not stall the server
- **WHEN** a client attempts to hold open connections without completing requests
- **THEN** server timeouts limit resource usage

### Requirement: Bootstrap Token Safety
The system SHALL treat `CP_BOOTSTRAP_TOKEN` as a bring-up-only mechanism and SHALL prevent its use in production.

#### Scenario: Production mode rejects bootstrap token
- **WHEN** `CP_ENV=prod` and `CP_BOOTSTRAP_TOKEN` is configured
- **THEN** startup fails or the bootstrap token is ignored with a clear warning
