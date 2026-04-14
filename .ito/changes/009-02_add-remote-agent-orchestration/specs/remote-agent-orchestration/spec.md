<!-- ITO:START -->
## ADDED Requirements

### Requirement: Named Remote Agents

The system SHALL support named remote agents, teams, or pools that abstract over raw harness run IDs.

- **Requirement ID**: remote-agent-orchestration:named-remote-agents

#### Scenario: Operator targets a named agent

- **WHEN** a user dispatches work to a named remote agent
- **THEN** the system resolves that logical identity to the appropriate live or newly provisioned agent session without requiring the user to provide a raw run ID

### Requirement: Task Dispatch Lifecycle

The system SHALL support explicit task dispatch with task identity, state, retry, timeout, and cancellation semantics.

- **Requirement ID**: remote-agent-orchestration:task-dispatch-lifecycle

#### Scenario: Task is assigned and completed

- **WHEN** a user dispatches a task to a remote agent
- **THEN** the system records the task, tracks it through assignment and completion states, and returns a durable result reference

#### Scenario: Task times out or is cancelled

- **WHEN** a task exceeds its timeout or is cancelled by an operator
- **THEN** the orchestration state reflects that terminal outcome and the remote agent is not left in an ambiguous running state

### Requirement: Multi-Agent Workflow Coordination

The system SHALL support coordinating multiple remote agents in one workflow.

- **Requirement ID**: remote-agent-orchestration:multi-agent-workflow-coordination

#### Scenario: Reviewer and implementer are coordinated

- **WHEN** a workflow dispatches implementation work to one agent and review work to another
- **THEN** the system preserves task boundaries, associated outputs, and current status for both agents independently
<!-- ITO:END -->
