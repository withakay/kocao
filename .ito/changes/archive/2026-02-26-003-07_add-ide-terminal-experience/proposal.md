# Change: Add IDE-quality terminal experience to attach UI

## Why

The attach page currently renders terminal output into a plain `<div>` using `textContent` appending -- a regression from the UI refactor that removed the real terminal integration. The `ghostty-web` and `@xterm/xterm` packages are installed but unused. Users need an IDE-quality terminal experience: a real terminal emulator (ghostty-web with xterm fallback), resizable panels, fullscreen mode, a command palette, toggleable sidebar, and keyboard shortcuts to make the attach workflow feel like a desktop application.

## What Changes

- Replace the plain `<div>` terminal output in AttachPage with a real ghostty-web/xterm.js terminal emulator using the adapter pattern from the archived 003-06 design
- Create a reusable `GhosttyTerminal` component with proper lifecycle management, auto-fit resize, and theme integration
- Wire WebSocket stdout/stdin through the terminal emulator instead of `textContent` appending
- Add resizable terminal panel (VS Code-style drag handle) with expand/collapse/fullscreen modes
- Add a command palette (`cmdk` package, Cmd+K trigger) with navigation actions, terminal controls, and search
- Make the sidebar toggleable with Cmd+\ shortcut and smooth collapse animation
- Add a keyboard shortcuts system: Cmd+K (palette), Cmd+\ (sidebar), Cmd+Enter (send stdin), Escape (exit fullscreen/close palette)
- Send `resize` messages over WebSocket when terminal dimensions change so the backend PTY stays in sync

## Impact

- Affected specs: `ide-terminal-experience` (new), `workflow-ui-github` (modified)
- Affected code:
  - `web/src/ui/pages/AttachPage.tsx` -- terminal integration, panel layout, keyboard shortcuts
  - `web/src/ui/components/GhosttyTerminal.tsx` -- new reusable terminal component
  - `web/src/ui/components/TerminalAdapter.ts` -- engine adapter interface
  - `web/src/ui/components/CommandPalette.tsx` -- new Cmd+K palette component
  - `web/src/ui/components/Shell.tsx` -- toggleable sidebar
  - `web/src/ui/components/ResizablePanel.tsx` -- drag-to-resize panel primitive
  - `web/package.json` -- add `cmdk` dependency
  - `web/src/ui/workflow.test.tsx` -- test updates for new behavior
