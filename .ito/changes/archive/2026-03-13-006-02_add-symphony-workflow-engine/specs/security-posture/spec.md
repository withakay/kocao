## ADDED Requirements

### Requirement: Symphony Workflow Trust Posture
The system SHALL document and enforce the approval, sandbox, and hook-execution posture used for workflow-driven Symphony runs.

#### Scenario: Operator inspects Symphony execution posture
- **WHEN** a Symphony project is configured for workflow-driven execution
- **THEN** the system defines the approval and sandbox posture applied to Codex turns and workflow hooks

### Requirement: Workflow and Agent Secrets Stay Bounded
The system SHALL expose only the secrets required for the claimed repository and SHALL prevent workflow or agent telemetry surfaces from disclosing raw credential values.

#### Scenario: Worker telemetry remains secret-safe
- **WHEN** Symphony runtime status includes worker and agent details
- **THEN** secret values are not exposed in status, logs, or audit metadata
