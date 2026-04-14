<!-- ITO:START -->
## ADDED Requirements

### Requirement: Sandbox-Agent Compatibility Across Profiles

Every harness image profile SHALL preserve the sandbox-agent control surface required by module 007.

- **Requirement ID**: harness-runtime:sandbox-agent-compatibility-across-profiles

#### Scenario: Base profile still exposes sandbox-agent

- **WHEN** the minimal base profile is launched
- **THEN** sandbox-agent starts correctly and exposes the same health and control behavior as the full profile
<!-- ITO:END -->
