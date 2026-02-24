# Tasks for: 004-07_harden-harness-runtime

## Execution Notes

- **Tool**: Any (OpenCode, Codex, Claude Code)
- **Mode**: Sequential (or parallel if tool supports)
- **Template**: Enhanced task format with waves, verification, and status tracking
- **Tracking**: Prefer the tasks CLI to drive status updates and pick work

```bash
ito tasks status 004-07_harden-harness-runtime
ito tasks next 004-07_harden-harness-runtime
ito tasks start 004-07_harden-harness-runtime 1.1
ito tasks complete 004-07_harden-harness-runtime 1.1
ito tasks shelve 004-07_harden-harness-runtime 1.1
ito tasks unshelve 004-07_harden-harness-runtime 1.1
ito tasks show 004-07_harden-harness-runtime
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Add path safety checks in the harness entrypoint

- **Files**: `build/harness/kocao-harness-entrypoint.sh`
- **Dependencies**: None
- **Action**: Validate `KOCAO_WORKSPACE_DIR` and `KOCAO_REPO_DIR` are non-empty, absolute (or normalized), and ensure repo dir is within workspace before performing `rm -rf` or `git clone`.
- **Verify**: `make harness-smoke`
- **Done When**: The entrypoint fails fast on unsafe paths and cannot delete outside workspace.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

### Task 1.2: Harden git invocation (option injection + revision safety)

- **Files**: `build/harness/kocao-harness-entrypoint.sh`
- **Dependencies**: None
- **Action**: Ensure git operations use `--` separators where appropriate and handle edge-case repo URLs/revisions safely.
- **Verify**: `make harness-smoke`
- **Done When**: Entrypoint cannot treat user-provided repo/revision values as flags.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Reserve critical KOCAO env vars in the operator

- **Files**: `internal/operator/controllers/pod.go`, `internal/operator/controllers/harnessrun_controller.go`, `internal/operator/controllers/harnessrun_controller_test.go`
- **Dependencies**: None
- **Action**: Prevent `run.Spec.Env` from overriding reserved `KOCAO_*` variables (reject spec or drop overrides deterministically) and add tests.
- **Verify**: `make lint && make test`
- **Done When**: Reserved env variables cannot be overridden by user input.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

### Task 2.2: Harden harness pod security context

- **Files**: `internal/operator/controllers/pod.go`, `build/Dockerfile.harness`
- **Dependencies**: None
- **Action**: Add a reasonable default security context (run as non-root if feasible, drop capabilities, restrict privilege escalation). Update harness image/user setup to support it.
- **Verify**: `make lint && make test && make harness-smoke`
- **Done When**: Harness pods run with a restrictive security context without breaking normal workflows.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

______________________________________________________________________

## Checkpoints

### Checkpoint: Review Implementation

- **Type**: checkpoint (requires human approval)
- **Dependencies**: All Wave 2 tasks
- **Action**: Review the implementation before proceeding
- **Done When**: User confirms implementation is correct
- **Updated At**: 2026-02-23
- **Status**: [-] shelved
