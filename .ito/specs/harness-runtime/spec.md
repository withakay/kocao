## ADDED Requirements

### Requirement: Workspace Path Safety
The harness entrypoint SHALL prevent destructive operations outside the configured workspace directory.

#### Scenario: Repo dir outside workspace is rejected
- **WHEN** the harness is configured with a repo directory outside the workspace
- **THEN** the entrypoint fails fast and does not delete any directories

### Requirement: Reserved Environment Variables
The system SHALL treat critical harness environment variables as reserved and SHALL prevent user overrides.

#### Scenario: User env cannot override reserved KOCAO vars
- **WHEN** a run specifies an environment variable that conflicts with reserved `KOCAO_*` variables
- **THEN** the operator rejects the spec or drops the override

### Requirement: Non-Root Harness Container
The harness container SHALL run as a non-root user by default and SHALL drop unnecessary Linux capabilities.

#### Scenario: Hardened security context
- **WHEN** a harness pod is created
- **THEN** the container runs as non-root where possible and has a restrictive security context
