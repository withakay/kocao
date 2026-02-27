## ADDED Requirements

### Requirement: Namespace Cluster Overview Endpoint
The API SHALL provide an authenticated namespace-scoped overview endpoint for kocao runtime state.

#### Scenario: User requests cluster overview
- **WHEN** an authorized client calls `GET /api/v1/cluster-overview`
- **THEN** the API returns namespace summary metrics, deployment status, pod status, and non-secret config indicators

### Requirement: Pod Log Tail Endpoint
The API SHALL provide an authenticated endpoint for retrieving bounded pod logs for namespace pods.

#### Scenario: User requests pod logs
- **WHEN** an authorized client calls `GET /api/v1/pods/{podName}/logs`
- **THEN** the API returns log text for the requested pod/container with bounded tail-line size and input validation
