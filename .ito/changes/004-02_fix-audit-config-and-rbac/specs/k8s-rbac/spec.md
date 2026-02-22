## ADDED Requirements

### Requirement: Least-Privilege Service Accounts
The system SHALL run the control-plane API and operator under distinct Kubernetes service accounts with least-privilege roles.

#### Scenario: Separate service accounts
- **WHEN** deployed using the provided manifests
- **THEN** the API and operator pods run under different service accounts and role bindings

### Requirement: Attach Exec Permission
The control-plane API SHALL have the minimum RBAC required to exec into harness pods to support attach.

#### Scenario: Attach can exec
- **WHEN** a session has an active harness pod
- **THEN** the API can perform `pods/exec` in the configured namespace to attach to the harness container
