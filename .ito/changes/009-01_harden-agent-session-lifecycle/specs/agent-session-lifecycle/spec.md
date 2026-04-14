<!-- ITO:START -->
## ADDED Requirements

### Requirement: Explicit Agent Session State Machine

The system SHALL model sandbox-backed agent sessions with an explicit durable state machine instead of relying on inferred bridge or pod behavior.

- **Requirement ID**: agent-session-lifecycle:explicit-state-machine

#### Scenario: Session transitions through known states

- **WHEN** a client creates and uses an agent session
- **THEN** the session transitions through a defined subset of `Provisioning`, `Ready`, `Running`, `Stopping`, `Completed`, and `Failed`

#### Scenario: Terminal states are durable

- **WHEN** a session reaches `Completed` or `Failed`
- **THEN** the terminal state is persisted and remains observable after API or operator restart

### Requirement: Restart Reconciliation

The system SHALL reconcile agent session state after control-plane restart and SHALL reconstruct enough state to answer lifecycle queries without relying solely on in-memory bridges.

- **Requirement ID**: agent-session-lifecycle:restart-reconciliation

#### Scenario: API restarts during an active session

- **WHEN** the control-plane API restarts while a harness pod and sandbox-agent are still running
- **THEN** a subsequent session status request returns a valid lifecycle state without reporting false success or false completion

#### Scenario: Stop requested after restart

- **WHEN** a client requests stop for a session after the API has restarted
- **THEN** the system either completes the stop successfully or returns an explicit error saying stop could not be confirmed

### Requirement: Idempotent Session Operations

The session lifecycle APIs SHALL be idempotent for repeated create, get, stop, logs, and prompt operations.

- **Requirement ID**: agent-session-lifecycle:idempotent-operations

#### Scenario: Repeated create request for the same run

- **WHEN** a client calls create-session multiple times for the same harness run
- **THEN** the API returns the same logical session identity and current state instead of creating duplicates

#### Scenario: Repeated stop request on completed session

- **WHEN** a client calls stop on a session that is already completed
- **THEN** the API responds predictably without hanging or mutating the session into an invalid state
<!-- ITO:END -->
