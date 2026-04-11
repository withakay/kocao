<!-- ITO:START -->
# Tasks for: 006-02_add-symphony-workflow-engine

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 006-02_add-symphony-workflow-engine
ito tasks next 006-02_add-symphony-workflow-engine
ito tasks start 006-02_add-symphony-workflow-engine 1.1
ito tasks complete 006-02_add-symphony-workflow-engine 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Implement the workflow contract loader

- **Files**: new workflow contract package under `internal/`, tests, configuration helpers
- **Dependencies**: None
- **Action**: Implement `WORKFLOW.md` discovery, YAML front matter parsing, strict prompt-template handling, and typed workflow config defaults and validation.
- **Verify**: `go test ./internal/...`
- **Done When**: Workflow files can be loaded, validated, and rendered with deterministic tests for parse and render failures.
- **Updated At**: 2026-03-10
- **Status**: [x] complete

### Task 1.2: Implement the Codex app-server runner

- **Files**: new Symphony agent-runner package under `internal/`, protocol fixtures/tests
- **Dependencies**: Task 1.1
- **Action**: Implement app-server startup, turn streaming, continuation turn support, timeout handling, and telemetry extraction.
- **Verify**: `go test ./internal/...`
- **Done When**: The agent runner can execute fixture-backed sessions and report normalized worker outcomes plus telemetry in tests.
- **Updated At**: 2026-03-13
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Integrate workflow-driven execution into Symphony orchestration

- **Files**: `internal/operator/controllers/`, execution helpers, orchestration tests
- **Dependencies**: None
- **Action**: Wire workflow loading, workspace preparation, worker execution, continuation retries, and richer runtime state into the existing Symphony orchestrator.
- **Verify**: `go test ./internal/operator/... ./internal/...`
- **Done When**: Symphony projects execute workflow-driven workers and record continuation plus retry behavior in tests.
- **Updated At**: 2026-03-13
- **Status**: [x] complete

### Task 2.2: Extend API and UI for worker telemetry

- **Files**: `internal/controlplaneapi/`, `web/src/ui/`, API/UI tests
- **Dependencies**: Task 2.1
- **Action**: Expose recent worker events and aggregate execution counters through the control-plane API and Symphony detail UI.
- **Verify**: `go test ./internal/controlplaneapi/... && pnpm -C web test && pnpm -C web lint`
- **Done When**: Operators can inspect richer workflow-driven worker telemetry through the existing Symphony status surfaces.
- **Updated At**: 2026-03-13
- **Status**: [x] complete

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Harden workflow trust posture and secrets handling

- **Files**: security-sensitive controller/runner code, audit hooks, docs/tests
- **Dependencies**: None
- **Action**: Enforce and document approval posture, sandbox defaults, hook trust boundaries, and secret-safe telemetry for workflow-driven Symphony runs.
- **Verify**: `make test && make lint`
- **Done When**: Security-sensitive workflow execution paths are documented, audited, and covered by tests.
- **Updated At**: 2026-03-13
- **Status**: [x] complete

### Task 3.2: Add integration coverage and operator docs

- **Files**: docs, integration tests, example workflow config
- **Dependencies**: Task 3.1
- **Action**: Add end-to-end coverage and documentation for repository workflow contracts, agent execution, and operator debugging.
- **Verify**: `make test && pnpm -C web test && pnpm -C web lint`
- **Done When**: The workflow engine path is documented and validated by integration coverage.
- **Updated At**: 2026-03-13
- **Status**: [x] complete

______________________________________________________________________

## Wave Guidelines

- Waves group tasks that can run in parallel within the wave
- Wave N depends on all prior waves completing
- Task dependencies within a wave are fine; cross-wave deps use the wave dependency
- Checkpoint waves require human approval before proceeding
<!-- ITO:END -->
