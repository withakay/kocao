---
name: kocao-agent
description: |
    Manage remote Kocao agent sessions from an AI assistant.
    Use when: listing running agents, starting a new remote agent, checking
    agent status, viewing agent logs, stopping an agent session, or sending a
    prompt to a running remote agent.
---

Manage remote Kocao agent sessions from an AI assistant context.

The `kocao agent` CLI communicates with the Kocao control-plane API to create,
inspect, and control agent sessions running inside ephemeral Kubernetes pods.
This skill wraps those commands in small shell scripts with consistent defaults
and machine-readable output.

The important lifecycle contract is:

- the CLI uses `runId` as the canonical handle for follow-up actions
- create and status expose stable public `runId` / `sessionId` / `phase` fields, and stop returns the same identifiers under its nested `session` object
- non-ready sessions may include `diagnostic.class`, `diagnostic.summary`, and `diagnostic.detail`
- repeated create and stop calls are safe and should be treated as idempotent lifecycle reconciliation, not duplicate work

## Prerequisites

- `kocao` binary in `PATH`
- `KOCAO_API_URL` and `KOCAO_TOKEN` set when the CLI is talking to a local or
  port-forwarded control-plane
- `jq` installed for `agent-start.sh --quiet`
- Cluster deployed and agent secrets seeded

## Quick Reference

| Action | Script | Backing capability |
|---|---|---|
| List agent sessions | `scripts/agent-list.sh` | `kocao agent list --output json` |
| Start agent session | `scripts/agent-start.sh` | `kocao agent start ...` |
| Get session details | `scripts/agent-status.sh` | `kocao agent status <run-id> --output json` |
| Fetch events/logs | `scripts/agent-logs.sh` | `kocao agent logs <run-id> ...` |
| Send a prompt | `scripts/agent-exec.sh` | `kocao agent exec <run-id> --prompt ...` |
| Stop a session | `scripts/agent-stop.sh` | `kocao agent stop <run-id> --json` |

## Script Conventions

- JSON is the default output unless `--no-json` says otherwise
- usage problems exit with code `2`
- runtime/API failures exit with code `1`
- all wrappers rely on `kocao agent`, not ad-hoc REST calls
- lifecycle phases are `Provisioning`, `Ready`, `Running`, `Stopping`, `Completed`, and `Failed`
- blocker classes currently include `provisioning`, `image-pull`, `sandbox-agent-readiness`, `auth`, `repo-access`, and `network`

## Workflows

> Script paths below are relative to `.opencode/skills/kocao-agent/`.

### 1. List running agents

```bash
scripts/agent-list.sh
scripts/agent-list.sh --workspace <workspace-id>
```

Filter JSON output with `.[] | select(.phase == "Ready" or .phase == "Running")`
to find active agents.

The table view includes a `BLOCKER` column so assistants can spot unhealthy
sessions without opening each run individually.

### 2. Start a remote agent

```bash
scripts/agent-start.sh --repo https://github.com/org/repo --agent codex --timeout 5m
```

Useful flags:
- `--workspace` to reuse an existing workspace session
- `--revision` to select a branch, tag, or commit
- `--image` and `--image-pull-secret` for remote clusters
- `--egress-mode full` when the harness needs unrestricted outbound access
- `--quiet` to print only the run ID

The wrapper waits until the lifecycle reaches `Ready`. If the wait times out,
use `scripts/agent-status.sh <run-id>` to surface the current blocker.

### 3. Check agent status

```bash
scripts/agent-status.sh <run-id>
scripts/agent-status.sh <run-id> --no-json
```

The JSON status payload includes run ID, session ID, runtime, agent, phase,
workspace session ID, created time, and an optional `diagnostic` object.

When `diagnostic` is present, assistants should surface the blocker class and
summary directly instead of paraphrasing away the failure mode.

### 4. Send a task to a remote agent

```bash
scripts/agent-exec.sh <run-id> --prompt "Review PR #42 and summarize the risks"
scripts/agent-exec.sh <run-id> "Review PR #42 and summarize the risks"
```

Use `--no-json` for the CLI’s default formatted event rendering.

### 5. Stream agent logs

```bash
scripts/agent-logs.sh <run-id> --tail 50
scripts/agent-logs.sh <run-id> --follow --no-json
```

### 6. Stop a remote agent

```bash
scripts/agent-stop.sh <run-id>
scripts/agent-stop.sh <run-id> --no-json
```

`agent-stop.sh` returns the terminal session view from the CLI. Re-running stop
after reconnects is safe and is the preferred way to confirm a terminal phase.

### 7. Multi-agent workflow

```bash
run1=$(scripts/agent-start.sh --repo https://github.com/org/repo --agent opencode --quiet)
run2=$(scripts/agent-start.sh --repo https://github.com/org/repo --agent codex --quiet)

scripts/agent-exec.sh "$run1" --prompt "Implement the auth module"
scripts/agent-exec.sh "$run2" --prompt "Write tests for the API endpoints"

scripts/agent-status.sh "$run1"
scripts/agent-status.sh "$run2"

scripts/agent-logs.sh "$run1" --tail 50
scripts/agent-logs.sh "$run2" --tail 50
```

## Example Triggers

Use this skill for prompts like:

- “What remote agents are running right now?”
- “Start a codex agent for this repo and wait until it is ready.”
- “Check the status of run `hr-123` and tell me what blocker is preventing readiness.”
- “Show the last 100 events from that agent session.”
- “Stop the finished agent run.”
- “Send this prompt to the remote agent.”

## Environment Variables

| Variable | Description | Required |
|---|---|---|
| `KOCAO_API_URL` | Control-plane API base URL | Yes when not using built-in CLI config |
| `KOCAO_TOKEN` | Bearer token for authentication | Yes when not using built-in CLI config |
| `KOCAO_TIMEOUT` | CLI HTTP timeout | No |
| `KOCAO_VERBOSE` | Enable CLI debug output | No |

## Error Handling

- `0` — success
- `1` — API or runtime error
- `2` — usage error or missing dependency

See `reference/agents.md` for setup notes and troubleshooting.
