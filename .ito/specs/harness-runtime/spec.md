## ADDED Requirements

### Requirement: Reproducible Harness Runtime Image
The system SHALL provide a versioned harness image that includes the approved runtime matrix and required coding-agent tooling for MVP workloads.

#### Scenario: Harness pod starts with required toolchain availability
- **WHEN** a run pod starts from the published harness image
- **THEN** the required language runtimes and core CLI tools are available and report expected versions

### Requirement: Harness Pod Execution Contract
The system SHALL enforce a standard run-pod contract for workspace mounting, process startup, and secure credential injection.

#### Scenario: Run pod receives scoped credentials
- **WHEN** a run is created with valid repository credentials
- **THEN** the run pod receives credentials through the defined secure injection path without exposing them in logs
