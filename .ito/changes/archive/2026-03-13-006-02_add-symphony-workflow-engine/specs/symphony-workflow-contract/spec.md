## ADDED Requirements

### Requirement: Repository Workflow File Discovery
The system SHALL load Symphony workflow policy from a repository-owned `WORKFLOW.md` file, using an explicit configured path when present and the repository root `WORKFLOW.md` file otherwise.

#### Scenario: Default workflow path is used
- **WHEN** a claimed repository does not override the workflow file path
- **THEN** the worker loads `WORKFLOW.md` from the repository root before execution

### Requirement: Workflow Front Matter Parsing
The workflow contract SHALL parse optional YAML front matter into a configuration map and treat the remaining markdown body as the prompt template.

#### Scenario: Invalid front matter blocks execution
- **WHEN** the repository workflow file contains invalid YAML or a non-map front matter value
- **THEN** the worker attempt fails with a typed workflow validation error

### Requirement: Strict Prompt Rendering
The workflow contract SHALL render prompts with strict variable and filter checking using normalized issue data and retry metadata.

#### Scenario: Unknown template variable fails the attempt
- **WHEN** a workflow prompt references an unknown variable
- **THEN** the worker attempt fails with a template render error instead of silently continuing

### Requirement: Workflow Config Reload
The system SHALL re-read workflow contract changes for future worker attempts without requiring a control-plane restart.

#### Scenario: Updated workflow applies to a later retry
- **WHEN** a repository `WORKFLOW.md` file changes before the next worker attempt for the same issue
- **THEN** the next attempt uses the updated workflow config and prompt template
