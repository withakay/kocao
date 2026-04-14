<!-- ITO:START -->
## ADDED Requirements

### Requirement: Session Blocker Diagnostics

The system SHALL expose a machine-readable explanation for why an agent session is not ready.

- **Requirement ID**: agent-diagnostics:session-blocker-diagnostics

#### Scenario: Pod scheduling or image pull is blocking readiness

- **WHEN** a session remains in provisioning because the pod is unscheduled or still pulling an image
- **THEN** the diagnostic output identifies that blocker class explicitly

#### Scenario: Sandbox-agent is unreachable

- **WHEN** the harness pod is running but sandbox-agent is not yet reachable on its health endpoint
- **THEN** the diagnostic output identifies sandbox-agent readiness as the blocker

#### Scenario: Auth or network prerequisite is failing

- **WHEN** the session cannot initialize because credentials, repo access, or egress checks are failing
- **THEN** the diagnostic output identifies auth, repo, or network reachability as the blocker class

### Requirement: Operator and CLI Visibility

The diagnostic data SHALL be visible through the control-plane API and consumable by the CLI and demos.

- **Requirement ID**: agent-diagnostics:operator-and-cli-visibility

#### Scenario: CLI requests session status

- **WHEN** a user runs `kocao agent status <run-id>` for a non-ready session
- **THEN** the status output includes the current lifecycle phase and the most relevant blocker explanation
<!-- ITO:END -->
