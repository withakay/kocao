## ADDED Requirements

### Requirement: Symphony Token Secret Management
The API SHALL accept an optional write-only GitHub token field for Symphony project create and update requests and SHALL create or update the backing Kubernetes Secret automatically.

#### Scenario: Create request includes GitHub token
- **WHEN** an authorized client creates a Symphony project with a GitHub token value
- **THEN** the API creates or updates the derived Secret, sets `spec.source.tokenSecretRef` to that Secret name, and does not echo the raw token in the response

#### Scenario: Update request rotates GitHub token
- **WHEN** an authorized client updates an existing Symphony project with a new GitHub token value
- **THEN** the API updates the derived Secret contents and preserves the same deterministic Secret reference

#### Scenario: Secret-name field contains a PAT-shaped value
- **WHEN** a create or update request provides a PAT-looking string where a Secret name is expected
- **THEN** the API rejects the request with a validation error that does not echo the raw token value
