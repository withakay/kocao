# Tasks for: 003-07_add-ide-terminal-experience

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 003-07_add-ide-terminal-experience
ito tasks next 003-07_add-ide-terminal-experience
ito tasks start 003-07_add-ide-terminal-experience 1.1
ito tasks complete 003-07_add-ide-terminal-experience 1.1
```

______________________________________________________________________

## Wave 1 - Terminal Engine Foundation

- **Depends On**: None

### Task 1.1: Create terminal engine adapter interface and implementations

- **Files**: `web/src/ui/components/TerminalAdapter.ts` (new)
- **Dependencies**: None
- **Action**: Define `TerminalEngine` interface with `mount(el)`, `write(data: Uint8Array)`, `resize(cols, rows)`, `dispose()`, `onInput(cb: (data: string) => void)`, `dimensions(): { cols: number; rows: number }`. Implement `GhosttyEngine` (using ghostty-web + init()) and `XtermEngine` (using @xterm/xterm + @xterm/addon-fit). Add `createEngine()` factory that tries ghostty-web first, falls back to xterm.js.
- **Verify**: `cd web && pnpm lint`
- **Done When**: Adapter interface and both engine implementations compile. Factory function handles ghostty init failure gracefully.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 1.2: Create reusable GhosttyTerminal React component

- **Files**: `web/src/ui/components/GhosttyTerminal.tsx` (new)
- **Dependencies**: Task 1.1
- **Action**: Create a React component that wraps the terminal engine adapter. Handles: engine creation on mount, auto-fit on container resize (ResizeObserver), cleanup on unmount, theme integration (use CSS variables from styles.css for terminal colors). Exposes imperative handle for `write(data)` and `dimensions()`. Calls `onInput` callback when user types.
- **Verify**: `cd web && pnpm lint`
- **Done When**: Component mounts a real terminal emulator in a container div, auto-fits, and exposes write/input APIs.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 1.3: Wire terminal component into AttachPage replacing plain div

- **Files**: `web/src/ui/pages/AttachPage.tsx`
- **Dependencies**: Task 1.2
- **Action**: Replace the plain `<div ref={termRef}>` with `<GhosttyTerminal>`. Wire: WebSocket stdout messages -> `termRef.write(decodedBytes)`, terminal onInput -> WebSocket stdin send. Send `resize` messages when terminal dimensions change. Remove the text Input + Send button in driver mode (terminal handles input directly). Keep the Input + Send as a fallback for viewer mode or when terminal is not focused.
- **Verify**: `cd web && pnpm test && pnpm lint`
- **Done When**: Attach page renders a real terminal emulator, bidirectional I/O works through WebSocket, resize messages are sent.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

______________________________________________________________________

## Wave 2 - IDE Panel System

- **Depends On**: Wave 1

### Task 2.1: Add resizable terminal panel with drag handle

- **Files**: `web/src/ui/components/ResizablePanel.tsx` (new), `web/src/ui/pages/AttachPage.tsx`
- **Dependencies**: None
- **Action**: Create a ResizablePanel component with a drag handle (pointer events), min/max height constraints, and smooth resize. Integrate into AttachPage so the connection info card is above and the terminal panel fills remaining space with adjustable height. Fire resize callback so terminal can re-fit.
- **Verify**: `cd web && pnpm lint`
- **Done When**: Terminal panel can be dragged to resize, respects min/max bounds, terminal re-fits on resize.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 2.2: Add fullscreen terminal mode

- **Files**: `web/src/ui/pages/AttachPage.tsx`
- **Dependencies**: Task 2.1
- **Action**: Fullscreen mode renders terminal as viewport overlay with minimal status bar. Toggle via button and Escape to exit. Terminal re-fits on fullscreen enter/exit. Send resize message to WebSocket.
- **Verify**: `cd web && pnpm lint`
- **Done When**: Fullscreen toggle works, terminal fills viewport, Escape exits, resize messages sent.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

______________________________________________________________________

## Wave 3 - Shell & Navigation Enhancements

- **Depends On**: Wave 1

### Task 3.1: Make sidebar toggleable with keyboard shortcut

- **Files**: `web/src/ui/components/Shell.tsx`, `web/src/ui/lib/useLayoutState.ts` (new)
- **Dependencies**: None
- **Action**: Add layout context provider with sidebar open/closed state persisted in localStorage. Add Cmd+\ (Ctrl+\ on non-Mac) keyboard shortcut to toggle. Add collapse/expand animation (CSS transition on width). Add a small toggle button visible when sidebar is collapsed.
- **Verify**: `cd web && pnpm test && pnpm lint`
- **Done When**: Sidebar collapses/expands with shortcut, state persists across reload, animation is smooth.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 3.2: Install cmdk and build command palette

- **Files**: `web/package.json`, `web/src/ui/components/CommandPalette.tsx` (new), `web/src/ui/components/Shell.tsx`
- **Dependencies**: Task 3.1
- **Action**: Install `cmdk` package. Build a command palette component with: navigation actions (Go to Sessions, Go to Runs), sidebar toggle, fullscreen terminal toggle (when on attach page). Style with monochrome theme. Trigger with Cmd+K. Wire into Shell so it's available on all routes.
- **Verify**: `cd web && pnpm test && pnpm lint`
- **Done When**: Command palette opens with Cmd+K, shows searchable actions, navigates on selection, closes on Escape.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 3.3: Add keyboard shortcuts system

- **Files**: `web/src/ui/lib/useKeyboardShortcuts.ts` (new), `web/src/ui/components/Shell.tsx`
- **Dependencies**: Task 3.1, Task 3.2
- **Action**: Create a `useKeyboardShortcuts` hook that registers global keydown listeners for: Cmd+K (palette), Cmd+\ (sidebar), Escape (close palette / exit fullscreen). Handle Mac vs non-Mac modifier detection. Prevent conflicts with terminal input by checking `event.target`.
- **Verify**: `cd web && pnpm test && pnpm lint`
- **Done When**: All shortcuts work from any page, do not interfere with terminal input in attach page.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

______________________________________________________________________

## Wave 4 - Integration & Verification

- **Depends On**: Wave 1, Wave 2, Wave 3

### Task 4.1: Update tests for terminal and IDE features

- **Files**: `web/src/ui/workflow.test.tsx`
- **Dependencies**: None
- **Action**: Update existing attach-ui test to work with the terminal component (mock ghostty-web/xterm init). Add tests for: sidebar toggle persistence, command palette open/close, keyboard shortcuts. Ensure all 6 existing tests still pass.
- **Verify**: `cd web && pnpm test`
- **Done When**: All existing tests pass, new tests cover terminal mount, sidebar toggle, and command palette.
- **Updated At**: 2026-02-26
- **Status**: [x] complete

### Task 4.2: Full build and type-check verification

- **Files**: All web source files
- **Dependencies**: Task 4.1
- **Action**: Run full TypeScript compilation (`pnpm lint`), Vite production build (`pnpm build`), and test suite (`pnpm test`). Fix any issues found.
- **Verify**: `cd web && pnpm lint && pnpm build && pnpm test`
- **Done When**: Zero TypeScript errors, successful production build, all tests pass.
- **Updated At**: 2026-02-26
- **Status**: [x] complete
