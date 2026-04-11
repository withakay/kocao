<!-- ITO:START -->
# Tasks for: 007-01_integrate-sandbox-agent-universal-api

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 007-01_integrate-sandbox-agent-universal-api
ito tasks next 007-01_integrate-sandbox-agent-universal-api
ito tasks start 007-01_integrate-sandbox-agent-universal-api 1.1
ito tasks complete 007-01_integrate-sandbox-agent-universal-api 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Define sandbox-backed run/session contract

- **Files**: `internal/operator/api/v1alpha1/*`, `internal/controlplaneapi/*`, `.ito/specs/*`
- **Dependencies**: None
- **Action**: Extend run and API contract types to represent sandbox-backed session creation, supported agent selection, normalized lifecycle, and run/session identity mapping.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/operator/api/v1alpha1/...`
- **Done When**: Kocao contracts can describe a sandbox-backed session for `opencode`, `claude`, `codex`, and `pi` without provider-specific public endpoints.
- **Requirements**: sandbox-agent-runtime:unified-session-contract, sandbox-agent-runtime:supported-agent-catalog, sandbox-agent-runtime:normalized-agent-lifecycle, control-plane-api:sandbox-agent-session-management-endpoints, control-plane-api:sandbox-agent-lifecycle-control-api
- **Updated At**: 2026-04-11
- **Status**: [x] complete

### Task 1.2: Add sandbox-agent to harness runtime

- **Files**: `build/Dockerfile.harness`, `build/harness/*`, `internal/harnessruntime/*`, `internal/operator/controllers/pod.go`
- **Dependencies**: Task 1.1
- **Action**: Pin and install `sandbox-agent`, ensure supported agent dependencies remain available, update pod/runtime supervision so harness runs expose a healthy in-pod sandbox-agent server, and add contract validation against the pinned sandbox-agent API surface.
- **Verify**: `make harness-smoke && go test ./internal/harnessruntime/... ./internal/operator/controllers/...`
- **Done When**: A harness pod can start sandbox-agent reliably, Kocao can target it as the single in-pod agent API, and contract drift in the pinned sandbox-agent dependency is caught by automated validation.
- **Requirements**: harness-runtime:sandbox-agent-server-present, harness-runtime:supported-agent-dependencies-available, harness-runtime:sandbox-agent-version-and-api-contract-are-verified, harness-runtime:reproducible-harness-runtime-image
- **Updated At**: 2026-04-11
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Implement mediated sandbox-agent API flows

- **Files**: `internal/controlplaneapi/*`, `internal/controlplanecli/*`, `internal/operator/controllers/*`, `internal/operator/api/v1alpha1/*`
- **Dependencies**: None
- **Action**: Add Kocao endpoints and internal services for session create/inspect, message send, event replay/streaming, readiness, stop, and resume using sandbox-agent as the backend.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/operator/controllers/...`
- **Done When**: Authorized API clients can launch, interact with, and manage sandbox-backed sessions without direct access to sandbox-agent.
- **Requirements**: control-plane-api:sandbox-agent-session-management-endpoints, control-plane-api:sandbox-agent-messaging-and-event-replay-api, control-plane-api:sandbox-agent-lifecycle-control-api, security-posture:sandbox-agent-access-is-control-plane-mediated
- **Updated At**: 2026-04-11
- **Status**: [x] complete

### Task 2.2: Persist transcript history and resume metadata

- **Files**: `internal/controlplaneapi/*`, `internal/operator/controllers/*`, persistence-related stores/tests
- **Dependencies**: Task 2.1
- **Action**: Persist normalized sandbox-agent events and session metadata, reload by offset, and bind resume flows to Workspace Session durability.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/operator/controllers/... ./internal/auditlog/...`
- **Done When**: Transcript history survives client reconnects and resumed runs preserve prior session context.
- **Requirements**: sandbox-agent-runtime:ordered-prompt-and-event-interaction, session-durability:sandbox-agent-event-history-is-durable, session-durability:resumed-runs-preserve-agent-session-context, session-durability:workspace-session-is-the-durable-workspace-anchor, security-posture:provider-credentials-remain-pod-scoped, security-posture:workflow-and-agent-secrets-stay-bounded
- **Updated At**: 2026-04-11
- **Status**: [x] complete

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Ship UI launch and interaction flow

- **Files**: `web/src/ui/pages/*`, `web/src/ui/components/*`, `web/src/ui/lib/api.ts`, `web/src/ui/workflow.test.tsx`
- **Dependencies**: None
- **Action**: Add agent selection, live transcript/event rendering, prompt input, and lifecycle controls to the Kocao UI for sandbox-backed sessions.
- **Verify**: `pnpm -C web test && pnpm -C web lint`
- **Done When**: Users can start an OpenCode/Claude/Codex/Pi sandbox-backed session from the UI, interact with it, and stop/resume/reconnect without losing context.
- **Requirements**: agent-session-ui:agent-selection-and-launch, agent-session-ui:live-agent-interaction-view, agent-session-ui:lifecycle-controls-and-reconnect
- **Updated At**: 2026-04-11
- **Status**: [x] complete

### Task 3.2: Validate Kubernetes, API, and UI happy paths

- **Files**: `docs/*`, integration test assets, smoke scripts
- **Dependencies**: Task 3.1
- **Action**: Add verification coverage and operator docs for the cluster deployment, API flow, and UI flow covering supported agents and lifecycle operations.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/operator/... ./internal/harnessruntime/... && pnpm -C web test && pnpm -C web lint && kubectl kustomize deploy/base >/dev/null`
- **Done When**: The repository contains explicit verification guidance for the Kubernetes + API + UI acceptance path of sandbox-backed sessions.
- **Requirements**: sandbox-agent-runtime:supported-agent-catalog, control-plane-api:sandbox-agent-session-management-endpoints, control-plane-api:sandbox-agent-messaging-and-event-replay-api, control-plane-api:sandbox-agent-lifecycle-control-api, agent-session-ui:agent-selection-and-launch, agent-session-ui:live-agent-interaction-view, agent-session-ui:lifecycle-controls-and-reconnect
- **Updated At**: 2026-04-09
- **Status**: [ ] pending

______________________________________________________________________

## Wave Guidelines

- Waves group tasks that can run in parallel within the wave
- Wave N depends on all prior waves completing
- Task dependencies within a wave are fine; cross-wave deps use the wave dependency
- Checkpoint waves require human approval before proceeding
<!-- ITO:END -->
