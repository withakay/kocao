# Agent CLI Lifecycle Demo

This document summarizes the production lifecycle contract for `kocao agent`.
For concrete outputs, use `demos/agent-cli-live-demo.md` and
`demos/agent-cli-errors-demo.md`.

## Prerequisites

1. Deploy Kocao to a reachable cluster.
2. Seed the agent credentials and image pull secrets required by the harness image.
3. Export `KOCAO_API_URL` and `KOCAO_TOKEN`, or configure `~/.config/kocao/settings.json`.

## Lifecycle model

Agent sessions move through explicit durable phases:

- `Provisioning`: the harness pod or sandbox-agent is not ready yet.
- `Ready`: the session exists and can accept prompts.
- `Running`: prompt execution is in flight.
- `Stopping`: shutdown has started but is not yet terminal.
- `Completed`: the session stopped cleanly and remains terminal after restart.
- `Failed`: the session hit a hard lifecycle failure and remains terminal after restart.

## Public contract

Create, status, and stop all center on the same public identifiers:

- `runId`: canonical handle for CLI follow-up operations.
- `sessionId`: sandbox-agent session identity once initialized.
- `phase`: lifecycle phase from the explicit state machine.
- `agent`, `runtime`, `workspaceSessionId`, `displayName`, `createdAt`.
- `startupMetrics`: per-run startup timings for `imagePullDurationMs`, `timeToReadyMs`, and `timeToFirstPromptMs`.
- `diagnostic`: optional blocker details when the session is not ready.

When `diagnostic` is present it uses:

- `class`: blocker class such as `provisioning`, `image-pull`, `sandbox-agent-readiness`, `auth`, `repo-access`, or `network`.
- `summary`: short operator-facing explanation.
- `detail`: specific Kubernetes or runtime detail when available.

## Start an agent

```bash
kocao agent start --repo https://github.com/withakay/kocao --agent codex --timeout 5m
kocao agent start --repo https://github.com/withakay/kocao --agent codex --output json
```

`start` creates or reuses a workspace session, creates the harness run, then polls the idempotent session create/status path until the lifecycle reaches `Ready` or the timeout expires.

If the timeout expires, the CLI prints the last known `phase` and `sessionId` so the operator can continue with `status`, `logs`, or `stop`.

## Check status

```bash
kocao agent status <run-id>
kocao agent status <run-id> --output json
```

Representative provisioning response:

```json
{
  "sessionId": "ses_demo123",
  "runId": "run-demo123",
  "runtime": "sandbox-agent",
  "agent": "codex",
  "phase": "Provisioning",
  "workspaceSessionId": "ws-demo123",
  "createdAt": "2026-04-14T09:30:00Z",
  "startupMetrics": {
    "imagePullDurationMs": 12000,
    "timeToReadyMs": 18500
  },
  "diagnostic": {
    "class": "image-pull",
    "summary": "Image pull is blocking session readiness.",
    "detail": "container \"workspace\" waiting: ImagePullBackOff: Back-off pulling image \"ghcr.io/private/image:missing\""
  }
}
```

Table output includes the same lifecycle summary plus startup timing lines and blocker lines when they are populated.

## Profile startup measurement

The fastest demo loop for comparing `base`, `go`, `web`, and `full` is:

```bash
kocao agent start --repo https://github.com/withakay/kocao --agent codex --image-profile web --output json
kocao agent status <run-id> --output json | jq '.startupMetrics'
kocao agent exec <run-id> --prompt "Summarize the repo" --output json >/dev/null
kocao agent status <run-id> --output json | jq '.startupMetrics'
```

Use the first status call to capture image pull and time-to-ready. Use the second status call after the first prompt to capture `timeToFirstPromptMs`.

## List active agents

```bash
kocao agent list
kocao agent list --workspace <workspace-id> --output json
kocao agent list --output yaml
```

The table view includes a `BLOCKER` column so non-ready sessions can be triaged without opening each run individually.

## Send a prompt

```bash
kocao agent exec <run-id> --prompt "What files are in the repo?"
kocao agent exec <run-id> "What files are in the repo?"
```

Use `--output json` to inspect the full event payloads returned by the prompt API.

## Stream logs

```bash
kocao agent logs <run-id> --tail 50
kocao agent logs <run-id> --follow --output json
```

Logs and prompt results use the same event shape: `seq`, `timestamp`, and `data`.

## Stop the session

```bash
kocao agent stop <run-id>
kocao agent stop <run-id> --json
```

`stop --json` returns a stable wrapper:

```json
{
  "status": "stopped",
  "session": {
    "runId": "run-demo123",
    "sessionId": "ses_demo123",
    "phase": "Completed"
  }
}
```

Repeated stop requests are safe. If the session is already terminal, the CLI returns the current terminal state instead of creating a duplicate shutdown flow.

## Operator notes

- Prefer `kocao agent status <run-id> --output json` before dropping into Kubernetes. The blocker class usually tells you whether the issue is scheduling, image pull, sandbox-agent readiness, auth, repo access, or network reachability.
- Use `kocao agent list` to spot sessions stuck in `Provisioning` or `Stopping` across all workspaces.
- Use `demos/agent-cli-errors-demo.md` when you want representative blocker outputs instead of historical gap notes.
