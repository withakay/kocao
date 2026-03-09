## ADDED Requirements

### Requirement: Symphony Project CRUD Endpoints
The API SHALL provide authenticated endpoints to create, list, and fetch Symphony Projects.

#### Scenario: Client creates a Symphony project
- **WHEN** an authorized client submits a valid Symphony Project configuration to the API
- **THEN** the API persists the project resource and returns its initial configuration and status

#### Scenario: Client lists Symphony projects
- **WHEN** an authorized client calls the Symphony project list endpoint
- **THEN** the API returns the configured Symphony Projects with summary runtime status

### Requirement: Symphony Project Control Endpoints
The API SHALL allow authorized clients to pause, resume, and trigger an immediate refresh for a Symphony Project.

#### Scenario: Client pauses a Symphony project
- **WHEN** an authorized client invokes the Symphony project pause action
- **THEN** the API updates the project so no new items are claimed until resumed

### Requirement: Symphony Runtime Detail Endpoint
The API SHALL provide per-project runtime state including active items, retry queue entries, linked Session/HarnessRun identities, and recent errors.

#### Scenario: Client fetches Symphony runtime detail
- **WHEN** an authorized client requests Symphony project detail
- **THEN** the API returns bounded runtime state for the project and its currently tracked work
