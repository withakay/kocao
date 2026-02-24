## ADDED Requirements

### Requirement: Repository Bootstrap
The system SHALL provide a consistent repository layout and bootstrap commands for control-plane services, harness assets, and the web client.

#### Scenario: Fresh checkout bootstrap
- **WHEN** a contributor clones the repository and runs the documented bootstrap command
- **THEN** required dependencies and generated assets are prepared without manual file edits

### Requirement: Configuration Baseline
The system SHALL load required runtime configuration from environment-aware sources and fail fast on invalid startup configuration.

#### Scenario: Invalid configuration rejected at startup
- **WHEN** a required configuration value is missing or malformed
- **THEN** startup exits with an actionable validation error before serving traffic

### Requirement: Local Kind Development Workflow
The system SHALL provide a documented and scriptable local development workflow using `kind`, including support for OrbStack-built local images.

#### Scenario: Developer deploys local images without a remote registry
- **WHEN** a contributor builds platform images locally in OrbStack and runs the documented `kind` image-load workflow
- **THEN** the local cluster runs the updated images without requiring a push to an external container registry
