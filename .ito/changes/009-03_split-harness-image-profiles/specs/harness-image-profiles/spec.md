<!-- ITO:START -->
## ADDED Requirements

### Requirement: Profile-Based Harness Images

The system SHALL support a family of harness image profiles rather than a single monolithic runtime image.

- **Requirement ID**: harness-image-profiles:profile-based-harness-images

#### Scenario: Minimal base profile is available

- **WHEN** a workflow only needs the core sandbox-agent control surface and common tooling
- **THEN** the system can use a minimal base harness image instead of pulling the full all-runtimes image

#### Scenario: Full profile remains available

- **WHEN** a workflow explicitly requires the broadest runtime/tooling coverage
- **THEN** the system can still select a full profile that preserves current capability coverage

### Requirement: Deterministic Profile Selection

The system SHALL deterministically select an image profile based on explicit request, policy, or repo/task heuristics.

- **Requirement ID**: harness-image-profiles:deterministic-profile-selection

#### Scenario: Explicit image profile requested

- **WHEN** a user or policy specifies a profile
- **THEN** the run uses that profile and reports it in run/session status

#### Scenario: Profile inferred from task context

- **WHEN** no profile is specified
- **THEN** the system applies a documented default selection rule rather than choosing arbitrarily

### Requirement: Development Cluster Pre-Pull Support

The system SHALL support pre-pulling commonly used profiles onto development clusters.

- **Requirement ID**: harness-image-profiles:development-cluster-prepull-support

#### Scenario: Dev cluster primes common images

- **WHEN** a dev cluster is prepared for agent workflows
- **THEN** the common harness profiles can be pre-pulled so first-run latency is reduced
<!-- ITO:END -->
