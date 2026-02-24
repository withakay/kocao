## ADDED Requirements

### Requirement: Command-Configured Harness Run Execution
The system SHALL define a Harness Run as a single harness pod execution attempt that runs configured command and argument values against the checked-out repository and exits when the command completes.

#### Scenario: Run executes non-interactively by default
- **WHEN** a Harness Run is created with a configured command such as `go test ./...`
- **THEN** the harness pod executes that command to completion without requiring interactive attach

### Requirement: Attach Is an Optional Capability on a Running Harness Run
The system SHALL treat attach as an optional capability that connects a human to an already running Harness Run pod for viewer/driver interaction over websocket plus pod exec.

#### Scenario: Human attaches to an active run
- **WHEN** a user invokes attach for a running Harness Run
- **THEN** the system connects to the existing run pod and does not create a separate run pod for interactivity
