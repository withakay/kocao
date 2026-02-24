# Tasks for: 003-06_terminal-engine-toggle-ghostty-xterm

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 003-06_terminal-engine-toggle-ghostty-xterm
ito tasks next 003-06_terminal-engine-toggle-ghostty-xterm
ito tasks start 003-06_terminal-engine-toggle-ghostty-xterm 1.1
ito tasks complete 003-06_terminal-engine-toggle-ghostty-xterm 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Add terminal engine adapter boundary

- **Files**: `web/src/ui/pages/AttachPage.tsx`, `web/src/ui/components/*` (new adapter files as needed)
- **Dependencies**: None
- **Action**: Introduce a renderer-agnostic terminal adapter interface and move attach transport plumbing so the UI can bind either xterm or ghostty without changing websocket logic.
- **Verify**: `cd web && pnpm test`
- **Done When**: Attach transport/input/output path is decoupled from concrete renderer and existing attach behavior still works.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending

### Task 1.2: Integrate xterm and ghostty engines with per-session toggle

- **Files**: `web/src/ui/pages/AttachPage.tsx`, `web/package.json`, renderer adapter files
- **Dependencies**: Task 1.1
- **Action**: Add engine selector in attach UI with `xterm.js` and `ghostty-web (experimental)`, default to xterm, and support immediate engine switching in active sessions.
- **Verify**: `cd web && pnpm test`
- **Done When**: Users can hot-switch engines during an attach session and output continues streaming without requiring a new session.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending

### Task 1.3: Persist engine selection in cookie by workspace session

- **Files**: `web/src/ui/pages/AttachPage.tsx`, `web/src/ui/lib/*` (cookie helper if added)
- **Dependencies**: Task 1.2
- **Action**: Persist selected engine in cookie keyed by workspace session ID and restore choice on reload for that same session.
- **Verify**: `cd web && pnpm test`
- **Done When**: Reloading attach page for a session restores prior engine choice from cookie state.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending

### Task 1.4: Add tests for interchangeability and persistence

- **Files**: `web/src/ui/workflow.test.tsx` (or new attach test file)
- **Dependencies**: Task 1.2, Task 1.3
- **Action**: Add tests covering engine toggle visibility, immediate switching semantics, and cookie-based restore behavior.
- **Verify**: `cd web && pnpm test`
- **Done When**: New tests fail before implementation and pass after; key acceptance scenarios are automated.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: End-to-end verification and compatibility check

- **Files**: `web/src/ui/pages/AttachPage.tsx`, test files, docs/comments as needed
- **Dependencies**: None
- **Action**: Validate no attach protocol changes were introduced and verify fallback/default behavior remains xterm-first with ghostty marked experimental.
- **Verify**: `cd web && pnpm test` and `make test`
- **Done When**: Web and repo test suites pass and acceptance criteria in spec are satisfied.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending
