<!-- ITO:START -->
## ADDED Requirements

### Requirement: Agent Selection and Launch Experience
The web UI SHALL let an authorized user choose a supported agent (`opencode`, `claude`, `codex`, or `pi`) when starting a sandbox-backed run/session from the Kocao workflow surfaces.

- **Requirement ID**: agent-session-ui:agent-selection-and-launch

#### Scenario: User launches a Codex-backed session
- **WHEN** the user selects `codex` in the launch flow and submits the run
- **THEN** the UI calls the Kocao API with the selected agent and shows the resulting sandbox-backed session state

### Requirement: Live Agent Interaction View
The web UI SHALL provide a live interaction surface for sandbox-backed sessions that shows session state, normalized transcript/events, and an input affordance for follow-up prompts.

- **Requirement ID**: agent-session-ui:live-agent-interaction-view

#### Scenario: User interacts with a running session
- **WHEN** a sandbox-backed session is active
- **THEN** the UI renders the current transcript/event stream and allows the user to send a follow-up prompt without leaving the run view

### Requirement: Lifecycle Controls and Reconnect
The web UI SHALL let an authorized user stop, resume, or reconnect to a sandbox-backed session and SHALL preserve transcript continuity across reloads or transient disconnects.

- **Requirement ID**: agent-session-ui:lifecycle-controls-and-reconnect

#### Scenario: User reloads the page during an active session
- **WHEN** the browser reloads while a sandbox-backed session is still active
- **THEN** the UI reconnects to the Kocao API, reloads persisted events from the last known offset, and continues showing the same session history
<!-- ITO:END -->
