## Context

The attach page lost its terminal emulator during a UI refactor and now appends text to a plain `<div>`. The ghostty-web, @xterm/xterm, and @xterm/addon-fit packages are already installed as dependencies. The backend sends base64-encoded stdout over WebSocket and accepts base64-encoded stdin and `resize` messages with cols/rows.

The archived change 003-06 designed a terminal engine adapter interface. This change implements that adapter, wires it to the WebSocket transport, and adds the IDE-like UX layer on top.

## Goals / Non-Goals

**Goals:**

- Restore real terminal emulation in the attach page using ghostty-web (primary) with xterm.js fallback
- Create a narrow adapter interface (`mount`, `write`, `resize`, `dispose`, `onInput`) so engines are swappable
- Wire WebSocket stdout -> `term.write()` and `term.onData()` -> WebSocket stdin
- Send WebSocket `resize` messages when terminal dimensions change
- Add resizable panel with drag handle, expand/collapse, and fullscreen
- Add command palette (Cmd+K) for navigation and terminal actions
- Make sidebar collapsible with keyboard shortcut (Cmd+\)
- Full keyboard shortcut system for power users

**Non-Goals:**

- Backend protocol changes (attach WebSocket protocol is unchanged)
- Global user preferences system
- Terminal engine selection UI (was 003-06, can be re-added as follow-up)
- Mobile/touch support

## Decisions

- **ghostty-web as primary engine, xterm.js as fallback.**
  - Decision: Try ghostty-web first; if `init()` fails (WASM load error), fall back to xterm.js silently.
  - Rationale: ghostty-web is the preferred renderer per product direction but may fail in some browser environments.

- **Adapter interface for terminal engines.**
  - Decision: Define `TerminalEngine` interface with `mount(el)`, `write(data: Uint8Array)`, `resize(cols, rows)`, `dispose()`, `onInput(cb)`, `dimensions()`.
  - Rationale: decouples transport from renderer, enables future engine additions.

- **Resize notification over WebSocket.**
  - Decision: When terminal dimensions change (fit, panel resize, fullscreen), send `{ type: "resize", cols, rows }` to the WebSocket.
  - Rationale: Backend uses SPDY executor with TerminalSizeQueue; it already handles resize messages.

- **cmdk for command palette.**
  - Decision: Use the `cmdk` package for the palette UI.
  - Rationale: Small, unstyled, React-native, widely adopted. Avoids building a custom palette from scratch.

- **Panel resize via CSS + pointer events.**
  - Decision: Implement drag-to-resize using a thin handle element with `pointerdown`/`pointermove`/`pointerup` listeners that adjust a CSS height value.
  - Rationale: Zero-dependency, performant, matches VS Code's bottom panel pattern.

- **Sidebar toggle state in React context.**
  - Decision: Store sidebar open/closed state in a layout context provider. Persist in localStorage.
  - Rationale: Shell and pages need to read sidebar state for layout. localStorage survives reload.

## Risks / Trade-offs

- [ghostty-web WASM init failure] -> Mitigation: auto-fallback to xterm.js with console warning.
- [Terminal resize flicker during panel drag] -> Mitigation: debounce resize messages, use `requestAnimationFrame`.
- [cmdk styling conflicts with monochrome theme] -> Mitigation: cmdk is unstyled by default; we apply our own classes.
- [Keyboard shortcut conflicts with terminal input] -> Mitigation: only capture shortcuts when terminal is not focused, or use Cmd prefix (not captured by terminal).

## Open Questions

- None blocking. Engine selection toggle UI (archived 003-06) can be re-added as a follow-up if desired.
