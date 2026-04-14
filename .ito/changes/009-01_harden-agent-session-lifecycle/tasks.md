<!-- ITO:START -->
# Tasks for: 009-01_harden-agent-session-lifecycle

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 009-01_harden-agent-session-lifecycle
ito tasks next 009-01_harden-agent-session-lifecycle
ito tasks start 009-01_harden-agent-session-lifecycle 1.1
ito tasks complete 009-01_harden-agent-session-lifecycle 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Introduce explicit agent session lifecycle model

- **Files**: `internal/controlplaneapi/agent_session.go`, `internal/controlplaneapi/*test.go`, `internal/operator/api/v1alpha1/*`
- **Dependencies**: None
- **Action**: Define explicit lifecycle states, transition rules, and durable state representation for sandbox-backed sessions.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/operator/...`
- **Done When**: Session creation, status, stop, and replay paths operate against an explicit lifecycle model with regression tests.
- **Requirements**: agent-session-lifecycle:explicit-state-machine, control-plane-api:session-api-contract-consistency
- **Updated At**: 2026-04-14
- **Status**: [x] complete

### Task 1.2: Add restart reconciliation for active sessions

- **Files**: `internal/controlplaneapi/*`, `internal/operator/controllers/*`, `internal/controlplaneapi/*test.go`
- **Dependencies**: Task 1.1
- **Action**: Reconstruct session state after API restart and ensure stop/get calls behave safely without an existing in-memory bridge.
- **Verify**: `go test ./internal/controlplaneapi/... -run Reconciliation`
- **Done When**: Restart-oriented tests prove that status and stop remain correct after API restart.
- **Requirements**: agent-session-lifecycle:restart-reconciliation
- **Updated At**: 2026-04-14
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Make session operations idempotent

- **Files**: `internal/controlplaneapi/*`, `internal/controlplanecli/*`, `cmd/kocao/*`
- **Dependencies**: None
- **Action**: Make create/get/stop/logs/prompt predictable and repeatable, including repeated create/stop calls.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/controlplanecli/... ./cmd/kocao/...`
- **Done When**: Repeated lifecycle calls return stable, documented outcomes and tests cover duplicate calls.
- **Requirements**: agent-session-lifecycle:idempotent-operations, control-plane-api:session-api-safety-on-failure
- **Updated At**: 2026-04-14
- **Status**: [x] complete

### Task 2.2: Add session diagnostics surfaces

- **Files**: `internal/controlplaneapi/*`, `internal/controlplanecli/*`, `cmd/kocao/*`, `docs/demos/*`
- **Dependencies**: None
- **Action**: Expose blocker diagnostics for provisioning, sandbox-agent readiness, auth, repo access, network, and image pull failure classes.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/controlplanecli/...`
- **Done When**: API and CLI status surfaces include meaningful blocker output and tests cover representative failure modes.
- **Requirements**: agent-diagnostics:session-blocker-diagnostics, agent-diagnostics:operator-and-cli-visibility
- **Updated At**: 2026-04-14
- **Status**: [x] complete

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Add live end-to-end lifecycle coverage in CI

- **Files**: `.github/workflows/*`, `Makefile`, `test/*`, `demos/*`
- **Dependencies**: None
- **Action**: Add Kind-backed live tests for start/status/list/logs/exec/stop and ensure the expected lifecycle is asserted in CI.
- **Verify**: CI workflow run or equivalent local Kind workflow command
- **Done When**: CI exercises the live lifecycle and fails on regressions that previously only appeared in MicroK8s demos.
- **Requirements**: agent-session-lifecycle:explicit-state-machine, agent-diagnostics:operator-and-cli-visibility
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 3.2: Refresh demos and operational guidance

- **Files**: `demos/*`, `docs/demos/*`, `.opencode/skills/kocao-agent/*`
- **Dependencies**: None
- **Action**: Update the live demos and skill guidance so they reflect the production-reliable lifecycle and diagnostics.
- **Verify**: `showboat verify <demo>` or documented manual replay
- **Done When**: Demos and skill instructions match the final lifecycle semantics and diagnostic output.
- **Requirements**: agent-diagnostics:operator-and-cli-visibility, control-plane-api:session-api-contract-consistency
- **Updated At**: 2026-04-13
- **Status**: [ ] pending
<!-- ITO:END -->
