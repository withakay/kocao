# Tasks for: 003-04_add-scalar-and-cluster-ui-serving

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for all task status updates.
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`
- **Docs route**: lock to `/scalar`.

```bash
ito tasks status 003-04_add-scalar-and-cluster-ui-serving
ito tasks next 003-04_add-scalar-and-cluster-ui-serving
ito tasks start 003-04_add-scalar-and-cluster-ui-serving 1.1
ito tasks complete 003-04_add-scalar-and-cluster-ui-serving 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Add same-pod Caddy container and edge service wiring

- **Files**: `deploy/base/api-deployment.yaml`, `deploy/base/api-service.yaml`, `deploy/base/kustomization.yaml`
- **Dependencies**: None
- **Action**: Add Caddy container, ports, probes, and shared volumes so the API pod has a web edge and the Service points to Caddy as the user-facing port.
- **Verify**: `kustomize build deploy/base`
- **Done When**: Base manifests render with API+Caddy in one pod and service traffic enters through the Caddy edge port.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 1.2: Add concrete Caddy routing config

- **Files**: `deploy/base/caddy/Caddyfile`, `deploy/base/kustomization.yaml`
- **Dependencies**: Task 1.1
- **Action**: Define explicit route table for `/`, static assets, `/scalar`, `/openapi.json`, `/api/v1/*`, `/healthz`, `/readyz`, and attach websocket upgrade forwarding.
- **Verify**: `docker run --rm -v "$PWD/deploy/base/caddy/Caddyfile:/etc/caddy/Caddyfile" caddy:2 caddy validate --config /etc/caddy/Caddyfile`
- **Done When**: Caddyfile validates and includes explicit websocket proxy behavior.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Add Scalar entrypoint at `/scalar` wired to live OpenAPI

- **Files**: `deploy/base/caddy/scalar.html`, `deploy/base/caddy/Caddyfile`, `internal/controlplaneapi/openapi.go`, `docs/planning/coding-agent-orchestrator-architecture.md`
- **Dependencies**: None
- **Action**: Add Scalar static page and route wiring so Scalar fetches live `/openapi.json` from the same deployed endpoint.
- **Verify**: `go test ./internal/controlplaneapi/...`
- **Done When**: `/scalar` loads Scalar and renders the live API schema.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 2.2: Add dev-kind overlay exposure and smoke runbook

- **Files**: `deploy/overlays/dev-kind/kustomization.yaml`, `deploy/overlays/dev-kind/patch-use-configmap.yaml`, `docs/planning/cluster-ui-serving-dev-kind-smoke.md`
- **Dependencies**: Task 2.1
- **Action**: Update dev-kind overlay to expose Caddy edge and write explicit smoke checks for UI, `/scalar`, `/api/v1/*`, and attach websocket.
- **Verify**: `kustomize build deploy/overlays/dev-kind`
- **Done When**: Dev-kind overlay renders and smoke runbook is executable end-to-end.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Add plan-only optional Tailscale overlay artifacts

- **Files**: `deploy/overlays/dev-kind/tailscale-patch.yaml`, `deploy/overlays/dev-kind/kustomization.yaml`, `docs/planning/cluster-ui-serving-tailscale-plan.md`, `docs/security/posture.md`
- **Dependencies**: None
- **Action**: Add disabled-by-default Tailscale sidecar overlay plan, operator enablement steps, and security posture notes.
- **Verify**: `kustomize build deploy/overlays/dev-kind`
- **Done When**: Optional Tailscale path is documented, renderable, and clearly opt-in.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 3.2: Add test coverage and final verification script

- **Files**: `internal/controlplaneapi/api_test.go`, `internal/controlplaneapi/attach_test.go`, `web/src/ui/workflow.test.tsx`, `docs/planning/cluster-ui-serving-dev-kind-smoke.md`
- **Dependencies**: Task 3.1
- **Action**: Add/adjust tests for edge route behavior, Scalar live spec loading, and attach websocket proxy path; finalize verification steps in docs.
- **Verify**: `go test ./... && pnpm -C web test`
- **Done When**: Tests pass and docs include a single reproducible verification flow.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Checkpoint

### Checkpoint: Approve optional Tailscale rollout plan

- **Type**: checkpoint (requires human approval)
- **Dependencies**: All Wave 3 tasks
- **Action**: Confirm optional Tailscale sidecar overlay scope before enabling outside dev-kind.
- **Done When**: Maintainer approves the rollout checklist.
- **Updated At**: 2026-02-24
- **Status**: [-] shelved
