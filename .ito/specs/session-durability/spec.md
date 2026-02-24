## ADDED Requirements

### Requirement: Workspace Session Is the Durable Workspace Anchor
The system SHALL define Workspace Session as the durable unit that anchors repository identity, workspace storage, and policy context across multiple Harness Runs.

#### Scenario: Multiple runs share one workspace session context
- **WHEN** multiple Harness Runs are created under one Workspace Session
- **THEN** each run uses the same workspace and policy anchor defined by that Workspace Session

### Requirement: Qualified Lifecycle Labels Across Surfaces
The system SHALL use object-qualified lifecycle labels and MUST NOT present bare `lifecycle` in API, UI, or documentation surfaces.

#### Scenario: Lifecycle label shown for run state
- **WHEN** lifecycle state is presented for a Harness Run
- **THEN** the label is `Harness Run Lifecycle`

#### Scenario: Lifecycle label shown for workspace state
- **WHEN** lifecycle state is presented for a Workspace Session
- **THEN** the label is `Workspace Session Lifecycle`

### Requirement: Hard-Cutover Terminology Contract
The system SHALL replace legacy session/run naming with Workspace Session and Harness Run terminology as the only supported contract vocabulary.

#### Scenario: Legacy terminology is no longer accepted in contract surfaces
- **WHEN** clients rely on deprecated session/run/lifecycle naming in contract-bound API or UI integration points
- **THEN** only the renamed Workspace Session and Harness Run contract vocabulary is supported
