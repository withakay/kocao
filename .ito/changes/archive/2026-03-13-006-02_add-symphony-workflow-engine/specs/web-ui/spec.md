## ADDED Requirements

### Requirement: Symphony Worker Telemetry View
The web UI SHALL present the richer Symphony worker telemetry required to inspect workflow-driven execution, including recent worker failures and aggregate execution counters.

#### Scenario: User inspects workflow-driven worker state
- **WHEN** the Symphony detail page loads for a project with active or recently failed agent work
- **THEN** the UI shows bounded worker/session telemetry, recent failure context, and aggregate execution counters
