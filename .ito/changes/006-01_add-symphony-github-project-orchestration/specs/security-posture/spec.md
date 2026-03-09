## ADDED Requirements

### Requirement: Symphony Tracker Secrets Remain Secret-Scoped
The system SHALL store GitHub project credentials in Kubernetes Secrets referenced by Symphony Projects and SHALL NOT expose secret values in logs, API responses, UI surfaces, or status fields.

#### Scenario: Project detail omits secret material
- **WHEN** an operator reads Symphony project configuration through the API or UI
- **THEN** only secret reference names are visible and raw token values are never returned

### Requirement: Worker Pods Receive Only Explicit Execution Credentials
Symphony worker pods SHALL receive only the repository-specific Git and agent credentials explicitly configured for the resolved repository, and SHALL NOT receive the GitHub board polling secret.

#### Scenario: Board token stays out of worker pod
- **WHEN** a Symphony item launches a worker HarnessRun
- **THEN** the resulting pod mounts only the configured execution credentials for the target repository and not the board polling secret

### Requirement: Symphony Actions Are Auditable
The system SHALL record audit events for Symphony project sync, claim, retry, release, pause/resume, and terminal completion transitions.

#### Scenario: Claim transition is auditable
- **WHEN** a Symphony Project claims a GitHub issue for execution
- **THEN** an audit event records the project identity, issue identity, transition, and outcome without exposing secret values

### Requirement: Symphony Worker Egress Defaults
Symphony-created worker pods SHALL default to restricted egress unless a Symphony Project explicitly opts into a broader execution policy.

#### Scenario: Default worker egress is restricted
- **WHEN** a Symphony Project launches a worker without an explicit egress override
- **THEN** the worker pod uses the restricted egress policy by default
