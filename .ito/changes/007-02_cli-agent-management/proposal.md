<!-- ITO:START -->
## Why
After PR #4 merged the sandbox-agent integration, the control plane API can manage agent sessions (create, prompt, stop, stream events), but operators have no CLI interface to these capabilities. Every interaction requires raw HTTP calls or the web UI. A CLI is critical for scripting, CI/CD pipelines, headless environments, and developer ergonomics — especially for operators who live in the terminal and need to quickly spin up an agent, send a prompt, and tail events without leaving their shell.

## What Changes
- Add a new `agent` top-level subcommand to the `kocao` CLI with subcommands: `list`, `start`, `stop`, `logs`, `exec`, and `status`.
- Extend the existing `controlplanecli.Client` with methods for agent session CRUD, prompt sending, and event streaming (SSE).
- Support `--output json|table|yaml` for all agent commands (default: `table` for humans, `json` for scripts).
- Auth via existing config resolution (`--api-url`, `--token`, env vars `KOCAO_API_URL`, `KOCAO_TOKEN`).
- Follow the existing CLI patterns from `sessions` and `symphony` subcommands (flag-based, `flag.FlagSet`, `tabwriter` tables, JSON output).
- TDD: red-green-refactor for all commands with table-driven tests.

## Capabilities

### New Capabilities
- `cli-agent-management`: CLI interface for managing sandbox-agent sessions from the terminal.

### Modified Capabilities
- `control-plane-api`: no API changes needed — the endpoints already exist from 007-01. The CLI is a consumer only.

## Impact
- Affected code: `internal/controlplanecli/` (new files: `agent.go`, `agent_test.go`; modified: `root.go`, `client.go`).
- Dependencies: none new — uses existing `controlplanecli.Client` HTTP methods and control-plane API endpoints.
- Operations: no deployment changes — this is a CLI-only change shipped in the `kocao` binary.
- Breaking changes: none. Additive subcommand on an existing CLI binary.

## Scope

### In Scope
- `kocao agent list` — list agent sessions across workspace sessions
- `kocao agent start` — create a workspace session + harness run + agent session, wait for ready
- `kocao agent stop` — stop an agent session
- `kocao agent logs` — stream agent session events (SSE-based)
- `kocao agent exec` — send a prompt and display response events
- `kocao agent status` — detailed status of a single agent session
- Client methods on `controlplanecli.Client` for each API operation
- Table-driven unit tests for all commands
- Integration test helpers

### Out of Scope
- New API endpoints (007-01 already provides them)
- Web UI changes
- Symphony-specific agent workflows
- Agent binary installation or image changes
- Attach/WebSocket terminal workflows (existing `sessions attach` covers this)

## Success Criteria
- All six `agent` subcommands work end-to-end against a running control plane.
- `--output json` produces machine-parseable output for all commands.
- Test coverage >= 80% for new code.
- `kocao agent start` creates the full resource chain (workspace session -> harness run -> agent session) and waits for ready state.
- `kocao agent logs` streams events in real-time and exits cleanly on Ctrl+C.
- `kocao agent exec` sends a prompt and returns when the agent response is complete.
<!-- ITO:END -->
