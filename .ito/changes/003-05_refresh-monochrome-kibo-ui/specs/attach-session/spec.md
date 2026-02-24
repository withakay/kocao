## ADDED Requirements

### Requirement: Attach Console Visual Parity
The system SHALL render the browser attach interface with the same monochrome-dark design language and Kibo-based control surfaces used in the main workflow console, while preserving existing attach protocol behavior.

#### Scenario: Driver attach flow uses refreshed console styling
- **WHEN** a user opens the attach page in driver mode
- **THEN** terminal, input, and action controls follow the shared monochrome styling and component patterns used across other workflow routes

#### Scenario: Viewer role remains read-only with explicit affordance
- **WHEN** a user opens the attach page in viewer mode
- **THEN** input and send controls remain disabled and visually indicate read-only state in the refreshed UI
