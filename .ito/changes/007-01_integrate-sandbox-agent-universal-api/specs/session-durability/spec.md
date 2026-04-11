<!-- ITO:START -->
## ADDED Requirements

### Requirement: Sandbox Agent Event History Is Durable
The system SHALL persist normalized sandbox-agent events and session metadata outside the in-memory sandbox-agent process so that transcript history survives API reconnects and pod churn.

- **Requirement ID**: session-durability:sandbox-agent-event-history-is-durable

#### Scenario: Client reconnects after a disconnect
- **WHEN** a client disconnects from an active sandbox-backed session and reconnects later
- **THEN** Kocao serves the persisted transcript/events from the last acknowledged offset without requiring the user to restart the session

### Requirement: Resumed Runs Preserve Agent Session Context
The system SHALL preserve enough sandbox-backed session context across Harness Run resume flows for users to understand prior interaction history and continue from the same Workspace Session context.

- **Requirement ID**: session-durability:resumed-runs-preserve-agent-session-context

#### Scenario: Harness Run is resumed after termination
- **WHEN** a Workspace Session starts a replacement Harness Run after the prior run terminated
- **THEN** Kocao restores the persisted session metadata and transcript history so the resumed run can continue with the same workspace context and visible prior history

## MODIFIED Requirements

### Requirement: Workspace Session Is the Durable Workspace Anchor
The system SHALL define Workspace Session as the durable unit that anchors repository identity, workspace storage, policy context, and sandbox-backed session history across multiple Harness Runs.

- **Requirement ID**: session-durability:workspace-session-is-the-durable-workspace-anchor

#### Scenario: Multiple runs share one workspace and transcript anchor
- **WHEN** multiple Harness Runs are created under one Workspace Session
- **THEN** each run uses the same workspace and the same durable session-history anchor defined by that Workspace Session
<!-- ITO:END -->
