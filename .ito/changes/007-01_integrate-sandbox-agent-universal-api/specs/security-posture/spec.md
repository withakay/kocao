<!-- ITO:START -->
## ADDED Requirements

### Requirement: Sandbox Agent Access Is Control-Plane Mediated
The system SHALL keep sandbox-agent behind Kocao's control-plane authn/authz boundary and SHALL NOT require end users or browsers to connect directly to the in-pod sandbox-agent server.

- **Requirement ID**: security-posture:sandbox-agent-access-is-control-plane-mediated

#### Scenario: Browser interacts through Kocao rather than direct pod access
- **WHEN** a user interacts with a sandbox-backed session from the UI
- **THEN** the browser communicates only with Kocao-managed endpoints and Kocao proxies authorized calls to the in-pod sandbox-agent server

### Requirement: Provider Credentials Remain Pod-Scoped
The system SHALL keep upstream provider credentials and agent auth material scoped to the selected harness pod and SHALL NOT return them through API responses, UI payloads, or persisted transcript events.

- **Requirement ID**: security-posture:provider-credentials-remain-pod-scoped

#### Scenario: Client inspects session metadata
- **WHEN** an authorized client fetches sandbox-backed session details or events
- **THEN** the response excludes raw provider credentials, auth files, and secret values while still exposing useful lifecycle diagnostics

## MODIFIED Requirements

### Requirement: Workflow and Agent Secrets Stay Bounded
The system SHALL keep workflow and agent-related secrets bounded to the minimum control-plane or harness-pod surface that needs them, including sandbox-agent-backed execution paths.

- **Requirement ID**: security-posture:workflow-and-agent-secrets-stay-bounded

#### Scenario: Worker telemetry remains secret-safe
- **WHEN** Kocao records telemetry or status for sandbox-backed or workflow-driven execution
- **THEN** secret-shaped values are redacted before persistence or API exposure
<!-- ITO:END -->
