# Tasks for: 002-03_clarify-harness-run-and-session-terminology

## Execution Notes

- **Tool**: Any (OpenCode, Codex, Claude Code)
- **Mode**: Sequential (or parallel if tool supports)
- **Template**: Enhanced task format with waves, verification, and status tracking
- **Tracking**: Prefer the tasks CLI to drive status updates and pick work

```bash
ito tasks status 002-03_clarify-harness-run-and-session-terminology
ito tasks next 002-03_clarify-harness-run-and-session-terminology
ito tasks start 002-03_clarify-harness-run-and-session-terminology 1.1
ito tasks complete 002-03_clarify-harness-run-and-session-terminology 1.1
ito tasks shelve 002-03_clarify-harness-run-and-session-terminology 1.1
ito tasks unshelve 002-03_clarify-harness-run-and-session-terminology 1.1
ito tasks show 002-03_clarify-harness-run-and-session-terminology
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Finalize canonical glossary and migration matrix

- **Files**: `.ito/changes/002-03_clarify-harness-run-and-session-terminology/proposal.md`, `.ito/changes/002-03_clarify-harness-run-and-session-terminology/design.md`, `.ito/changes/002-03_clarify-harness-run-and-session-terminology/specs/*/spec.md`
- **Dependencies**: None
- **Action**: Lock canonical terms and add/verify explicit old-to-new migration mapping for all renamed concepts.
- **Verify**: `ito validate 002-03_clarify-harness-run-and-session-terminology --strict`
- **Done When**: Glossary is canonical, matrix is present, and no bare `lifecycle` language remains in change artifacts.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 1.2: Define surface-specific language rules

- **Files**: `.ito/changes/002-03_clarify-harness-run-and-session-terminology/proposal.md`, `.ito/changes/002-03_clarify-harness-run-and-session-terminology/design.md`
- **Dependencies**: Task 1.1
- **Action**: Specify contract-facing language requirements and product-facing UX copy guidance while keeping canonical object names.
- **Verify**: `ito validate 002-03_clarify-harness-run-and-session-terminology --strict`
- **Done When**: Proposal and design clearly separate contract wording from UX explanatory copy rules.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Rename contract surfaces for hard cutover

- **Files**: `internal/controlplaneapi/*`, `internal/operator/controllers/*`, `api/**`, `deploy/**`, `docs/**`
- **Dependencies**: None
- **Action**: Apply hard-cutover terminology to API/schema/CRD/status and controller-facing contract surfaces.
- **Verify**: `make test`
- **Done When**: Contract surfaces use only Workspace Session and Harness Run naming.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 2.2: Enforce qualified lifecycle labels everywhere

- **Files**: `internal/controlplaneapi/*`, `internal/operator/controllers/*`, `web/src/ui/**/*`, `docs/**`
- **Dependencies**: Task 2.1
- **Action**: Replace unqualified lifecycle wording with object-qualified labels in API/UI/docs outputs.
- **Verify**: `make test && cd web && pnpm test`
- **Done When**: Only `Workspace Session Lifecycle` and `Harness Run Lifecycle` labels are exposed.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Align UI copy with canonical terminology and helper text

- **Files**: `web/src/ui/pages/*`, `web/src/ui/components/*`, `web/src/ui/lib/*`, `web/src/ui/workflow.test.tsx`
- **Dependencies**: None
- **Action**: Update labels/badges/messages to canonical nouns and add concise helper text where needed to preserve readability.
- **Verify**: `cd web && pnpm test`
- **Done When**: UX is understandable while preserving canonical naming and no ambiguous session/run wording remains.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 3.2: Publish migration and release guidance

- **Files**: `docs/**`, `.ito/changes/002-03_clarify-harness-run-and-session-terminology/proposal.md`, `.ito/changes/002-03_clarify-harness-run-and-session-terminology/design.md`
- **Dependencies**: Task 3.1
- **Action**: Add release-note-ready migration guidance for the breaking rename, including old-to-new term examples.
- **Verify**: `ito validate 002-03_clarify-harness-run-and-session-terminology --strict`
- **Done When**: Migration guidance is explicit enough for downstream clients to update integrations.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Checkpoints

### Checkpoint: Review terminology and migration impact

- **Type**: checkpoint (requires human approval)
- **Dependencies**: All Wave 3 tasks
- **Action**: Review naming, UX copy split, and migration matrix before implementation proceeds.
- **Done When**: User confirms proposal artifacts are ready.
- **Updated At**: 2026-02-24
- **Status**: [-] shelved
