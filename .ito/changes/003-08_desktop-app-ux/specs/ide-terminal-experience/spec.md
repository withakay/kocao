<!-- ITO:START -->
## MODIFIED Requirements

### Requirement: Terminal panel is user-resizable via drag handle

The attach page terminal panel SHALL be resizable by dragging a handle between the connection info bar and the terminal. The terminal SHALL refit to new dimensions on resize. Panel size SHALL persist in localStorage.

#### Scenario: User drags the terminal panel resize handle

- **WHEN** a user drags the resize handle between connection info and terminal
- **THEN** the panel resizes smoothly, the terminal refits, and the new size persists across page reloads

### Requirement: Command palette can toggle terminal fullscreen

The command palette SHALL include a "Toggle Fullscreen Terminal" action when on the attach page that toggles the terminal between normal and fullscreen mode via shared context.

#### Scenario: User triggers fullscreen via command palette

- **WHEN** a user opens Cmd+K on the attach page and selects "Toggle Fullscreen Terminal"
- **THEN** the terminal enters or exits fullscreen mode

### Requirement: Command palette uses Cursor-style dark aesthetic

The command palette SHALL use a dark glassmorphic design with near-black background, backdrop blur, category icons, keyboard shortcut badges, and polished typography matching the Cursor IDE aesthetic.

#### Scenario: User opens command palette

- **WHEN** a user presses Cmd+K
- **THEN** the palette renders with a dark glassmorphic overlay, searchable actions with icons, and keyboard hint badges

### Requirement: Sidebar is resizable via drag handle

The sidebar SHALL be resizable by dragging its right edge between 180px minimum and 320px maximum width. The width SHALL persist in localStorage.

#### Scenario: User drags sidebar edge

- **WHEN** a user drags the sidebar right edge
- **THEN** the sidebar width changes smoothly within min/max bounds and the width persists across reloads
<!-- ITO:END -->
