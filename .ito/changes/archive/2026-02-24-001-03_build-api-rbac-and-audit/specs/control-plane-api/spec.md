## ADDED Requirements

### Requirement: Session and Run Lifecycle API
The system SHALL expose authenticated REST endpoints to create, list, inspect, start, stop, and resume sessions and harness runs.

#### Scenario: User starts a run from an existing session
- **WHEN** an authorized caller submits a run start request for a valid session
- **THEN** the API persists the run intent, returns a run identifier, and reports initial lifecycle status

### Requirement: Authorization and Audit Enforcement
The system SHALL enforce scope-based RBAC checks and append audit events for security-sensitive API actions.

#### Scenario: Unauthorized action denied and audited
- **WHEN** a caller without required scope attempts a protected mutation
- **THEN** the API rejects the request and records an audit event describing the denied action
