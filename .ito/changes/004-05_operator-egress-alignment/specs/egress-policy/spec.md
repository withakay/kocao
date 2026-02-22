## ADDED Requirements

### Requirement: Egress Modes Are Enforced by NetworkPolicy
The system SHALL enforce run egress mode using Kubernetes NetworkPolicy.

#### Scenario: Restricted mode denies by default
- **WHEN** a run is created with egress mode `restricted`
- **THEN** egress is denied by default except for DNS and admin-configured GitHub CIDRs

#### Scenario: Full mode allows all
- **WHEN** a run is created with egress mode `full`
- **THEN** egress is allowed and the override is auditable

### Requirement: Unsupported Egress Overrides Are Rejected
The system SHALL reject egress override parameters that are not enforced by the operator.

#### Scenario: allowedHosts is rejected
- **WHEN** a client requests an egress override containing `allowedHosts`
- **THEN** the API responds with HTTP 400 and a message that host-based egress allowlisting is not supported

### Requirement: GitHub CIDR Allowlist Validation
The system SHALL validate configured GitHub CIDRs and surface invalid configuration clearly.

#### Scenario: Invalid CIDR is detected
- **WHEN** `CP_GITHUB_EGRESS_CIDRS` contains an invalid CIDR
- **THEN** the system logs/records an explicit error and does not silently ignore the entry
