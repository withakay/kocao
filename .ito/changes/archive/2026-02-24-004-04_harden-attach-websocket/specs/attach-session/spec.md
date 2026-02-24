## ADDED Requirements

### Requirement: Websocket Origin Validation
The system SHALL validate websocket Origins for attach connections.

#### Scenario: Unexpected origin is rejected
- **WHEN** a websocket upgrade request includes an Origin not in the allowlist
- **THEN** the server rejects the upgrade

### Requirement: Websocket Resource Limits
The system SHALL enforce websocket message size limits and connection deadlines.

#### Scenario: Oversized message closes connection
- **WHEN** a client sends a message larger than the configured maximum
- **THEN** the server closes the connection without exhausting memory

### Requirement: Safe Token Transport for Browser Attach
The system SHALL support a browser attach flow that does not place secrets in URLs.

#### Scenario: Browser attach without URL token
- **WHEN** a user opens the attach page in the UI
- **THEN** the websocket connection is established without putting the attach token in the URL query string

### Requirement: Attach Auditing
The system SHALL record attach lifecycle and control events as audit events.

#### Scenario: Control takeover is auditable
- **WHEN** a client takes control of an attach session
- **THEN** an audit event is recorded including the session ID and client ID
