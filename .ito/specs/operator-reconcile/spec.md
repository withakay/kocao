## ADDED Requirements

### Requirement: Session and Run Resource Contract
The system SHALL define Session and HarnessRun custom resources with desired-state inputs and observed-state outputs required to orchestrate a coding run.

#### Scenario: Valid run specification accepted
- **WHEN** a HarnessRun resource is created with required repository and runtime inputs
- **THEN** the Kubernetes API accepts the resource and exposes initialized status fields

### Requirement: Deterministic Reconciliation
The operator SHALL reconcile HarnessRun resources into run pod lifecycle actions and update resource status as lifecycle phases change.

#### Scenario: Run progresses from scheduling to completion
- **WHEN** reconcile observes pod creation and execution events for a valid HarnessRun
- **THEN** the HarnessRun status reflects ordered phase transitions and terminal outcome details
