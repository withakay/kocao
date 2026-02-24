## MODIFIED Requirements

### Requirement: Session and Run Workflow Console
The system SHALL provide a web console for creating sessions, starting runs, and viewing run lifecycle state, and SHALL serve this console from a cluster endpoint fronted by the control-plane pod web edge.

#### Scenario: User creates a session and starts a run
- **WHEN** an authorized user submits repository and run inputs from the web console
- **THEN** the UI shows the new session, created run, and current lifecycle state

#### Scenario: Cluster endpoint serves the workflow console
- **WHEN** an operator accesses the deployed control-plane endpoint in-cluster (or via configured ingress)
- **THEN** the workflow console is served without requiring a separate web deployment

## ADDED Requirements

### Requirement: Scalar OpenAPI Docs from Live Spec
The system SHALL serve a Scalar API documentation UI from the same control-plane pod web edge, and Scalar MUST load the API definition from the live `/openapi.json` endpoint.

#### Scenario: Scalar loads live OpenAPI
- **WHEN** a user opens `/scalar`
- **THEN** Scalar fetches `/openapi.json` from the deployed control-plane endpoint and renders the current API schema

### Requirement: Unified Edge Routing for UI and API
The system SHALL use a single web edge route surface that serves UI/docs content and proxies API plus websocket traffic to the control-plane API container.

#### Scenario: API request is proxied through edge
- **WHEN** a client calls an API route under `/api/v1/*` through the web edge endpoint
- **THEN** the request is forwarded to the control-plane API and returns the API response

#### Scenario: Attach websocket upgrade passes through edge
- **WHEN** a browser initiates an attach websocket connection through the web edge endpoint
- **THEN** the edge forwards the websocket upgrade and stream to the control-plane API without breaking attach behavior

### Requirement: Optional Tailscale Front Door Plan
The system SHALL define an optional Tailscale integration plan for the web edge endpoint, including deployment shape, auth/network boundaries, and rollout checklist, without requiring immediate production enablement.

#### Scenario: Tailscale remains disabled by default
- **WHEN** the default deployment is installed
- **THEN** Tailscale integration is not enabled unless the operator applies the optional integration configuration
