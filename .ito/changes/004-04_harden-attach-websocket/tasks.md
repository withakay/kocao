# Tasks for: 004-04_harden-attach-websocket

## Execution Notes

- **Tool**: Any (OpenCode, Codex, Claude Code)
- **Mode**: Sequential (or parallel if tool supports)
- **Template**: Enhanced task format with waves, verification, and status tracking
- **Tracking**: Prefer the tasks CLI to drive status updates and pick work

```bash
ito tasks status 004-04_harden-attach-websocket
ito tasks next 004-04_harden-attach-websocket
ito tasks start 004-04_harden-attach-websocket 1.1
ito tasks complete 004-04_harden-attach-websocket 1.1
ito tasks shelve 004-04_harden-attach-websocket 1.1
ito tasks unshelve 004-04_harden-attach-websocket 1.1
ito tasks show 004-04_harden-attach-websocket
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Add websocket limits and deadlines

- **Files**: `internal/controlplaneapi/attach.go`, `internal/controlplaneapi/attach_test.go`
- **Dependencies**: None
- **Action**: Add websocket `SetReadLimit`, read deadlines, and ping/pong handling. Add tests that prove oversized messages and idle connections are handled safely.
- **Verify**: `make lint && make test`
- **Done When**: Attach websocket resists basic abuse patterns and tests cover the limits.
- **Updated At**: 2026-02-22
- **Status**: [ ] pending

### Task 1.2: Enforce websocket Origin allowlist

- **Files**: `internal/controlplaneapi/attach.go`, `internal/config/config.go`, `internal/controlplaneapi/api_test.go`
- **Dependencies**: None
- **Action**: Add config for allowed Origins and wire Origin validation into the websocket upgrader. Default strict in prod, permissive only where necessary for local dev.
- **Verify**: `make lint && make test`
- **Done When**: Unexpected Origins are rejected and tests cover allow/deny behavior.
- **Updated At**: 2026-02-22
- **Status**: [ ] pending

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Implement browser attach token transport without URL query secrets

- **Files**: `internal/controlplaneapi/attach.go`, `internal/controlplaneapi/api.go`, `web/src/ui/pages/AttachPage.tsx`, `web/src/ui/lib/api.ts`, `web/src/ui/workflow.test.tsx`
- **Dependencies**: None
- **Action**: Add a browser-friendly attach flow (prefer HttpOnly cookie) so the UI can establish a websocket connection without placing tokens in the URL.
- **Verify**: `make lint && make test && (cd web && pnpm test)`
- **Done When**: UI attach works without URL tokens, and tests verify the new behavior.
- **Updated At**: 2026-02-22
- **Status**: [ ] pending

### Task 2.2: Expand auditing for attach events

- **Files**: `internal/controlplaneapi/attach.go`, `internal/controlplaneapi/audit.go`
- **Dependencies**: None
- **Action**: Record attach connect/disconnect, control acquisition, and stdin usage with minimally sensitive metadata.
- **Verify**: `make lint && make test`
- **Done When**: Attach activity is visible in audit output and tests cover key events.
- **Updated At**: 2026-02-22
- **Status**: [ ] pending

______________________________________________________________________

## Checkpoints

### Checkpoint: Review Implementation

- **Type**: checkpoint (requires human approval)
- **Dependencies**: All Wave 2 tasks
- **Action**: Review the implementation before proceeding
- **Done When**: User confirms implementation is correct
- **Updated At**: 2026-02-22
- **Status**: [ ] pending
