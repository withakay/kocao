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

The planned harness profile split keeps that contract intact across every profile. `base`, `go`, `web`, and `full` may differ in workload runtimes, but each profile must still ship the same sandbox-agent entrypoint, health endpoint, and supported agent catalog.

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

The same status payload now includes `startupMetrics` for the harness run:

- `imagePullDurationMs`
- `timeToReadyMs`
- `timeToFirstPromptMs`

This gives operators a stable place to compare the `base`, `go`, `web`, and `full` profiles without scraping ad hoc logs.

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

## Kocao Sidecar and Token Sync

Each Harness Run pod includes a `kocao-sidecar` container alongside the main `harness` container. The sidecar handles:

- **Token sync**: polls watched auth files inside the harness container (`/home/kocao/.local/share/opencode/auth.json`, `/home/kocao/.codex/auth.json`) and patches the `kocao-agent-oauth` Secret with any changes. This allows agents that perform interactive OAuth flows to persist their tokens back to the cluster.

An `auth-seed` init container runs before the main containers to bootstrap initial auth tokens from the `kocao-agent-oauth` Secret into the shared `agent-auth-live` emptyDir volume.

### Pod container layout

| Container | Role |
|-----------|------|
| `auth-seed` (init) | Copies initial OAuth tokens from Secret to shared volume |
| `harness` | Runs the agent runtime (sandbox-agent, git clone, agent CLI) |
| `kocao-sidecar` | Watches for auth file changes and syncs them back to the Secret |

### Seeding real OAuth tokens for local development

```bash
make seed-agent-secrets
```

This copies `~/.local/share/opencode/auth.json` and `~/.codex/auth.json` from the local machine into the `kocao-agent-oauth` Secret in the `kocao-system` namespace. The auth-seed init container then makes these available to the harness container at pod startup.

## Security notes

- Browsers only talk to Kocao endpoints.
- Kocao proxies sandbox-agent traffic internally.
- Provider credentials remain pod-scoped.
- Persisted event envelopes redact secret-shaped values before storage.
- The kocao-sidecar has RBAC access only to patch the `kocao-agent-oauth` Secret in its own namespace.

## Validation commands

Use these commands to validate the sandbox-agent integration path:

```bash
go test ./internal/controlplaneapi/... ./internal/operator/... ./internal/harnessruntime/...
pnpm -C web test
pnpm -C web lint
kubectl kustomize deploy/base >/dev/null
make harness-smoke
```

For profile startup measurements during demos or dev-cluster tuning:

```bash
kocao agent start --repo https://github.com/withakay/kocao --agent codex --image-profile web --output json
kocao agent status <run-id> --output json | jq '.startupMetrics'
kocao agent exec <run-id> --prompt "Summarize the repo" --output json >/dev/null
kocao agent status <run-id> --output json | jq '.startupMetrics'
```

## Local Kind Setup

```bash
kubectl config current-context  # Must be kind-kocao-dev
make images                     # Build all images (api, operator, web, harness, sidecar)
make kind-load-images           # Load into Kind
make kind-prepull-harness-profiles  # Load base/go/web/full harness profiles before live runs
make deploy                     # Apply kustomize overlay
make deploy-wait                # Wait for rollout
make seed-agent-secrets         # Copy local OAuth tokens
```

For registry-backed dev clusters, pre-pull the common harness profiles before a demo so the first profiled run does not pay the full image download cost:

```bash
HARNESS_IMAGE=ghcr.io/withakay/kocao/harness-runtime IMAGE_TAG=dev-microk8s-amd64fix \
  IMAGE_PULL_SECRETS=ghcr-pull \
  make registry-prepull-harness-profiles
```

The script uses a short-lived DaemonSet for MicroK8s and other registry-backed dev clusters, and cleans it up on exit unless `KEEP_PREPULL_DAEMONSET=1` is set. Use `make microk8s-prepull-harness-profiles` as a convenience wrapper for the default `microk8s` kube context, set `PREPULL_CONTEXT` when targeting another registry-backed cluster, or use `kind` mode to load locally built images straight into Kind.

**Important**: When creating Harness Runs in Kind, use `"egressMode":"full"` to allow the pod to reach external git hosts and API endpoints. The default `restricted` egress mode only allows DNS.

## Acceptance checklist

The intended happy path is:

1. Run Kocao in Kubernetes.
2. Create a Workspace Session.
3. Start a Harness Run from the UI or API with one of `opencode`, `claude`, `codex`, or `pi`.
4. Verify the pod has three containers: `auth-seed` (init), `harness`, `kocao-sidecar`.
5. Create or resume the run's sandbox-backed agent session.
6. Send prompts and observe transcript events.
7. Stop or resume the session/run and retain visible history.
8. Verify the kocao-sidecar syncs auth tokens back to the Secret.
