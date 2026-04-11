## ADDED Requirements

### Requirement: Symphony Project Configuration
The system SHALL provide a namespaced Symphony Project resource that defines one GitHub Projects v2 orchestration target, its scheduling settings, and its runtime defaults.

#### Scenario: Operator creates a Symphony project
- **WHEN** an operator creates a Symphony Project with a GitHub project owner, project number, and valid runtime configuration
- **THEN** the resource is accepted and its status is initialized for orchestration

### Requirement: Supported Repository Catalog
Each Symphony Project SHALL declare the GitHub repositories that may be executed for project items, including per-repository clone and authentication settings.

#### Scenario: Item resolves to a configured repository
- **WHEN** a GitHub project item points to an issue in a repository listed in the Symphony Project
- **THEN** the orchestrator resolves execution settings from the matching repository entry

#### Scenario: Item resolves to an unconfigured repository
- **WHEN** a GitHub project item points to an issue in a repository that is not listed in the Symphony Project
- **THEN** the item is skipped and the project status records an unsupported-repository reason

### Requirement: Project Pause Control
The system SHALL allow operators to pause a Symphony Project without deleting its configuration.

#### Scenario: Paused project stops new claims
- **WHEN** a Symphony Project is paused
- **THEN** no new project items are claimed until the project is resumed
