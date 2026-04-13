<!-- ITO:START -->
## ADDED Requirements

### Requirement: Skill Trigger Conditions

The zoekt search skill SHALL be loaded when an agent needs to find code patterns, function definitions, type declarations, usages of a symbol, or navigate unfamiliar parts of the codebase. The skill MUST define clear trigger descriptions so OpenCode, Claude Code, and Codex can auto-load it.

- **Requirement ID**: zoekt-agent-skill:skill-trigger-conditions

#### Scenario: Agent needs structural code search

- **WHEN** an agent is asked to find all implementations of an interface, usages of a function, or files matching a code pattern
- **THEN** the skill is loaded and the agent uses `agent-zoekt search` instead of sequential grep/glob

#### Scenario: Agent explores unfamiliar codebase area

- **WHEN** an agent needs to understand a subsystem it has not read files from
- **THEN** the skill guides the agent to search for relevant types, functions, and packages before reading individual files

### Requirement: Search Workflow Guidance

The skill SHALL provide a structured workflow for when and how to use zoekt search, including query construction tips, result interpretation, and follow-up actions.

- **Requirement ID**: zoekt-agent-skill:search-workflow-guidance

#### Scenario: Skill provides query construction guidance

- **WHEN** the skill is loaded
- **THEN** the agent has access to guidance on constructing effective zoekt queries (literal strings, regex patterns, file filters, symbol searches)

#### Scenario: Skill provides result interpretation

- **WHEN** the agent receives JSONL search results
- **THEN** the skill provides guidance on interpreting fields (file path, line number, matched content, score) and how to use results for next steps

### Requirement: Index Freshness Awareness

The skill SHALL inform agents that the index may be stale if files have been edited since the last indexing run, and SHALL provide guidance on when to re-index.

- **Requirement ID**: zoekt-agent-skill:index-freshness-awareness

#### Scenario: Agent is warned about potential staleness

- **WHEN** the skill is loaded and the agent is about to search
- **THEN** the skill notes that results reflect the last index time and suggests running `agent-zoekt index` if recent edits may not be captured

### Requirement: Cross-Tool Portability

The skill SHALL be placed at `.agents/skills/zoekt-search/SKILL.md` so it is portable across Claude Code, OpenCode, and Codex without tool-specific configuration.

- **Requirement ID**: zoekt-agent-skill:cross-tool-portability

#### Scenario: Skill is loadable by OpenCode

- **WHEN** an OpenCode session starts in a repository containing `.agents/skills/zoekt-search/SKILL.md`
- **THEN** the skill appears in the available skills list and can be loaded on demand

#### Scenario: Skill is loadable by Claude Code

- **WHEN** a Claude Code session starts in a repository containing `.agents/skills/zoekt-search/SKILL.md`
- **THEN** the skill is available via the standard skill loading mechanism
<!-- ITO:END -->
