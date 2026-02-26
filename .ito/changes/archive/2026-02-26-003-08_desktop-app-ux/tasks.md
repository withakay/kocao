# Tasks for: 003-08_desktop-app-ux

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 003-08_desktop-app-ux
ito tasks next 003-08_desktop-app-ux
ito tasks start 003-08_desktop-app-ux 1.1
ito tasks complete 003-08_desktop-app-ux 1.1
```

______________________________________________________________________

## Wave 1 - Resizable Panel System & Fullscreen Context

- **Depends On**: None

### Task 1.1: Create ResizablePanel primitive component

- **Files**: `web/src/ui/components/ResizablePanel.tsx` (new)
- **Dependencies**: None
- **Action**: Build a generic resizable panel component using pointer events (pointerdown/move/up). Supports horizontal and vertical split. Drag handle renders as a thin bar with hover highlight. Respects min/max size constraints. Fires onResize callback for consumers to refit content (e.g., terminal). Persists panel sizes in localStorage keyed by panel ID.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: Panel component renders, drag handle resizes panels smoothly, constraints respected, sizes persist.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 1.2: Wire resizable terminal panel into AttachPage

- **Files**: `web/src/ui/pages/AttachPage.tsx`
- **Dependencies**: Task 1.1
- **Action**: Replace the flex-1 terminal panel with ResizablePanel. Connection info card above, terminal below with drag handle between them. Terminal refits on resize. Add a secondary bottom panel slot for future inspector/logs.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: Terminal panel is user-resizable via drag handle, terminal refits correctly, min/max bounds work.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 1.3: Add fullscreen context and wire command palette toggle

- **Files**: `web/src/ui/lib/useLayoutState.ts`, `web/src/ui/components/CommandPalette.tsx`, `web/src/ui/pages/AttachPage.tsx`
- **Dependencies**: None
- **Action**: Add a FullscreenContext to useLayoutState that AttachPage provides and CommandPalette can consume. Re-add "Toggle Fullscreen Terminal" action in command palette that actually toggles the state.
- **Verify**: `cd web && pnpm test`
- **Done When**: Cmd+K palette shows "Toggle Fullscreen Terminal" on attach page, and clicking it actually enters/exits fullscreen.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 1.4: Add resizable sidebar width

- **Files**: `web/src/ui/components/Shell.tsx`
- **Dependencies**: Task 1.1
- **Action**: Replace the fixed w-52 sidebar with a ResizablePanel so users can drag the sidebar edge to resize. Persist width in localStorage. Respect min (180px) and max (320px) constraints.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: Sidebar is user-resizable by dragging its right edge, width persists across reloads.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

______________________________________________________________________

## Wave 2 - Collapsible Sections

- **Depends On**: None

### Task 2.1: Create CollapsibleSection primitive

- **Files**: `web/src/ui/components/primitives.tsx`
- **Dependencies**: None
- **Action**: Add a CollapsibleSection component (title bar with chevron toggle, smooth height animation via CSS grid trick). Open/closed state optionally persisted via localStorage key. Used for detail page sections.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: Section collapses/expands with smooth animation, chevron rotates, optional persistence works.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 2.2: Refactor detail pages to use collapsible sections

- **Files**: `web/src/ui/pages/SessionDetailPage.tsx`, `web/src/ui/pages/RunDetailPage.tsx`
- **Dependencies**: Task 2.1
- **Action**: Wrap each card/section in SessionDetailPage and RunDetailPage with CollapsibleSection. Sections: Connection Info, Start Run Form, Runs List (session detail); Run Info, GitHub Outcome, Logs (run detail). All start expanded.
- **Verify**: `cd web && pnpm test`
- **Done When**: Each section can be individually collapsed/expanded, page still functions correctly.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

______________________________________________________________________

## Wave 3 - Command Palette & Visual Polish

- **Depends On**: None

### Task 3.1: Upgrade command palette to Cursor-style dark aesthetic

- **Files**: `web/src/ui/components/CommandPalette.tsx`, `web/src/ui/styles.css`
- **Dependencies**: None
- **Action**: Redesign command palette with: darker bg (#0a0a0a), glassmorphism backdrop (backdrop-blur-xl + semi-transparent border), larger input with heavier placeholder, category icons (inline SVG or lucide), keyboard shortcut badges (rounded pill style like Cursor), subtle separator lines, smoother focus ring, wider max-width (lg instead of md). Add more actions: theme toggle placeholder, refresh data, close all panels.
- **Verify**: `cd web && pnpm tsc --noEmit && pnpm build`
- **Done When**: Command palette looks like Cursor's Cmd+K — dark, polished, glassmorphic, with icons and shortcut badges.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 3.2: Tighten spacing and typography for desktop density

- **Files**: `web/src/ui/styles.css`, `web/src/ui/components/primitives.tsx`
- **Dependencies**: None
- **Action**: Audit all spacing and reduce where web-page-like. Tighter card padding (p-2.5 instead of p-3), smaller gaps between sections, slightly smaller headings. Ensure everything still fits viewport without scrollbar on 1080p. Fine-tune font weights for visual hierarchy.
- **Verify**: `cd web && pnpm build`
- **Done When**: UI feels denser and more app-like, no unnecessary whitespace, everything viewport-contained.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

______________________________________________________________________

## Wave 4 - Verification

- **Depends On**: Wave 1, Wave 2, Wave 3

### Task 4.1: Update tests for new components

- **Files**: `web/src/ui/workflow.test.tsx`
- **Dependencies**: None
- **Action**: Add tests for: collapsible section toggle, resizable panel drag, fullscreen context wiring, command palette new actions. Ensure all existing tests still pass.
- **Verify**: `cd web && pnpm test`
- **Done When**: All existing tests pass, new tests cover key interactions.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 4.2: Full build and type-check verification

- **Files**: All web source files
- **Dependencies**: Task 4.1
- **Action**: Run full TypeScript compilation, Vite production build, and test suite. Fix any issues.
- **Verify**: `cd web && pnpm tsc --noEmit && pnpm build && pnpm test`
- **Done When**: Zero TypeScript errors, successful production build, all tests pass.
- **Updated At**: 2026-02-26
- **Status**: [x] complete
