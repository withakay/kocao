# Kocao Agent CLI Diagnostics Demo

*2026-04-14T09:30:00Z — blocker-focused outputs aligned with the current lifecycle contract*

This document replaces the earlier "resolved gaps" notes. The CLI and API now
share a stable lifecycle contract, so the operator-facing demo value is in the
diagnostics surfaced for non-ready sessions.

See `agent-cli-live-demo.md` for the happy path. Use the examples below when
triaging provisioning and runtime blockers.

## 1. Unschedulable pod

```bash
kocao agent status run-unschedulable --output json
```

```output
{
  "runId": "run-unschedulable",
  "sessionId": "ses_waiting123",
  "agent": "codex",
  "runtime": "sandbox-agent",
  "phase": "Provisioning",
  "diagnostic": {
    "class": "provisioning",
    "summary": "Pod scheduling is blocking session readiness.",
    "detail": "Unschedulable: 0/3 nodes are available: insufficient cpu."
  }
}
```

## 2. Image pull failure

```bash
kocao agent status run-image-pull --output json
```

```output
{
  "runId": "run-image-pull",
  "sessionId": "ses_imagepull123",
  "agent": "codex",
  "runtime": "sandbox-agent",
  "phase": "Provisioning",
  "diagnostic": {
    "class": "image-pull",
    "summary": "Image pull is blocking session readiness.",
    "detail": "container \"workspace\" waiting: ImagePullBackOff: Back-off pulling image \"ghcr.io/private/image:missing\""
  }
}
```

## 3. Sandbox-agent readiness delay

```bash
kocao agent status run-sandbox-readiness
```

```output
  Run ID:     run-sandbox-readiness
  Session ID: ses_readywait123
  Name:       demo-sbox-123
  Runtime:    sandbox-agent
  Agent:      codex
  Phase:      Provisioning
  Workspace:  ws-demo123
  Created:    2026-04-14T09:30:00Z
  Blocker:    sandbox-agent-readiness
  Summary:    Sandbox-agent is not ready yet.
  Detail:     Pod "demo-sbox-123" is running, but the sandbox-agent health path has not produced a ready session.
```

## 4. Credential, repo, and network blockers

Representative blocker classes from real classification logic:

- `auth`: missing secret, invalid token, or other credential bootstrap failure
- `repo-access`: repository checkout or Git authentication failure
- `network`: DNS, egress, TLS, or connection timeout failure

## 5. Fleet visibility with `agent list`

```bash
kocao agent list
```

```output
SESSION ID     RUN                  AGENT     PHASE         BLOCKER                  WORKSPACE    CREATED
ses_ok123      run-ready            opencode  Ready         -                        ws-alpha     2026-04-14T09:25:00Z
ses_wait123    run-image-pull       codex     Provisioning  image-pull               ws-alpha     2026-04-14T09:26:00Z
ses_wait456    run-sandbox-check    claude    Provisioning  sandbox-agent-readiness  ws-beta      2026-04-14T09:27:00Z
```

## 6. Stop remains safe after retries or reconnects

```bash
kocao agent stop run-ready --json
```

```output
{
  "status": "stopped",
  "session": {
    "runId": "run-ready",
    "sessionId": "ses_ok123",
    "phase": "Completed"
  }
}
```

Running the same stop command again returns the same terminal lifecycle view,
which is the intended idempotent behavior.
