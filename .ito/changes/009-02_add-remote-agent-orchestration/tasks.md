<!-- ITO:START -->
# Tasks for: 009-02_add-remote-agent-orchestration

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 009-02_add-remote-agent-orchestration
ito tasks next 009-02_add-remote-agent-orchestration
ito tasks start 009-02_add-remote-agent-orchestration 1.1
ito tasks complete 009-02_add-remote-agent-orchestration 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Define orchestration data model and API contract

- **Files**: `internal/controlplaneapi/*`, durability/storage model files, related tests
- **Dependencies**: None
- **Action**: Define named agents/pools, task records, transcript storage, artifact references, and API contracts for dispatch/status/cancel/result retrieval.
- **Verify**: `go test ./internal/controlplaneapi/...`
- **Done When**: Data model and API semantics are encoded in code/tests and match the proposal specs.
- **Requirements**: remote-agent-orchestration:named-remote-agents, remote-agent-orchestration:task-dispatch-lifecycle, agent-artifacts:persistent-agent-transcripts, agent-artifacts:attached-task-artifacts
- **Updated At**: 2026-04-14
- **Status**: [x] complete

### Task 1.2: Define dashboard information architecture

- **Files**: `web/src/ui/*`, design docs/mockups if needed
- **Dependencies**: Task 1.1
- **Action**: Define the operator dashboard views for active agents, tasks, transcripts, and artifacts.
- **Verify**: UI design review or component-level tests
- **Done When**: The dashboard structure is concrete enough to implement without guessing data shape later.
- **Requirements**: web-ui:remote-agent-operations-dashboard
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Implement orchestration API and persistence

- **Files**: `internal/controlplaneapi/*`, durability/storage code, `internal/controlplanecli/*`, tests
- **Dependencies**: None
- **Action**: Implement named-agent resolution, task dispatch, cancellation, retry, result retrieval, transcript persistence, and artifact reference storage.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/controlplanecli/...`
- **Done When**: API and storage behavior satisfy orchestration and artifact requirements with passing tests.
- **Requirements**: remote-agent-orchestration:named-remote-agents, remote-agent-orchestration:task-dispatch-lifecycle, agent-artifacts:persistent-agent-transcripts, agent-artifacts:attached-task-artifacts
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 2.2: Extend CLI for orchestration workflows

- **Files**: `cmd/kocao/*`, `internal/controlplanecli/*`, tests, demos
- **Dependencies**: None
- **Action**: Add commands for dispatching tasks, inspecting orchestrated agents, cancelling work, and retrieving transcripts/artifacts.
- **Verify**: `go test ./cmd/kocao/... ./internal/controlplanecli/...`
- **Done When**: CLI covers the orchestration lifecycle without requiring raw API calls.
- **Requirements**: remote-agent-orchestration:named-remote-agents, remote-agent-orchestration:task-dispatch-lifecycle
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Build remote-agent dashboard

- **Files**: `web/src/ui/*`, component tests, API integration points
- **Dependencies**: None
- **Action**: Build the dashboard and detail views for active agents, current tasks, transcripts, and artifacts.
- **Verify**: `pnpm -C web test && pnpm -C web lint`
- **Done When**: Operators can inspect remote-agent activity from the UI.
- **Requirements**: web-ui:remote-agent-operations-dashboard
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 3.2: Add multi-agent workflow demonstrations and E2E coverage

- **Files**: `demos/*`, integration tests, CI workflow updates
- **Dependencies**: None
- **Action**: Add end-to-end coverage and demos for reviewer/implementer/researcher multi-agent workflows.
- **Verify**: live or Kind-backed integration workflow + showboat verification
- **Done When**: Orchestration behavior is exercised in automated coverage and documented demos.
- **Requirements**: remote-agent-orchestration:multi-agent-workflow-coordination, agent-artifacts:persistent-agent-transcripts
- **Updated At**: 2026-04-13
- **Status**: [ ] pending
<!-- ITO:END -->
