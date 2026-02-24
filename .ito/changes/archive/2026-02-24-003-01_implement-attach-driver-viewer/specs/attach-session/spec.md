## ADDED Requirements

### Requirement: Session-scoped attach credentials
The system SHALL provide short-lived, session-scoped attach credentials that allow a client to connect to an attach session without granting unrelated API access.

#### Scenario: Issue attach token
- **WHEN** an authenticated client requests an attach token for a session
- **THEN** the system returns a token that expires within a short time window

#### Scenario: Reject expired attach token
- **WHEN** a client attempts to attach using an expired token
- **THEN** the system denies the attach request

### Requirement: Driver and viewer roles
The system SHALL support one interactive driver and multiple read-only viewers per session.

#### Scenario: Viewer cannot send input
- **WHEN** a viewer sends terminal input
- **THEN** the system rejects the input

#### Scenario: Only one driver at a time
- **WHEN** a second client attempts to become the driver while another driver lease is active
- **THEN** the system preserves the existing driver lease

### Requirement: Lease-based driver transfer and reconnect
The system SHALL use a time-bound driver lease to allow safe control handoff and driver reconnect without requiring the pod to restart.

#### Scenario: Driver disconnect and reconnect
- **WHEN** the driver disconnects and reconnects before the lease expires
- **THEN** the driver retains control

#### Scenario: Take control after lease expiry
- **WHEN** the driver lease expires
- **THEN** another client can take control and become the driver

### Requirement: Attach stream fanout
The system SHALL bridge the run pod terminal stream to all connected clients, fanning out output to viewers and the driver.

#### Scenario: Broadcast output to all clients
- **WHEN** the run pod emits terminal output
- **THEN** all connected clients receive the output
