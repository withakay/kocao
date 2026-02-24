# Tasks for: 003-03_run-commands-parity

## Execution Notes

- **Tool**: Any (OpenCode, Codex, Claude Code)
- **Mode**: Sequential (or parallel if tool supports)
- **Template**: Enhanced task format with waves, verification, and status tracking
- **Tracking**: Prefer the tasks CLI to drive status updates and pick work

```bash
ito tasks status 003-03_run-commands-parity
ito tasks next 003-03_run-commands-parity
ito tasks start 003-03_run-commands-parity 1.1
ito tasks complete 003-03_run-commands-parity 1.1
ito tasks shelve 003-03_run-commands-parity 1.1
ito tasks unshelve 003-03_run-commands-parity 1.1
ito tasks show 003-03_run-commands-parity
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Add Task + Advanced args to Start Run UI

- **Files**: `web/src/ui/pages/SessionDetailPage.tsx`, `web/src/ui/lib/api.ts`
- **Dependencies**: None
- **Action**: Add a Task textbox and an Advanced args editor to the Start Run form; map non-empty Task to `args: ["bash","-lc",task]`; keep empty Task interactive (omit args); add a warning against embedding secrets in Task.
- **Verify**: `cd web && pnpm test`
- **Done When**: UI can start a run that executes a provided task; UI can start an interactive run when Task is empty; tests pass.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 1.2: Add tests for Task->args mapping

- **Files**: `web/src/ui/workflow.test.tsx`
- **Dependencies**: Task 1.1
- **Action**: Extend UI tests to cover starting a run with Task (assert startRun request body includes `args`), and starting a run with empty Task (assert `args` omitted).
- **Verify**: `cd web && pnpm test`
- **Done When**: Tests fail before implementation and pass after; start-run behavior is covered.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 1.3: Add API test for args pass-through

- **Files**: `internal/controlplaneapi/api_test.go`
- **Dependencies**: None
- **Action**: Add a unit/integration test that creates a run with `args` via the API endpoint and asserts the created `HarnessRun` has `spec.args` set and no container-command override is required.
- **Verify**: `make test`
- **Done When**: API run-create behavior with args is covered by a test.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Checkpoints

### Checkpoint: Review Implementation

- **Type**: checkpoint (requires human approval)
- **Dependencies**: All Wave 1 tasks
- **Action**: Review the implementation before proceeding
- **Done When**: User confirms implementation is correct
- **Updated At**: 2026-02-24
- **Status**: [x] complete
