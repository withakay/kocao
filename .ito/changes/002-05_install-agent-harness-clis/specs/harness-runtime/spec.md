<!-- ITO:START -->
## ADDED Requirements

### Requirement: Pre-Installed Agent CLI Binaries
The harness image SHALL include the coding agent CLI binaries pre-installed and available on PATH so that harness runs can invoke agents without per-run bootstrapping.

#### Scenario: Claude Code CLI available
- **WHEN** a harness pod starts from the published image
- **THEN** `claude --version` succeeds and reports the expected version

#### Scenario: OpenCode CLI available
- **WHEN** a harness pod starts from the published image
- **THEN** `opencode version` succeeds and reports the expected version

#### Scenario: OpenAI Codex CLI available
- **WHEN** a harness pod starts from the published image
- **THEN** `codex --version` succeeds and reports the expected version

#### Scenario: Smoke test validates agent CLIs
- **WHEN** the smoke test runs against the built image
- **THEN** all three agent CLIs are validated as present and reporting correct versions
<!-- ITO:END -->
