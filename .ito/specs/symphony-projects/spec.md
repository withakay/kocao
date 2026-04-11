## MODIFIED Requirements

### Requirement: Symphony Project Configuration
The system SHALL provide a namespaced Symphony Project resource that defines one GitHub Projects v2 orchestration target, its scheduling settings, and its runtime defaults. The control-plane create and update flow SHALL support operator-managed GitHub token secret creation and SHALL store the resulting reference in `spec.source.tokenSecretRef`.

#### Scenario: Operator creates a Symphony project with an inline GitHub token
- **WHEN** an operator creates a Symphony Project through the control-plane using a GitHub PAT value
- **THEN** the system derives a deterministic Secret name from the project name and GitHub owner, stores the token in that Secret, and persists the resulting `tokenSecretRef`

#### Scenario: Operator updates a Symphony project without a new GitHub token
- **WHEN** an operator updates a Symphony Project and leaves the write-only token field empty
- **THEN** the existing `tokenSecretRef` remains unchanged
