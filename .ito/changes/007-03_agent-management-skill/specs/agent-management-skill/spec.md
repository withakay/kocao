<!-- ITO:START -->
## ADDED Requirements

### Requirement: Skill Scaffold and Activation
The skill SHALL be located at `.opencode/skills/kocao-agent/` and SHALL include a `SKILL.md` file with natural language trigger descriptions that cause the skill to activate when a user requests remote agent management operations from an AI coding assistant.

- **Requirement ID**: agent-management-skill:skill-scaffold-and-activation

#### Scenario: Skill activates on agent management request
- **WHEN** a user asks their AI coding assistant to "start an agent on the cluster" or "check remote agent status" or "send a task to a Kocao agent"
- **THEN** the AI tool loads the skill's `SKILL.md` and uses its workflows and scripts to fulfill the request

#### Scenario: Skill is discoverable via skill listing
- **WHEN** a user or tool lists available skills
- **THEN** the `kocao-agent` skill appears with a description indicating it manages remote Kocao sandbox agents

### Requirement: CLI Wrapper Scripts
The skill SHALL include executable scripts in `scripts/` that wrap each `kocao agent` CLI subcommand. Each script SHALL accept arguments, invoke the corresponding CLI command, and return structured output suitable for AI tool consumption.

- **Requirement ID**: agent-management-skill:cli-wrapper-scripts

#### Scenario: Wrapper script invokes CLI and returns output
- **WHEN** the AI tool executes a wrapper script (e.g., `scripts/agent-list.sh`)
- **THEN** the script calls `kocao agent list --output json` and returns the JSON output to the AI tool

#### Scenario: Wrapper script handles CLI errors
- **WHEN** the `kocao` CLI returns a non-zero exit code
- **THEN** the wrapper script captures stderr, returns the error message, and exits with a non-zero code so the AI tool can report the failure to the user

### Requirement: Workflow Documentation
The skill's `SKILL.md` SHALL include documented workflows for common use cases: starting an agent, sending a prompt, monitoring status, stopping an agent, and multi-agent coordination patterns.

- **Requirement ID**: agent-management-skill:workflow-documentation

#### Scenario: AI tool follows documented workflow for starting an agent
- **WHEN** a user asks to "start a Claude agent for this repo"
- **THEN** the AI tool follows the documented workflow: (1) call the start script with repo URL and agent name, (2) wait for ready confirmation, (3) report the session ID and status to the user

#### Scenario: Multi-agent workflow is documented
- **WHEN** a user asks to "start a Codex agent to review this PR"
- **THEN** the AI tool follows the multi-agent workflow: (1) start an agent with the repo and PR context, (2) send a prompt with the review task, (3) stream events until the review is complete, (4) present the results

### Requirement: Reference Documentation
The skill SHALL include reference files in `reference/` documenting the supported agents, required environment variables (`KOCAO_API_URL`, `KOCAO_TOKEN`), CLI prerequisites, and troubleshooting guidance.

- **Requirement ID**: agent-management-skill:reference-documentation

#### Scenario: AI tool checks prerequisites before invoking CLI
- **WHEN** the AI tool loads the skill
- **THEN** it can consult the reference documentation to verify that `kocao` is installed and `KOCAO_API_URL`/`KOCAO_TOKEN` are set before attempting operations

### Requirement: Showboat Demo Artifact
The skill SHALL include a Showboat demo document demonstrating an end-to-end workflow where an AI assistant manages a remote Kocao sandbox agent.

- **Requirement ID**: agent-management-skill:showboat-demo-artifact

#### Scenario: Demo document is executable and reproducible
- **WHEN** the Showboat demo is executed
- **THEN** it demonstrates: starting an agent, sending a prompt, viewing events, checking status, and stopping the agent — with annotated output at each step
<!-- ITO:END -->
