---
name: kocao-agent
description: |
    Manage remote Kocao workspace sessions from an AI assistant.
    Use when: listing remote sessions, creating a new session, checking session
    status, viewing logs, stopping a session, or sending a prompt through the
    experimental exec endpoint when the control-plane supports it.
---

Manage remote Kocao workspace sessions from an AI assistant context.

The `kocao` CLI communicates with the Kocao control-plane API to create,
inspect, and control workspace sessions that run inside ephemeral Kubernetes
pods. This skill wraps the common session-management actions in small shell
scripts with consistent output and error handling.

## Prerequisites

- `kocao` binary in `PATH` for list/status/logs workflows
- `curl` and `jq` for the API-backed helper scripts
- `KOCAO_API_URL` and `KOCAO_TOKEN` environment variables set, or a config file at `~/.config/kocao/settings.json`
- Cluster deployed and agent secrets seeded (`seed-agent-secrets`)

## Quick Reference

| Action | Script | Backing capability |
|---|---|---|
| List sessions | `scripts/agent-list.sh` | `kocao sessions ls --json` |
| Create session | `scripts/agent-start.sh` | `POST /api/v1/workspace-sessions` |
| Get session details | `scripts/agent-status.sh` | `kocao sessions status <id> --json` |
| Fetch logs | `scripts/agent-logs.sh` | `kocao sessions logs <id> ...` |
| Send a prompt | `scripts/agent-exec.sh` | Experimental `POST /api/v1/workspace-sessions/<id>/exec` |
| Stop a session | `scripts/agent-stop.sh` | `DELETE /api/v1/workspace-sessions/<id>` |

## Script Conventions

All scripts now follow the same shape:

- `--help` prints a concise usage block
- usage problems exit with code `2`
- API/runtime failures exit with code `1`
- JSON is the default output unless `--no-json` says otherwise
- API scripts reuse `scripts/common.sh` for dependency checks, URL encoding, and HTTP error formatting

## Workflows

### 1. List running agents

List all workspace sessions to see what is active.

```bash
scripts/agent-list.sh
```

Filter the JSON output by `.workspaceSessions[] | select(.phase == "Running")`
to find active sessions.

### 2. Check agent status

Get the current session phase plus the latest harness-run details.

```bash
scripts/agent-status.sh <session-id>
```

The JSON response includes:
- session ID, display name, phase, and repo URL
- current harness run ID and phase (when present)
- pod name for debugging/log lookup

### 3. Stream agent logs

Fetch recent log output from the active harness pod.

```bash
# JSON one-shot fetch
scripts/agent-logs.sh <session-id> --tail 200

# Plain text
scripts/agent-logs.sh <session-id> --tail 50 --no-json

# Follow mode
scripts/agent-logs.sh <session-id> --follow --no-json
```

`--follow` and `--json` are mutually exclusive.

### 4. Create a remote session

Create a new workspace session for a repository.

```bash
scripts/agent-start.sh --repo https://github.com/org/repo --agent codex
```

Important details:
- `--repo` is required
- `--agent` currently acts as a **display-name label only** (`opencode`, `codex`, `claude`, `pi`)
- the control-plane API creates a generic session; it does **not** switch images or runtime behavior based on `--agent`
- by default the script waits for `Running` and then prints the final status JSON
- `--quiet` prints just the session ID for shell pipelines

### 5. Send a task to a remote session

Send a prompt through the control-plane exec endpoint when that endpoint exists.

```bash
scripts/agent-exec.sh <session-id> --prompt "Review PR #42 and summarize the risks"
```

If the API returns HTTP 404, the control-plane does not expose the exec
endpoint yet. In that case, use an interactive terminal instead:

```bash
kocao sessions attach <session-id> --driver
```

### 6. Stop a remote session

Stop/terminate a workspace session.

```bash
scripts/agent-stop.sh <session-id>
```

### 7. Multi-session workflow

```bash
session1=$(scripts/agent-start.sh --repo https://github.com/org/repo --agent opencode --quiet)
session2=$(scripts/agent-start.sh --repo https://github.com/org/repo --agent codex --quiet)

scripts/agent-exec.sh "$session1" --prompt "Implement the auth module"
scripts/agent-exec.sh "$session2" --prompt "Write tests for the API endpoints"

scripts/agent-status.sh "$session1"
scripts/agent-status.sh "$session2"

scripts/agent-logs.sh "$session1" --tail 50
scripts/agent-logs.sh "$session2" --tail 50
```

## Symphony Projects

For managed multi-agent orchestration driven by GitHub Projects, use the
`kocao symphony` commands directly:

```bash
kocao symphony ls --json
kocao symphony get <name> --json
kocao symphony create --file <path> --json
kocao symphony pause <name>
kocao symphony resume <name>
kocao symphony refresh <name>
```

## Example Triggers

Use this skill for prompts like:

- “What Kocao sessions are running right now?”
- “Start a remote session for this repo and wait until it is running.”
- “Check the status of session `ws-123`.”
- “Show the last 100 lines from that agent session.”
- “Stop the finished session.”
- “Send this prompt to the remote session if exec is supported.”

## Environment Variables

| Variable | Description | Required |
|---|---|---|
| `KOCAO_API_URL` | Control-plane API base URL | Yes for API-backed scripts unless config is used |
| `KOCAO_TOKEN` | Bearer token for authentication | Yes for API-backed scripts unless config is used |
| `KOCAO_TIMEOUT` | HTTP request timeout for the CLI | No |
| `KOCAO_VERBOSE` | Enable CLI debug output | No |

## Error Handling

- `0` — success
- `1` — API or runtime error
- `2` — usage error or missing dependency

See `reference/agents.md` for setup notes and troubleshooting.
