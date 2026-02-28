## ADDED Requirements

### Requirement: Workspace Session List Provides Triage Signals
The web UI SHALL display a Workspace Session list that supports fast triage when many sessions exist.

#### Scenario: User sees session start time and newest-first ordering
- **WHEN** the Workspace Sessions page loads
- **THEN** each session row shows a Started timestamp
- **AND** sessions are ordered newest-first by Started timestamp

#### Scenario: User manually refreshes the session list
- **WHEN** the user clicks Refresh on the Workspace Sessions table header
- **THEN** the UI performs an immediate reload of the session list

### Requirement: Workspace Session Termination Action
The web UI SHALL allow users to terminate a Workspace Session from the Workspace Sessions list when the session lifecycle phase is `Active`.

#### Scenario: User kills an active session from the list
- **WHEN** the user clicks Kill on an Active session and confirms
- **THEN** the UI calls the control-plane API to delete the workspace session
- **AND** the session transitions out of Active (for example to Terminating) or disappears from the list

### Requirement: Settings Page For API Authentication
The web UI SHALL provide a Settings page for configuring the bearer token used for API authentication.

#### Scenario: User navigates to Settings
- **WHEN** the user selects Settings from the sidebar
- **THEN** the UI navigates to the Settings route and shows token controls

#### Scenario: Token entry is not embedded in the Topbar
- **WHEN** any page Topbar is visible
- **THEN** token entry controls are not rendered in the Topbar
