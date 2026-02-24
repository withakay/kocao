## ADDED Requirements

### Requirement: Session and Run Workflow Console
The system SHALL provide a web console for creating sessions, starting runs, and viewing run lifecycle state.

#### Scenario: User creates a session and starts a run
- **WHEN** an authorized user submits repository and run inputs from the web console
- **THEN** the UI shows the new session, created run, and current lifecycle state

### Requirement: GitHub PR Outcome Visibility
The system SHALL display GitHub branch and pull request outcome metadata associated with a completed run.

#### Scenario: Run produces a pull request
- **WHEN** a run completes with a GitHub pull request result
- **THEN** the run detail view displays the PR URL and reported PR status
