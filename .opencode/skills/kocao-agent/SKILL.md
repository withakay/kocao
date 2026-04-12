---
name: kocao-agent
description: |
    Manage remote Kocao sandbox agents (workspace sessions) via the kocao CLI.
    Use when: starting an agent on the cluster, managing remote agents, running
    codex/claude/opencode remotely, listing running agents, sending a task to a
    remote agent, checking agent status, stopping a remote agent, streaming agent
    logs, or attaching to a running session.
---

Manage remote Kocao coding-agent sandbox sessions from an AI assistant context.

The `kocao` CLI communicates with the Kocao control-plane API to create, inspect,
and control workspace sessions that run coding agents in ephemeral Kubernetes pods.

## Prerequisites

- `kocao` binary in PATH (see `reference/agents.md` for install)
- `KOCAO_API_URL` and `KOCAO_TOKEN` environment variables set (or a config file at `~/.config/kocao/settings.json`)
- Cluster deployed and agent secrets seeded (`seed-agent-secrets`)

## Quick Reference

| Action | Script | CLI equivalent |
|---|---|---|
| List sessions | `scripts/agent-list.sh` | `kocao sessions ls --json` |
| Start/create session | `scripts/agent-start.sh` | (via control-plane API — see workflow) |
| Get session details | `scripts/agent-status.sh` | `kocao sessions status <id> --json` |
| Stream logs | `scripts/agent-logs.sh` | `kocao sessions logs <id> --json` |
| Execute/send task | `scripts/agent-exec.sh` | `kocao sessions attach <id> --driver` |
| Stop a session | `scripts/agent-stop.sh` | (via control-plane API — see workflow) |

## Workflows

### 1. List running agents

List all workspace sessions to see what agents are active.

```bash
scripts/agent-list.sh
```

Parse the JSON output to show session IDs, names, phases, and creation times.
Filter by phase `Running` to find active agents.

### 2. Check agent status

Get detailed status for a specific workspace session, including its current
harness run, pod name, and phase.

```bash
scripts/agent-status.sh <session-id>
```

The response includes:
- Session phase (Pending, Running, Succeeded, Failed)
- Active harness run ID and phase
- Pod name (for logs/debugging)

### 3. Stream agent logs

Fetch recent log output from an agent's pod.

```bash
# Last 200 lines (default)
scripts/agent-logs.sh <session-id>

# Last 50 lines
scripts/agent-logs.sh <session-id> --tail 50

# Follow mode (continuous polling)
scripts/agent-logs.sh <session-id> --follow
```

Note: `--follow` and `--json` cannot be combined. Use `--json` for structured
one-shot fetches; omit it when following.

### 4. Send a task to a remote agent (exec/attach)

Attach to a running session in driver mode to send input.

```bash
scripts/agent-exec.sh <session-id> --prompt "Review PR #42 and suggest improvements"
```

This attaches in driver mode, sends the prompt text, then detaches.
For interactive sessions, use `kocao sessions attach <id> --driver` directly.

**Important:** `attach` requires an interactive TTY for full bidirectional I/O.
The `agent-exec.sh` script provides a fire-and-forget prompt delivery mode
suitable for AI assistant use.

### 5. Start a remote agent

Start a new workspace session. This is typically done through the control-plane
API or UI. The script wraps the session creation flow.

```bash
scripts/agent-start.sh --repo https://github.com/org/repo --agent opencode
```

Supported `--agent` values: `opencode`, `codex`, `claude`, `pi`

The script will:
1. Call the control-plane API to create a workspace session
2. Wait for the session to reach `Running` phase
3. Output the session ID in JSON format

### 6. Stop a remote agent

Stop/terminate a workspace session.

```bash
scripts/agent-stop.sh <session-id>
```

This requests graceful termination of the session's harness run.

### 7. Multi-agent workflow

Start multiple agents and distribute tasks:

```bash
# Start agents
session1=$(scripts/agent-start.sh --repo https://github.com/org/repo --agent opencode --quiet)
session2=$(scripts/agent-start.sh --repo https://github.com/org/repo --agent codex --quiet)

# Send tasks
scripts/agent-exec.sh "$session1" --prompt "Implement the auth module"
scripts/agent-exec.sh "$session2" --prompt "Write tests for the API endpoints"

# Monitor
scripts/agent-status.sh "$session1"
scripts/agent-status.sh "$session2"

# Collect logs
scripts/agent-logs.sh "$session1" --tail 50
scripts/agent-logs.sh "$session2" --tail 50
```

## Symphony Projects

For managed multi-agent orchestration driven by GitHub Projects, use the
`kocao symphony` commands directly:

```bash
kocao symphony ls --json          # List symphony projects
kocao symphony get <name> --json  # Get project details
kocao symphony create --file <path> --json  # Create from spec
kocao symphony pause <name>       # Pause processing
kocao symphony resume <name>      # Resume processing
kocao symphony refresh <name>     # Force re-sync
```

## Usage Examples

**User:** "Start a codex agent on the cluster to work on this repo"
```
scripts/agent-start.sh --repo https://github.com/withakay/kocao --agent codex
```

**User:** "What agents are running?"
```
scripts/agent-list.sh
```

**User:** "Check on agent abc-123"
```
scripts/agent-status.sh abc-123
```

**User:** "Show me the logs for that agent"
```
scripts/agent-logs.sh abc-123 --tail 100
```

**User:** "Ask the remote agent to review this PR"
```
scripts/agent-exec.sh abc-123 --prompt "Review PR #42"
```

**User:** "Stop all agents"
```
# List all, then stop each
for id in $(scripts/agent-list.sh | jq -r '.workspaceSessions[] | select(.phase == "Running") | .id'); do
  scripts/agent-stop.sh "$id"
done
```

**User:** "Stream the agent's output"
```
scripts/agent-logs.sh abc-123 --follow
```

## Environment Variables

| Variable | Description | Required |
|---|---|---|
| `KOCAO_API_URL` | Control-plane API base URL | Yes (or config file) |
| `KOCAO_TOKEN` | Bearer token for authentication | Yes (or config file) |
| `KOCAO_TIMEOUT` | HTTP request timeout (e.g. `30s`) | No (default: 15s) |
| `KOCAO_VERBOSE` | Enable debug output (`true`/`false`) | No |

## Error Handling

All scripts exit with meaningful codes:
- `0` — success
- `1` — API/runtime error (check stderr for details)
- `2` — usage error (bad arguments, missing binary)

Common errors and fixes are documented in `reference/agents.md`.
