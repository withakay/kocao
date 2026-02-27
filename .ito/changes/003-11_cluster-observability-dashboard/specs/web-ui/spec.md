## ADDED Requirements

### Requirement: Cluster Dashboard Navigation
The web UI SHALL expose the cluster dashboard through shell navigation and command palette actions.

#### Scenario: User navigates from shell
- **WHEN** a user clicks the Cluster entry in sidebar or selects Cluster in the command palette
- **THEN** the UI navigates to the `/cluster` route

### Requirement: Cluster Config Visibility
The web UI SHALL present non-secret runtime configuration indicators relevant to cluster operations.

#### Scenario: User reviews runtime config indicators
- **WHEN** the cluster dashboard loads
- **THEN** the UI shows safe config indicators (for example environment and feature toggles) without revealing secret values
