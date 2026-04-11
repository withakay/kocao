## MODIFIED Requirements

### Requirement: Symphony Runtime Detail Endpoint
The API SHALL provide per-project runtime state including active items, retry queue entries, linked Session/HarnessRun identities, recent errors, recent session events, and aggregate execution counters.

#### Scenario: Client fetches Symphony runtime detail
- **WHEN** an authorized client requests Symphony project detail
- **THEN** the API returns bounded runtime state for the project, including worker/session telemetry needed to debug active or recently failed work
