## MODIFIED Requirements

### Requirement: Session and Run Workflow Console
The system SHALL provide a web console for creating sessions, starting runs, and viewing run lifecycle state across workflow routes, and SHALL present these routes with a consistent monochrome-dark interface using Kibo UI components wherever equivalent primitives exist.

#### Scenario: User creates a session and starts a run
- **WHEN** an authorized user submits repository and run inputs from the web console
- **THEN** the UI shows the created session, created run, and current lifecycle state without requiring backend/API changes

#### Scenario: Workflow routes share consistent visual and interaction primitives
- **WHEN** a user navigates between workspace sessions, harness runs, and run detail routes
- **THEN** the routes use consistent shell navigation, typography hierarchy, and Kibo-based form/table/action components

## ADDED Requirements

### Requirement: Monochrome Dark Operator Visual System
The system SHALL use a dark monochrome visual system for workflow UI surfaces, and SHALL limit accent usage to a single subtle accent used for priority actions and focus states.

#### Scenario: Operator scans status-dense pages
- **WHEN** a user views workflow pages containing lifecycle states, tables, and action controls
- **THEN** contrast and hierarchy are sufficient to distinguish primary actions, status signals, and secondary metadata in the monochrome theme

### Requirement: Technical Operator Copy Tone
The system SHALL use concise, technical copy for workflow labels, helper text, and action descriptions, and SHALL avoid generic marketing-oriented language.

#### Scenario: User reads route labels and helper text
- **WHEN** the user reviews page titles, button labels, and inline guidance
- **THEN** wording is direct, implementation-aware, and consistent with an expert operator audience
