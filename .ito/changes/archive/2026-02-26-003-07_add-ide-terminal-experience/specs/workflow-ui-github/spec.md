## MODIFIED Requirements

### Requirement: Session and Run Workflow Console

The system SHALL provide a web console for creating sessions, starting runs, and viewing run lifecycle state across workflow routes, and SHALL present these routes with a consistent monochrome-dark interface using shared UI primitives.

The console SHALL provide IDE-like navigation with a collapsible sidebar, command palette, and keyboard shortcuts for efficient power-user workflows.

#### Scenario: User creates a session and starts a run

- **WHEN** an authorized user submits repository and run inputs from the web console
- **THEN** the UI shows the created session, created run, and current lifecycle state without requiring backend/API changes

#### Scenario: Workflow routes share consistent visual and interaction primitives

- **WHEN** a user navigates between workspace sessions, harness runs, and run detail routes
- **THEN** the routes use consistent shell navigation, typography hierarchy, and shared form/table/action components

#### Scenario: Console supports IDE-like navigation

- **WHEN** a user opens the command palette or uses keyboard shortcuts
- **THEN** the user can navigate between routes, toggle the sidebar, and access terminal controls without using the mouse
