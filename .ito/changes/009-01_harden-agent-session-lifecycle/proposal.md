<!-- ITO:START -->
## Why

Module 007 established the sandbox-agent-backed session path, but live MicroK8s validation showed that the platform still has lifecycle ambiguity, restart sensitivity, non-idempotent behavior, and poor diagnostics when something goes wrong. Kocao needs a production-grade session lifecycle before additional orchestration or runtime optimization work will be trustworthy.

## What Changes

- Add an explicit agent session state machine with durable transitions and terminal-state semantics.
- Add control-plane reconciliation and resume behavior so sessions remain understandable after API/operator restarts.
- Make session create, get, stop, logs, and prompt paths idempotent and restart-safe.
- Add diagnostic surfaces that explain whether a session is blocked on pod scheduling, image pull, sandbox-agent readiness, auth, or network reachability.
- Add live end-to-end verification against Kind in CI so lifecycle regressions are caught before merge.
- Build directly on the sandbox-agent platform from module 007 rather than introducing a second runtime abstraction.

## Capabilities

### New Capabilities

- `agent-session-lifecycle`: explicit state machine and reconciliation contract for sandbox-backed agent sessions.
- `agent-diagnostics`: operator/API/CLI-visible diagnostics describing why a session is blocked, degraded, or failed.

### Modified Capabilities

- `control-plane-api`: session endpoints become idempotent, restart-safe, and state-machine-aware.
- `session-durability`: persisted agent session state must survive control-plane restarts and enable safe resume/replay.
- `run-execution`: harness runs must surface readiness and failure reasons that map cleanly onto the agent session lifecycle.

## Impact

- Affected code: `internal/controlplaneapi/*`, `internal/controlplanecli/*`, `internal/operator/*`, `build/harness/*`, `cmd/kocao/*`, CI workflow/test infrastructure, and associated docs/demos.
- APIs: refined semantics for create/status/prompt/logs/stop plus new diagnostic fields and/or endpoints.
- Operations: sessions become restart-safe and diagnosable instead of relying on transient in-memory bridges alone.
- Module relationship: this change hardens module 007's sandbox-agent session path and should be treated as a dependency for later orchestration and runtime-profile work in module 009.
<!-- ITO:END -->
