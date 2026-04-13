<!-- ITO:START -->
## ADDED Requirements

### Requirement: Agent Subcommand Group
The CLI SHALL provide a top-level `agent` subcommand group under the `kocao` binary that organizes all sandbox-agent management operations. Running `kocao agent` without a subcommand SHALL print usage help listing available subcommands.

- **Requirement ID**: cli-agent-management:agent-subcommand-group

#### Scenario: User runs agent with no subcommand
- **WHEN** a user runs `kocao agent`
- **THEN** the CLI prints usage help listing the available agent subcommands (list, start, stop, logs, exec, status) and exits with code 0

#### Scenario: Root help includes agent command
- **WHEN** a user runs `kocao help` or `kocao`
- **THEN** the output includes `agent` in the list of available commands with a brief description

### Requirement: Agent List Command
The CLI SHALL provide `kocao agent list` to list sandbox-agent sessions across all workspace sessions. The command SHALL support filtering by workspace session ID and output format selection.

- **Requirement ID**: cli-agent-management:agent-list-command

#### Scenario: List all agent sessions with table output
- **WHEN** a user runs `kocao agent list`
- **THEN** the CLI displays a table with columns: SESSION-ID, AGENT, WORKSPACE, PHASE, CREATED
- **AND** sessions are sorted newest-first

#### Scenario: List agent sessions filtered by workspace
- **WHEN** a user runs `kocao agent list --workspace <ws-id>`
- **THEN** the CLI displays only agent sessions belonging to that workspace session

#### Scenario: List agent sessions with JSON output
- **WHEN** a user runs `kocao agent list --output json`
- **THEN** the CLI outputs a JSON object with an `agentSessions` array

### Requirement: Agent Start Command
The CLI SHALL provide `kocao agent start` to create a new sandbox-agent session. The command SHALL create the full resource chain (workspace session, harness run, agent session) and SHALL wait for the agent to reach ready state before returning, with a configurable timeout.

- **Requirement ID**: cli-agent-management:agent-start-command

#### Scenario: Start an agent successfully
- **WHEN** a user runs `kocao agent start --repo https://github.com/org/repo --agent claude`
- **THEN** the CLI creates a workspace session, starts a harness run with the sandbox-agent image, creates an agent session for `claude`, waits until the agent reports ready, and prints the session ID and status

#### Scenario: Start with an existing workspace session
- **WHEN** a user runs `kocao agent start --workspace <ws-id> --repo https://github.com/org/repo --agent opencode`
- **THEN** the CLI reuses the specified workspace session rather than creating a new one

#### Scenario: Start times out waiting for ready
- **WHEN** a user runs `kocao agent start --agent codex --repo <url> --timeout 30s` and the agent does not become ready within 30 seconds
- **THEN** the CLI prints the current status, warns that the agent is not yet ready, prints the session ID for later use, and exits with code 1

#### Scenario: Start with an unsupported agent name
- **WHEN** a user runs `kocao agent start --agent unsupported --repo <url>`
- **THEN** the CLI rejects the request with a validation error listing supported agents (opencode, claude, codex, pi)

### Requirement: Agent Stop Command
The CLI SHALL provide `kocao agent stop <session-id>` to stop a running sandbox-agent session.

- **Requirement ID**: cli-agent-management:agent-stop-command

#### Scenario: Stop a running agent session
- **WHEN** a user runs `kocao agent stop <session-id>`
- **THEN** the CLI sends a stop request to the control plane API, waits for confirmation, and prints the terminal status

#### Scenario: Stop a non-existent session
- **WHEN** a user runs `kocao agent stop <invalid-id>`
- **THEN** the CLI prints "agent session not found" and exits with code 1

### Requirement: Agent Logs Command
The CLI SHALL provide `kocao agent logs <session-id>` to stream normalized events from a sandbox-agent session. The command SHALL support both snapshot and follow modes.

- **Requirement ID**: cli-agent-management:agent-logs-command

#### Scenario: Tail agent session events
- **WHEN** a user runs `kocao agent logs <session-id>`
- **THEN** the CLI fetches and displays recent normalized events from the agent session

#### Scenario: Follow agent session events in real-time
- **WHEN** a user runs `kocao agent logs <session-id> --follow`
- **THEN** the CLI opens an SSE stream and displays events as they arrive until the session reaches a terminal state or the user presses Ctrl+C

#### Scenario: Logs with JSON output
- **WHEN** a user runs `kocao agent logs <session-id> --output json`
- **THEN** each event is printed as a JSON object on a separate line (JSONL format)

### Requirement: Agent Exec Command
The CLI SHALL provide `kocao agent exec <session-id> --prompt "..."` to send a prompt to a running sandbox-agent session and display the response events.

- **Requirement ID**: cli-agent-management:agent-exec-command

#### Scenario: Send a prompt and display response
- **WHEN** a user runs `kocao agent exec <session-id> --prompt "Fix the failing tests"`
- **THEN** the CLI sends the prompt to the agent session, streams response events to stdout as they arrive, and exits when the response is complete

#### Scenario: Exec against an inactive session
- **WHEN** a user runs `kocao agent exec <session-id> --prompt "..."` and the session is not in an active state
- **THEN** the CLI prints an error indicating the session is not active and exits with code 1

#### Scenario: Exec with JSON output
- **WHEN** a user runs `kocao agent exec <session-id> --prompt "..." --output json`
- **THEN** each response event is printed as a JSON object (JSONL format)

### Requirement: Agent Status Command
The CLI SHALL provide `kocao agent status <session-id>` to display detailed status of a single agent session including its workspace session, harness run, agent type, phase, and timing information.

- **Requirement ID**: cli-agent-management:agent-status-command

#### Scenario: Display agent session status
- **WHEN** a user runs `kocao agent status <session-id>`
- **THEN** the CLI prints a key-value detail view showing: Session ID, Agent, Workspace Session ID, Harness Run ID, Phase, Pod Name, Created At, and Last Event timestamp

#### Scenario: Status with JSON output
- **WHEN** a user runs `kocao agent status <session-id> --output json`
- **THEN** the CLI outputs the full status as a JSON object

### Requirement: Output Format Support
All agent subcommands SHALL support `--output <format>` where format is one of `table` (default), `json`, or `yaml`. Commands that produce list output SHALL default to `table`; detail views SHALL default to key-value text.

- **Requirement ID**: cli-agent-management:output-format-support

#### Scenario: JSON output for scripting
- **WHEN** any agent subcommand is invoked with `--output json`
- **THEN** the output is valid JSON suitable for piping to `jq` or programmatic consumption

#### Scenario: Default human-readable output
- **WHEN** any agent subcommand is invoked without `--output`
- **THEN** the output uses the default human-readable format (table for lists, key-value for details)

### Requirement: Agent Client Methods
The `controlplanecli.Client` SHALL be extended with methods for agent session operations: list, create, get, stop, send prompt, and stream events. These methods SHALL follow the existing `doJSON` pattern and error handling conventions.

- **Requirement ID**: cli-agent-management:agent-client-methods

#### Scenario: Client method returns typed response
- **WHEN** a client method for agent sessions is called with valid parameters
- **THEN** it returns a typed Go struct matching the API response schema and nil error

#### Scenario: Client method propagates API errors
- **WHEN** a client method receives a non-2xx response
- **THEN** it returns an `*APIError` with status code and message, following the existing error pattern
<!-- ITO:END -->
