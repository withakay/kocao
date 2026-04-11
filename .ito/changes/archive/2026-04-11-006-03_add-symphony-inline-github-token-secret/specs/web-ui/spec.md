## ADDED Requirements

### Requirement: Symphony Managed Token Input
The web UI SHALL provide a write-only GitHub PAT input for Symphony project configuration and SHALL avoid asking operators to type a raw Kubernetes Secret name during the standard create flow.

#### Scenario: User creates a Symphony project from the UI
- **WHEN** an operator submits the Symphony project form with a GitHub PAT
- **THEN** the UI sends the write-only token field to the API and does not display the token after submission

#### Scenario: User edits a Symphony project without rotating the token
- **WHEN** an operator opens the edit form for an existing Symphony project
- **THEN** the PAT field is blank and optional, while the managed Secret reference is shown read-only for diagnostics
