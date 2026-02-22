## ADDED Requirements

### Requirement: Driver and Viewer Attach Roles
The system SHALL support one interactive attach driver and multiple concurrent read-only viewers per run.

#### Scenario: Additional viewers join an active run
- **WHEN** multiple authorized clients attach to a running session
- **THEN** exactly one client has interactive input privileges and all others receive read-only output

### Requirement: Attach Reconnect and Control Transfer
The system SHALL issue short-lived attach credentials and preserve run attach state across transient client disconnects, including explicit control transfer.

#### Scenario: Driver disconnects and reconnects
- **WHEN** the active driver connection drops and reconnects with a valid attach token
- **THEN** the session continues without pod restart and driver privileges are restored or reassigned according to lease rules
