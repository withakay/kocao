<!-- ITO:START -->
## ADDED Requirements

### Requirement: Unified Sandbox Agent Session Contract
The system SHALL use `sandbox-agent` as the only generic in-sandbox control API for supported coding agents in the standard Kocao harness flow. Kocao SHALL not require provider-specific control paths for OpenCode, Claude, Codex, or Pi once a run is using the sandbox-agent path.

- **Requirement ID**: sandbox-agent-runtime:unified-session-contract

#### Scenario: Supported agent session is created through one runtime contract
- **WHEN** an authorized client requests a sandbox-backed agent session for `opencode`, `claude`, `codex`, or `pi`
- **THEN** Kocao provisions or reuses the target Harness Run and creates the in-sandbox session through `sandbox-agent` rather than a provider-specific adapter path

#### Scenario: Unsupported agent is rejected consistently
- **WHEN** a client requests an agent outside the supported catalog
- **THEN** Kocao rejects the request with a validation error before attempting to start a session in the harness pod

### Requirement: Supported Agent Catalog
The system SHALL support starting sandbox-agent-backed sessions for `opencode`, `claude`, `codex`, and `pi` in Kubernetes-managed harness pods.

- **Requirement ID**: sandbox-agent-runtime:supported-agent-catalog

#### Scenario: Operator chooses a supported agent
- **WHEN** a run is started with one of the supported agent identifiers
- **THEN** the resulting session metadata records the chosen agent and the harness run starts that agent through `sandbox-agent`

### Requirement: Normalized Agent Lifecycle
The system SHALL expose a normalized lifecycle for sandbox-agent-backed sessions that distinguishes provisioning, ready, active, stopping, completed, and failed states regardless of the underlying agent implementation.

- **Requirement ID**: sandbox-agent-runtime:normalized-agent-lifecycle

#### Scenario: Session becomes ready before interaction
- **WHEN** the harness pod is running and `sandbox-agent` reports healthy session startup
- **THEN** Kocao marks the sandbox-backed session as ready for messages before reporting it as interactable to API or UI clients

#### Scenario: Provider-specific failure is normalized
- **WHEN** the underlying agent process exits unexpectedly or sandbox-agent reports a terminal error
- **THEN** Kocao records a normalized failed state with diagnostic context that can be surfaced through the API and UI

### Requirement: Ordered Prompt and Event Interaction
The system SHALL send prompts/messages to the active sandbox-agent session and SHALL expose the resulting normalized events in stable sequence order.

- **Requirement ID**: sandbox-agent-runtime:ordered-prompt-and-event-interaction

#### Scenario: Prompt produces ordered events
- **WHEN** a client sends a prompt to an active sandbox-backed session
- **THEN** Kocao forwards the prompt to `sandbox-agent` and returns or streams normalized events in sequence order for replay-safe consumption
<!-- ITO:END -->
