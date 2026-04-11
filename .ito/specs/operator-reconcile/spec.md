## ADDED Requirements

### Requirement: Symphony Project Reconciliation
The operator SHALL reconcile Symphony Project resources and publish conditions for configuration validity, source sync health, and orchestration lifecycle.

#### Scenario: Valid Symphony project reports ready-to-sync status
- **WHEN** the operator reconciles a Symphony Project with valid configuration and credentials
- **THEN** the project status reports conditions that show the project is ready to poll and orchestrate work

### Requirement: Symphony Claims Materialize as Session and HarnessRun Children
The operator SHALL create and label child Session and HarnessRun resources for each claimed Symphony item, linking them back to the Symphony Project and source issue identity.

#### Scenario: Claimed item creates execution children
- **WHEN** the operator claims an eligible GitHub project item
- **THEN** it creates or selects the mapped Session and launches a child HarnessRun labeled with the Symphony Project and issue identifiers

### Requirement: Ineligible Item Reconciliation Stops or Releases Work
The operator SHALL stop or release active work when a tracked item becomes terminal, inactive, unsupported, or missing in the source system.

#### Scenario: Item leaves the active state set
- **WHEN** reconciliation finds that a tracked item is no longer in an active Symphony state
- **THEN** the operator stops or releases the associated execution and updates project status to reflect the reason
