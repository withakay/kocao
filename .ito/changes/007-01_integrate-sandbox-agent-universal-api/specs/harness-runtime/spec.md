<!-- ITO:START -->
## ADDED Requirements

### Requirement: Sandbox Agent Server Is Present in the Harness Runtime
The harness image SHALL include a pinned `sandbox-agent` binary or CLI wrapper and SHALL be able to start a sandbox-agent server inside the harness pod as part of the standard run contract.

- **Requirement ID**: harness-runtime:sandbox-agent-server-present

#### Scenario: Sandbox Agent binary is available in the image
- **WHEN** a harness pod starts from the published runtime image
- **THEN** `sandbox-agent --version` succeeds and the pinned version is part of the runtime contract

#### Scenario: Harness pod can start sandbox-agent server
- **WHEN** a sandbox-backed run starts
- **THEN** the harness pod starts `sandbox-agent server` successfully and exposes a healthy in-pod control surface for Kocao to use

### Requirement: Supported Agent Dependencies Are Available to Sandbox Agent
The harness runtime SHALL provide the dependencies and credential mount points needed for sandbox-agent to launch `opencode`, `claude`, `codex`, and `pi` sessions inside the pod.

- **Requirement ID**: harness-runtime:supported-agent-dependencies-available

#### Scenario: Sandbox Agent can launch a supported agent
- **WHEN** Kocao requests a supported agent session through sandbox-agent
- **THEN** the harness runtime has the required agent binary, configuration path, and credential injection surface available for that agent

### Requirement: Sandbox Agent Version and API Contract Are Verified
The harness runtime SHALL pin the sandbox-agent version used by Kocao and SHALL verify compatibility with the expected API contract during repository validation or image smoke testing.

- **Requirement ID**: harness-runtime:sandbox-agent-version-and-api-contract-are-verified

#### Scenario: Contract drift is detected before release
- **WHEN** the pinned sandbox-agent binary or API behavior changes incompatibly with Kocao's expected contract
- **THEN** repository validation or harness smoke testing fails before the image is treated as releasable

## MODIFIED Requirements

### Requirement: Reproducible Harness Runtime Image
The system SHALL provide a versioned harness image that includes the approved runtime matrix, required coding-agent tooling for MVP workloads, and the pinned `sandbox-agent` server dependency used for provider-neutral agent control. The image SHALL use `mise` for runtime version management, bake all runtimes specified in `mise.toml`, and include compilation prerequisites and developer utilities. The image SHALL be built from Ubuntu 24.04.

- **Requirement ID**: harness-runtime:reproducible-harness-runtime-image

#### Scenario: Harness pod starts with required toolchain and sandbox-agent availability
- **WHEN** a run pod starts from the published harness image
- **THEN** the required language runtimes, core CLI tools, supported agent binaries, and `sandbox-agent` are available and report expected versions
<!-- ITO:END -->
