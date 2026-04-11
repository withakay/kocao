<!-- ITO:START -->
## ADDED Requirements

### Requirement: Sandbox Agent Session Management Endpoints
The API SHALL provide authenticated Kocao-managed endpoints to create, inspect, and identify sandbox-agent-backed sessions associated with a Workspace Session and Harness Run.

- **Requirement ID**: control-plane-api:sandbox-agent-session-management-endpoints

#### Scenario: Client creates a sandbox-backed session through Kocao API
- **WHEN** an authorized client requests a supported agent session for a running Harness Run
- **THEN** the Kocao API validates the request, creates the corresponding sandbox-agent session, and returns Kocao-owned session metadata without exposing raw provider credentials

### Requirement: Sandbox Agent Messaging and Event Replay API
The API SHALL provide authenticated endpoints to send prompts/messages to a sandbox-backed session and to read back normalized events by sequence offset for replay-safe clients. The API SHALL also provide a live streaming path for incremental event delivery during an active session.

- **Requirement ID**: control-plane-api:sandbox-agent-messaging-and-event-replay-api

#### Scenario: Client resumes reading from a prior event offset
- **WHEN** an authorized client requests session events with the last persisted offset
- **THEN** the API returns only events after that offset in stable sequence order

#### Scenario: Client receives live incremental events
- **WHEN** an authorized client opens the live event stream for an active sandbox-backed session
- **THEN** the API forwards incremental normalized events as they are produced and closes the stream cleanly when the session reaches a terminal state

### Requirement: Sandbox Agent Lifecycle Control API
The API SHALL provide authenticated lifecycle controls to stop, resume, and inspect sandbox-backed sessions and SHALL surface normalized readiness and terminal-state information.

- **Requirement ID**: control-plane-api:sandbox-agent-lifecycle-control-api

#### Scenario: Client stops an active sandbox-backed session
- **WHEN** an authorized client issues a stop request for an active sandbox-backed session
- **THEN** the API requests graceful shutdown through Kocao's mediated lifecycle path and returns updated terminal or stopping state information

<!-- ITO:END -->
