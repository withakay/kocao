## ADDED Requirements

### Requirement: Token Persistence Is Opt-In
The UI SHALL NOT persist API tokens across browser restarts by default.

#### Scenario: Default token storage
- **WHEN** a user enters an API token in the UI
- **THEN** the token is stored only for the current browser session unless the user explicitly opts in to persistence

### Requirement: No Secrets in URLs
The UI SHALL avoid placing secrets (API tokens, attach tokens) in URLs.

#### Scenario: Attach does not include token in URL
- **WHEN** a user opens an attach session
- **THEN** the websocket connection is established without a token in the URL query string

### Requirement: Auth Failure Handling
The UI SHALL handle authorization failures safely.

#### Scenario: Invalid token is cleared
- **WHEN** the API returns HTTP 401 for an authenticated request
- **THEN** the UI prompts the user to re-authenticate and clears any stored token
