<!-- ITO:START -->
## Why
AI coding assistants (OpenCode, Claude Code, Codex) need to manage remote sandbox agents programmatically — starting agents on the cluster, sending prompts, monitoring status, and stopping sessions — without the user manually running CLI commands. An agent skill bridges this gap by providing natural-language triggers and structured workflows that these tools can invoke automatically. This enables multi-agent workflows where a local AI assistant delegates tasks to remote sandbox agents running on the Kocao cluster.

## What Changes
- Create a new skill at `.opencode/skills/kocao-agent/` (or `.agents/skills/kocao-agent/`) that wraps the `kocao agent` CLI commands.
- Write `SKILL.md` with natural language trigger descriptions so the skill activates on relevant user requests (e.g., "start an agent on the cluster", "check agent status", "send a task to a remote agent").
- Include wrapper scripts in `scripts/` that call `kocao agent` subcommands with appropriate argument mapping.
- Provide reference documentation and example workflows for multi-agent use cases.
- Include a Showboat demo document demonstrating an AI assistant managing a remote agent end-to-end.

## Capabilities

### New Capabilities
- `agent-management-skill`: AI skill enabling coding assistants to manage remote Kocao sandbox agents through natural language.

### Modified Capabilities
- None. The skill is a new artifact that wraps the existing CLI.

## Impact
- Affected code: `.opencode/skills/kocao-agent/` (new directory with SKILL.md, scripts, references).
- Dependencies: requires the `kocao` CLI binary to be installed and configured (KOCAO_API_URL, KOCAO_TOKEN).
- Operations: no deployment changes — skill is loaded by AI coding tools at runtime.
- Breaking changes: none. Purely additive.

## Scope

### In Scope
- Skill scaffold: `SKILL.md`, `scripts/`, `reference/`
- CLI wrapper scripts for: list, start, stop, logs, exec, status
- Natural language trigger descriptions for skill activation
- Example workflows: single-agent task delegation, multi-agent coordination
- Showboat demo document

### Out of Scope
- Direct API calls (skill wraps CLI, not HTTP)
- CLI implementation (that's 007-02)
- Web UI integration
- Skill distribution/packaging beyond the repository

## Success Criteria
- Skill loads correctly in OpenCode and Claude Code.
- All trigger descriptions activate the skill on relevant user requests.
- Wrapper scripts correctly invoke `kocao agent` subcommands and return structured output.
- Multi-agent workflow example demonstrates end-to-end task delegation.
- Showboat demo runs successfully.
<!-- ITO:END -->
