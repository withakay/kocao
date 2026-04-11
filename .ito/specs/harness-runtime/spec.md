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

### Requirement: API Key Injection via Environment Variables
The operator SHALL support injecting API keys from a Kubernetes Secret into harness pods as environment variables. This enables simple API key authentication for all three agent CLIs.

#### Scenario: API key Secret referenced by HarnessRun
- **GIVEN** a HarnessRun CR with `spec.agentAuth.apiKeySecretName` set to a Secret name
- **WHEN** the operator creates the harness pod
- **THEN** the Secret is injected via `envFrom` so all keys are available as env vars

#### Scenario: API key Secret not configured
- **GIVEN** a HarnessRun CR without `spec.agentAuth`
- **WHEN** the operator creates the harness pod
- **THEN** the pod starts normally without agent credential injection

#### Scenario: Supported API key environment variables
- **WHEN** the api-key Secret contains keys `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`, `GITHUB_TOKEN`, or `OPENROUTER_API_KEY`
- **THEN** each is available as an environment variable in the harness container

### Requirement: OAuth Token Injection via File Mounts
The operator SHALL support mounting OAuth token files from a Kubernetes Secret into harness pods at the file paths each agent CLI expects. This enables subscription-based authentication (Anthropic Pro/Max, OpenAI Plus/Pro, GitHub Copilot).

#### Scenario: OAuth Secret referenced by HarnessRun
- **GIVEN** a HarnessRun CR with `spec.agentAuth.oauthSecretName` set to a Secret name
- **WHEN** the operator creates the harness pod
- **THEN** the Secret keys are mounted as files at the expected paths with mode 0600

#### Scenario: OpenCode OAuth token file mounted
- **GIVEN** the oauth Secret contains key `opencode-auth.json`
- **WHEN** the operator mounts the Secret
- **THEN** the file is available at `/home/kocao/.local/share/opencode/auth.json` with mode 0600

#### Scenario: Codex CLI OAuth token file mounted
- **GIVEN** the oauth Secret contains key `codex-auth.json`
- **WHEN** the operator mounts the Secret
- **THEN** the file is available at `/home/kocao/.codex/auth.json` with mode 0600

#### Scenario: OAuth Secret not configured
- **GIVEN** a HarnessRun CR without `spec.agentAuth.oauthSecretName`
- **WHEN** the operator creates the harness pod
- **THEN** the pod starts normally without OAuth file mounts
<!-- ITO:END -->
