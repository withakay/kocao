<!-- ITO:START -->
## ADDED Requirements

### Requirement: Attach UI supports interchangeable terminal engines

The attach UI SHALL provide a per-session engine selector with `xterm.js` and `ghostty-web` options.

`ghostty-web` SHALL be labeled experimental in the selector.

#### Scenario: User selects terminal engine in attach session

- **WHEN** a user opens an attach session
- **THEN** the UI shows a terminal engine selector for `xterm.js` and `ghostty-web (experimental)`

### Requirement: Engine switching is immediate in the active session

The attach UI SHALL apply terminal engine changes immediately in the current session without requiring users to open a new session.

Switching engines SHALL preserve the active attach transport connection and continue streaming terminal output.

#### Scenario: User hot-switches renderer while attached

- **WHEN** a connected user switches from `xterm.js` to `ghostty-web` (or back)
- **THEN** the UI updates rendering immediately and the same attach session continues receiving output

### Requirement: Per-session engine choice is persisted in a cookie

The UI SHALL persist the selected terminal engine in a browser cookie scoped to the workspace session so the choice is restored after reload.

#### Scenario: Reload restores engine choice for same session

- **WHEN** a user selects `ghostty-web` for workspace session `A` and reloads the attach page for session `A`
- **THEN** the UI restores `ghostty-web` for session `A` from cookie state

### Requirement: Transport behavior is engine-agnostic

Terminal engine selection SHALL NOT change attach protocol messages, authentication flow, or backend attach semantics.

#### Scenario: Engine choice does not alter attach protocol

- **WHEN** a run is attached with either supported engine
- **THEN** the UI uses the same attach websocket endpoint and protocol message shapes
<!-- ITO:END -->
