## ADDED Requirements

### Requirement: Audit Log Configuration
The system SHALL configure audit persistence using `CP_AUDIT_PATH` and SHALL NOT conflate audit log paths with database configuration.

#### Scenario: Default audit log path
- **WHEN** `CP_AUDIT_PATH` is not set
- **THEN** the API uses a safe default path and documents where audit events are persisted

#### Scenario: Explicit audit log path
- **WHEN** `CP_AUDIT_PATH` is set
- **THEN** audit events are appended to that path with file permissions `0600`

### Requirement: Audit Integrity
The audit log SHALL be append-only and each event SHALL be encoded as a single JSON object per line.

#### Scenario: Append-only encoding
- **WHEN** an audit event is recorded
- **THEN** the store appends exactly one JSON line and does not rewrite prior content
