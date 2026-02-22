# Tasks for: 004-06_web-ui-security-hardening

## Execution Notes

- **Tool**: Any (OpenCode, Codex, Claude Code)
- **Mode**: Sequential (or parallel if tool supports)
- **Template**: Enhanced task format with waves, verification, and status tracking
- **Tracking**: Prefer the tasks CLI to drive status updates and pick work

```bash
ito tasks status 004-06_web-ui-security-hardening
ito tasks next 004-06_web-ui-security-hardening
ito tasks start 004-06_web-ui-security-hardening 1.1
ito tasks complete 004-06_web-ui-security-hardening 1.1
ito tasks shelve 004-06_web-ui-security-hardening 1.1
ito tasks unshelve 004-06_web-ui-security-hardening 1.1
ito tasks show 004-06_web-ui-security-hardening
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Make token persistence opt-in (session storage by default)

- **Files**: `web/src/ui/auth.tsx`, `web/src/ui/components/Topbar.tsx`, `web/src/ui/workflow.test.tsx`
- **Dependencies**: None
- **Action**: Change auth state to use session-scoped storage by default; add an explicit “remember” toggle for persistent storage.
- **Verify**: `cd web && pnpm test`
- **Done When**: Token does not persist across restarts by default; tests cover the UX.
- **Updated At**: 2026-02-22
- **Status**: [ ] pending

### Task 1.2: Clear token and guide user on auth failures

- **Files**: `web/src/ui/lib/api.ts`, `web/src/ui/auth.tsx`, `web/src/ui/pages/*`
- **Dependencies**: None
- **Action**: On 401 responses, clear stored token and render a clear prompt to re-enter credentials.
- **Verify**: `cd web && pnpm test`
- **Done When**: Auth failures do not repeatedly spam failing requests; user can recover easily.
- **Updated At**: 2026-02-22
- **Status**: [ ] pending

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Update attach UI to avoid URL tokens (align with backend)

- **Files**: `web/src/ui/pages/AttachPage.tsx`, `web/src/ui/lib/api.ts`, `web/src/ui/workflow.test.tsx`
- **Dependencies**: None
- **Action**: Update attach flow to use the backend-supported non-URL token transport (expected from change `004-04_harden-attach-websocket`).
- **Verify**: `cd web && pnpm test`
- **Done When**: Attach works without URL query tokens and tests cover the path.
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
