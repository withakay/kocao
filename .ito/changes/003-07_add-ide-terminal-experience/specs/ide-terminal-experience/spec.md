## ADDED Requirements

### Requirement: Real terminal emulation in attach UI

The attach UI SHALL render terminal output using a real terminal emulator (ghostty-web primary, xterm.js fallback) instead of plain text appending.

Terminal input from the emulator SHALL be sent as base64-encoded stdin messages over the active WebSocket connection.

Terminal output (base64-encoded stdout from WebSocket) SHALL be decoded and written to the terminal emulator.

#### Scenario: Terminal renders stdout with escape sequences

- **WHEN** the backend sends ANSI-colored output over the attach WebSocket
- **THEN** the terminal emulator renders colors, cursor positioning, and control sequences correctly

#### Scenario: User types in terminal and input reaches backend

- **WHEN** a driver-mode user types in the terminal emulator
- **THEN** keystrokes are sent as base64-encoded stdin messages over the WebSocket

#### Scenario: ghostty-web fails to initialize and xterm.js is used

- **WHEN** ghostty-web WASM initialization fails
- **THEN** the attach UI falls back to xterm.js without user intervention

### Requirement: Terminal engine adapter interface

The attach UI SHALL use a renderer-agnostic adapter interface to decouple transport logic from terminal engine internals.

The adapter interface SHALL support mount, write, resize, dispose, onInput, and dimensions operations.

#### Scenario: Engine swap does not require transport changes

- **WHEN** a different terminal engine adapter is configured
- **THEN** the WebSocket transport, auth, and message handling remain unchanged

### Requirement: Terminal resize synchronization

The attach UI SHALL send WebSocket `resize` messages with current cols and rows when terminal dimensions change due to panel resize, window resize, or fullscreen transitions.

#### Scenario: Panel resize updates backend PTY dimensions

- **WHEN** the user drags the terminal panel resize handle
- **THEN** a `resize` message with updated cols/rows is sent to the backend WebSocket

### Requirement: Resizable terminal panel

The attach UI SHALL provide a resizable terminal panel with a drag handle, similar to VS Code's bottom panel.

The panel SHALL support expand (fill available space), collapse (minimum height), and fullscreen (overlay entire viewport) modes.

#### Scenario: User drags panel resize handle

- **WHEN** the user drags the resize handle above the terminal panel
- **THEN** the panel height adjusts smoothly and the terminal re-fits to the new dimensions

#### Scenario: User enters fullscreen terminal mode

- **WHEN** the user activates fullscreen mode via button or keyboard shortcut
- **THEN** the terminal fills the entire viewport with a minimal status bar

### Requirement: Command palette

The attach UI SHALL provide a command palette accessible via Cmd+K (or Ctrl+K on non-Mac) that lists navigation actions, terminal controls, and common operations.

#### Scenario: User opens command palette with keyboard shortcut

- **WHEN** the user presses Cmd+K
- **THEN** a searchable command palette overlay appears with available actions

#### Scenario: User navigates to a page via command palette

- **WHEN** the user selects a navigation action (e.g., "Go to Sessions") from the palette
- **THEN** the app navigates to the selected route and the palette closes

### Requirement: Toggleable sidebar

The Shell sidebar SHALL be collapsible via a Cmd+\ keyboard shortcut (or Ctrl+\ on non-Mac) and a toggle button.

The sidebar state SHALL be persisted in localStorage so it survives page reload.

#### Scenario: User collapses sidebar with keyboard shortcut

- **WHEN** the user presses Cmd+\
- **THEN** the sidebar collapses with a smooth transition and the main content area expands

#### Scenario: Sidebar state persists across reloads

- **WHEN** the user collapses the sidebar and reloads the page
- **THEN** the sidebar remains collapsed

### Requirement: Keyboard shortcuts system

The attach UI SHALL support the following keyboard shortcuts:

- Cmd+K: Open command palette
- Cmd+\: Toggle sidebar
- Escape: Close command palette or exit fullscreen terminal

Shortcuts using Cmd SHALL use Ctrl on non-Mac platforms.

#### Scenario: Shortcuts work across all pages

- **WHEN** the user presses Cmd+K on any page
- **THEN** the command palette opens regardless of the current route
