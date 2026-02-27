## ADDED Requirements

### Requirement: Cluster Operations Dashboard Route
The workflow UI SHALL provide a cluster dashboard route with namespace-level health and runtime state for operators.

#### Scenario: Operator opens cluster route
- **WHEN** a user navigates to `/cluster`
- **THEN** the UI shows pod/deployment status and key namespace metrics for the active control-plane namespace

### Requirement: In-UI Pod Log Inspection
The workflow UI SHALL allow operators to inspect recent pod logs without leaving the dashboard.

#### Scenario: Operator inspects pod logs
- **WHEN** the user selects a pod and loads logs from the cluster dashboard
- **THEN** the UI renders recent logs with container selection and bounded tail length controls
