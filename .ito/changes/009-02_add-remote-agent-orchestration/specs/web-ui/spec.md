<!-- ITO:START -->
## ADDED Requirements

### Requirement: Remote Agent Operations Dashboard

The web UI SHALL expose a dashboard for active remote agents, current tasks, and recent outcomes.

- **Requirement ID**: web-ui:remote-agent-operations-dashboard

#### Scenario: Operator views active agents

- **WHEN** an operator opens the remote-agent dashboard
- **THEN** the UI shows active agents, current task, lifecycle phase, last activity time, and current pool or team association

#### Scenario: Operator inspects task results

- **WHEN** an operator drills into a completed or failed task
- **THEN** the UI shows transcript summary, artifact references, and terminal status information
<!-- ITO:END -->
