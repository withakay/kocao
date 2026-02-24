## Why
The web UI currently depends on local/dev serving assumptions and does not provide an in-cluster API exploration surface. We need a cluster-native serving model so operators can access both the workflow UI and live OpenAPI docs from the same deployed control-plane pod.

## What Changes
- Add a cluster serving topology where Caddy runs in the same pod as the control-plane API and serves the workflow UI.
- Add Scalar API reference UI at `/scalar`, served by Caddy from the same pod, loading the live `/openapi.json` spec.
- Define unified edge routing so UI assets and docs are served by Caddy while API and websocket paths are proxied to the local control-plane API container.
- Define a dev-kind-first rollout plan for manifests and operational verification.
- Add plan-only Tailscale integration design (optional entrypoint/overlay), without requiring immediate production rollout.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `workflow-ui-github`: extend workflow console requirements to include in-cluster serving, Scalar API docs, and same-pod routing behavior.

## Impact
- Affects deployment manifests and runtime topology for the control-plane pod (multi-container pod with Caddy + API).
- Adds/updates Caddy configuration, web asset serving paths, and Scalar embedding configuration.
- Adds documentation and runbook guidance for dev-kind rollout and optional Tailscale integration path.
