# Tasks for: 003-09_attach-inspector-and-foldable-terminal

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 003-09_attach-inspector-and-foldable-terminal
ito tasks next 003-09_attach-inspector-and-foldable-terminal
ito tasks start 003-09_attach-inspector-and-foldable-terminal 1.1
ito tasks complete 003-09_attach-inspector-and-foldable-terminal 1.1
```

______________________________________________________________________

## Wave 1 - Shared Attach Layout State

- **Depends On**: None

### Task 1.1: Add attach layout store in `useLayoutState`

- **Files**: `web/src/ui/lib/useLayoutState.ts`
- **Dependencies**: None
- **Action**: Add shared attach layout state (fullscreen, inspector open, activity panel open) accessible from AttachPage, CommandPalette, and Shell keyboard shortcuts.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: Attach layout state is shared and updates propagate across independent component trees.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 1.2: Add inspector panel primitive component

- **Files**: `web/src/ui/components/InspectorPanel.tsx` (new)
- **Dependencies**: None
- **Action**: Create right-side inspector panel with slide-in animation, close button, and optional keyboard close support.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: Inspector panel renders and can be opened/closed with smooth transitions.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

______________________________________________________________________

## Wave 2 - Attach UX Integration

- **Depends On**: Wave 1

### Task 2.1: Integrate inspector and foldable activity panel into AttachPage

- **Files**: `web/src/ui/pages/AttachPage.tsx`
- **Dependencies**: None
- **Action**: Add a right inspector panel and a foldable bottom activity panel in AttachPage. Keep terminal as primary focus. Wire controls in connection bar and fullscreen header.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: Attach page supports inspector toggle and foldable activity panel with persisted state.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 2.2: Add command palette actions for inspector/activity toggles

- **Files**: `web/src/ui/components/CommandPalette.tsx`
- **Dependencies**: None
- **Action**: Add attach-only palette actions: Toggle Inspector and Toggle Activity Panel.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: Cmd+K exposes working attach-specific actions that update shared attach layout state.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 2.3: Add attach keyboard shortcuts in Shell

- **Files**: `web/src/ui/components/Shell.tsx`
- **Dependencies**: None
- **Action**: Add attach-page shortcuts: Cmd+I (toggle inspector), Cmd+J (toggle activity panel), without affecting non-attach routes.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: Attach shortcuts work globally on attach route and do nothing on other routes.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

______________________________________________________________________

## Wave 3 - Verification

- **Depends On**: Wave 1, Wave 2

### Task 3.1: Add UI tests for attach inspector/activity controls

- **Files**: `web/src/ui/workflow.test.tsx`
- **Dependencies**: None
- **Action**: Add tests covering attach inspector toggle and foldable activity panel visibility.
- **Verify**: `cd web && pnpm test`
- **Done When**: Existing tests plus new attach interaction tests pass.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 3.2: Full verification (typecheck, build, test)

- **Files**: All web source files
- **Dependencies**: Task 3.1
- **Action**: Run `pnpm tsc --noEmit`, `pnpm build`, and `pnpm test`; resolve issues.
- **Verify**: `cd web && pnpm tsc --noEmit && pnpm build && pnpm test`
- **Done When**: Zero type errors, successful build, all tests passing.
- **Updated At**: 2026-02-26
- **Status**: [x] complete
