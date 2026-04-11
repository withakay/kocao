## ADDED Requirements

### Requirement: Workflow-Driven Worker Execution
The Symphony orchestrator SHALL prepare the per-issue workspace, load the repository workflow contract, render the task prompt, and run the agent session before deciding whether to retry or continue.

#### Scenario: Missing workflow prevents worker execution
- **WHEN** the claimed repository does not provide a valid workflow contract
- **THEN** the orchestrator records a workflow error for the issue and does not start agent turns for that attempt

### Requirement: Continuation Retry Semantics
The orchestrator SHALL schedule a short continuation retry after a clean worker exit so active issues can resume work in a new worker session when needed.

#### Scenario: Clean exit schedules continuation retry
- **WHEN** a worker finishes normally and the issue remains active
- **THEN** the orchestrator queues a short continuation retry instead of releasing the item immediately

### Requirement: Bounded Session Observability
The orchestrator SHALL retain bounded recent session details including current worker state, recent failures, token totals, and aggregate runtime counters.

#### Scenario: Operator requests detailed runtime state
- **WHEN** an operator inspects a Symphony project with active or recently failed work
- **THEN** the reported status includes bounded worker telemetry, recent errors, and aggregate runtime counters

## MODIFIED Requirements

### Requirement: Poll, Claim, Reconcile, and Retry Loop
The Symphony orchestrator SHALL poll eligible items on a fixed cadence, enforce per-project concurrency, avoid duplicate claims, and retry failed work with exponential backoff. The retry loop SHALL distinguish between failure-driven retries and continuation retries after a clean worker exit.

#### Scenario: Failed work is retried with backoff
- **WHEN** a claimed Symphony item exits abnormally
- **THEN** the item remains owned by the Symphony Project and is retried after an exponential backoff delay capped by project configuration

#### Scenario: Clean worker exit queues continuation retry
- **WHEN** a claimed Symphony item exits normally after completing its current worker session
- **THEN** the orchestrator queues a short continuation retry so active work can resume if the issue is still eligible

### Requirement: Repository Workflow Contract Loading
Before each worker attempt, the system SHALL load `WORKFLOW.md` from the target repository, parse front matter and prompt body, and fail the attempt if required workflow config or strict template rendering is invalid. Workflow changes SHALL apply to future attempts without requiring a control-plane restart.

#### Scenario: Missing workflow blocks execution
- **WHEN** the target repository does not provide a readable `WORKFLOW.md`
- **THEN** the worker attempt fails with a workflow validation error and the orchestrator records the failure for retry or operator action

#### Scenario: Updated workflow applies to a later attempt
- **WHEN** the repository workflow file changes before a future retry or continuation worker session
- **THEN** the later attempt uses the updated workflow contract instead of stale prompt state

### Requirement: Bounded Runtime Observability
The system SHALL expose bounded runtime state for running items, retry queues, recent failures, recent session events, and aggregate token/runtime totals for each Symphony Project.

#### Scenario: Operator inspects runtime state
- **WHEN** an operator requests Symphony project status
- **THEN** the response includes active items, retrying items, recent error context, recent session telemetry, and aggregate execution counters without mirroring the full remote board payload
