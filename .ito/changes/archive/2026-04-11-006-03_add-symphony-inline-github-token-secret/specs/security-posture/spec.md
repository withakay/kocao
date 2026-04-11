## ADDED Requirements

### Requirement: Symphony Inline Token Redaction
The system SHALL treat inline Symphony GitHub token input as write-only secret material and SHALL NOT expose it in API responses, audit metadata, validation errors, logs, or UI echoes.

#### Scenario: Validation fails after token input
- **WHEN** a Symphony create or update request includes a GitHub token and another field fails validation
- **THEN** the returned error response omits the raw token value

#### Scenario: Audit metadata records Symphony create
- **WHEN** the API records audit metadata for a Symphony create or update request that included a GitHub token
- **THEN** the audit event contains only redacted or derived Secret-reference information and never the raw token value
