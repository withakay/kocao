## ADDED Requirements

### Requirement: Codex App-Server Session Startup
The system SHALL start a Codex app-server process in the claimed issue workspace and perform the required initialize, thread-start, and turn-start handshake before streaming work.

#### Scenario: Session starts successfully
- **WHEN** the worker launches a valid Codex app-server command in the issue workspace
- **THEN** the worker records the thread and turn identifiers and begins streaming turn events

### Requirement: Turn Lifecycle Handling
The agent runner SHALL treat completed, failed, cancelled, timed-out, and stalled turns as distinct outcomes for orchestration and retry behavior.

#### Scenario: Stalled turn is terminated
- **WHEN** no app-server activity is observed beyond the configured stall timeout
- **THEN** the worker terminates the session and reports a stalled outcome for retry handling

### Requirement: Continuation Turns Reuse Thread Context
The agent runner SHALL reuse the same live thread for continuation turns while the issue remains active and the worker has not reached its configured turn limit.

#### Scenario: Active issue continues on the same thread
- **WHEN** a turn completes successfully and the issue is still active
- **THEN** the next continuation turn starts on the existing thread instead of opening a new session

### Requirement: Session Telemetry Extraction
The agent runner SHALL extract session identifiers, token usage, rate-limit snapshots, and human-readable event summaries from app-server messages when available.

#### Scenario: Token telemetry is reported
- **WHEN** the app-server emits absolute token usage updates for a running session
- **THEN** the worker publishes bounded token totals to Symphony runtime status without double-counting prior totals
