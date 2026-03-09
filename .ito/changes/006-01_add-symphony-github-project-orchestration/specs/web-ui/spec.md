## ADDED Requirements

### Requirement: Symphony Navigation
The web UI SHALL expose Symphony project management through the main application navigation and route tree.

#### Scenario: User opens Symphony from navigation
- **WHEN** a user selects Symphony in the application shell
- **THEN** the UI navigates to the Symphony project list view

### Requirement: Symphony Project Configuration Surface
The web UI SHALL allow operators to create and update Symphony project configuration, including GitHub project identity, repository allowlist entries, and secret reference names.

#### Scenario: User configures a new Symphony project
- **WHEN** an operator completes the Symphony project form with valid GitHub and repository settings
- **THEN** the UI submits the project configuration and shows the created project

### Requirement: Symphony Runtime Status View
The web UI SHALL show project runtime status including active items, retrying items, skip reasons, and links to GitHub issues and child Kocao runs.

#### Scenario: User inspects runtime status
- **WHEN** the Symphony project detail page loads
- **THEN** the UI renders the current active and retrying work plus links to the source issue and child execution objects

### Requirement: Symphony Operator Controls
The web UI SHALL provide pause, resume, and refresh controls for Symphony projects.

#### Scenario: User pauses a Symphony project from the UI
- **WHEN** an operator clicks Pause for a Symphony project
- **THEN** the UI invokes the control-plane API and updates the visible project state to paused
