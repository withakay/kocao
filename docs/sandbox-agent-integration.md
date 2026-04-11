# Sandbox Agent Integration Guide

This document describes Kocao's sandbox-agent-backed run flow.

## What changed

Kocao now treats `sandbox-agent` as the universal in-harness API for supported coding agents:

- `opencode`
- `claude`
- `codex`
- `pi`

Kocao still owns the public API, authn/authz, audit trail, Kubernetes orchestration, and UI. The sandbox-agent server runs inside each Harness Run pod and is not exposed directly to browsers.

## Runtime contract

The harness image now:

- installs `sandbox-agent`
- installs supported agent CLIs (`opencode`, `claude`, `codex`, `pi`)
- starts `sandbox-agent server` automatically when a Harness Run is configured with an `agentSession`
- validates the sandbox-agent health endpoint and supported agent catalog in `build/harness/smoke.sh`

## API flow

### 1. Start a Harness Run with an agent

`POST /api/v1/workspace-sessions/{workspaceSessionID}/harness-runs`

```json
{
  "repoURL": "https://github.com/withakay/kocao",
  "repoRevision": "main",
  "image": "kocao/harness-runtime:dev",
  "agentSession": {
    "agent": "codex"
  }
}
```

### 2. Create or resume the sandbox-backed session

`POST /api/v1/harness-runs/{harnessRunID}/agent-session`

Returns run-scoped session metadata including runtime, selected agent, session id, phase, and last seen sequence.

### 3. Send a prompt

`POST /api/v1/harness-runs/{harnessRunID}/agent-session/prompt`

```json
{
  "prompt": "Summarize this repository and propose the next refactor."
}
```

### 4. Read transcript events

Replay-safe polling:

`GET /api/v1/harness-runs/{harnessRunID}/agent-session/events?offset=0&limit=200`

Live stream:

`GET /api/v1/harness-runs/{harnessRunID}/agent-session/events/stream?offset=0`

### 5. Stop the agent session

`POST /api/v1/harness-runs/{harnessRunID}/agent-session/stop`

## UI flow

### Workspace Session page

The **Start Harness Run** form now includes an **Agent** selector. Choose one of:

- OpenCode
- Claude
- Codex
- Pi

Starting the run stores the requested sandbox-backed agent session contract on the Harness Run.

### Run detail page

If a run was created with an agent selection, the **Run Detail** view now shows an **Agent Session** section with:

- runtime
- selected agent
- session id
- current phase
- Start / Resume control
- Stop control
- prompt composer
- transcript/event feed

Reloading the page re-fetches the persisted event history through Kocao's API.

## Resume behavior

When a Harness Run is resumed through the existing run lifecycle, Kocao preserves agent-session history by loading persisted metadata/events from the prior run when the new run is labeled as resumed from it.

This means the replacement run can:

- display prior transcript history
- show the prior session id metadata
- continue the user-visible workflow from the same Workspace Session context

## Security notes

- Browsers only talk to Kocao endpoints.
- Kocao proxies sandbox-agent traffic internally.
- Provider credentials remain pod-scoped.
- Persisted event envelopes redact secret-shaped values before storage.

## Validation commands

Use these commands to validate the sandbox-agent integration path:

```bash
go test ./internal/controlplaneapi/... ./internal/operator/... ./internal/harnessruntime/...
pnpm -C web test
pnpm -C web lint
kubectl kustomize deploy/base >/dev/null
make harness-smoke
```

## Acceptance checklist

The intended happy path is:

1. Run Kocao in Kubernetes.
2. Create a Workspace Session.
3. Start a Harness Run from the UI or API with one of `opencode`, `claude`, `codex`, or `pi`.
4. Create or resume the run's sandbox-backed agent session.
5. Send prompts and observe transcript events.
6. Stop or resume the session/run and retain visible history.
