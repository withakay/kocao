## ADDED Requirements

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
