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

### Requirement: Workspace Session Responses Include Creation Timestamp
The API SHALL include a creation timestamp in workspace session responses so clients can order sessions by recency.

#### Scenario: Client lists workspace sessions
- **WHEN** an authorized client calls `GET /api/v1/workspace-sessions`
- **THEN** each returned session includes a `createdAt` field in RFC3339 format

#### Scenario: Client fetches a workspace session
- **WHEN** an authorized client calls `GET /api/v1/workspace-sessions/{workspaceSessionID}`
- **THEN** the returned session includes a `createdAt` field in RFC3339 format

### Requirement: Workspace Session Delete Endpoint
The API SHALL allow authorized clients to delete a workspace session.

#### Scenario: Client deletes a workspace session
- **WHEN** an authorized client calls `DELETE /api/v1/workspace-sessions/{workspaceSessionID}`
- **THEN** the API deletes the Session resource in the namespace
- **AND** the API returns a success response indicating the delete request was accepted
