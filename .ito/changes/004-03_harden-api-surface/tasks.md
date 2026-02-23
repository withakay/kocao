# Tasks for: 004-03_harden-api-surface

## Execution Notes

- **Tool**: Any (OpenCode, Codex, Claude Code)
- **Mode**: Sequential (or parallel if tool supports)
- **Template**: Enhanced task format with waves, verification, and status tracking
- **Tracking**: Prefer the tasks CLI to drive status updates and pick work

```bash
ito tasks status 004-03_harden-api-surface
ito tasks next 004-03_harden-api-surface
ito tasks start 004-03_harden-api-surface 1.1
ito tasks complete 004-03_harden-api-surface 1.1
ito tasks shelve 004-03_harden-api-surface 1.1
ito tasks unshelve 004-03_harden-api-surface 1.1
ito tasks show 004-03_harden-api-surface
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Add JSON request body size limits and tests

- **Files**: `internal/controlplaneapi/json.go`, `internal/controlplaneapi/api_test.go`
- **Dependencies**: None
- **Action**: Introduce a maximum body size (e.g. via `http.MaxBytesReader`) for JSON endpoints and add tests for oversized payload rejection.
- **Verify**: `make lint && make test`
- **Done When**: Oversized requests are rejected deterministically and tests cover the behavior.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

### Task 1.2: Add server timeouts and basic hardening headers

- **Files**: `cmd/control-plane-api/main.go`, `internal/controlplaneapi/api.go`
- **Dependencies**: None
- **Action**: Configure `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` (in addition to `ReadHeaderTimeout`). Add low-risk headers like `X-Content-Type-Options: nosniff` for JSON responses.
- **Verify**: `make lint && make test`
- **Done When**: Server timeouts are set and tests still pass.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

### Task 1.3: Enforce bootstrap-token safety in prod

- **Files**: `internal/config/config.go`, `cmd/control-plane-api/main.go`, `internal/controlplaneapi/tokens.go`
- **Dependencies**: None
- **Action**: Ensure `CP_ENV=prod` prevents unsafe bootstrap token usage (startup error or explicit ignore + warning) and add tests.
- **Verify**: `make lint && make test`
- **Done When**: Production config path cannot silently enable wildcard access.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Reduce authz footguns in routing

- **Files**: `internal/controlplaneapi/api.go`, `internal/controlplaneapi/auth.go`
- **Dependencies**: None
- **Action**: Refactor patterns so adding a new `/api/v1/*` handler is unlikely to bypass authorization checks (e.g. centralized scope enforcement wrappers).
- **Verify**: `make lint && make test`
- **Done When**: New handlers follow a consistent pattern that fails closed.
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
- **Status**: [x] complete
