## ADDED Requirements

### Requirement: Controlled Egress Modes
The system SHALL enforce default-deny outbound traffic for run pods while allowing a GitHub-only baseline mode and an explicit full-internet override mode.

#### Scenario: Default run receives restricted outbound policy
- **WHEN** a run is created without egress override
- **THEN** the run pod is attached to network policy rules that permit only required GitHub endpoints

### Requirement: Session Artifact Persistence and Restore
The system SHALL persist session artifacts on terminal run transitions and restore them before session resume.

#### Scenario: Session resumes after prior run termination
- **WHEN** a user resumes a session with previously persisted artifacts
- **THEN** the new run pod starts with the restored session bundle and prior context is available
