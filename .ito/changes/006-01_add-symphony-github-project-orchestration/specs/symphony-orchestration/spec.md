## ADDED Requirements

### Requirement: Poll, Claim, Reconcile, and Retry Loop
The Symphony orchestrator SHALL poll eligible items on a fixed cadence, enforce per-project concurrency, avoid duplicate claims, and retry failed work with exponential backoff.

#### Scenario: Failed work is retried with backoff
- **WHEN** a claimed Symphony item exits abnormally
- **THEN** the item remains owned by the Symphony Project and is retried after an exponential backoff delay capped by project configuration

### Requirement: Deterministic Worker Execution
For each claimed item, the system SHALL create or select an isolated execution context derived from the issue identifier and launch a HarnessRun against the target repository in that context.

#### Scenario: Retry reuses the same durable execution anchor
- **WHEN** the same issue is retried after a prior failed attempt
- **THEN** the Symphony Project reuses the same durable Session identity for that issue and creates a new HarnessRun attempt linked to it

### Requirement: Repository Workflow Contract Loading
Before each worker attempt, the system SHALL load `WORKFLOW.md` from the target repository, parse front matter and prompt body, and fail the attempt if required workflow config or strict template rendering is invalid.

#### Scenario: Missing workflow blocks execution
- **WHEN** the target repository does not provide a readable `WORKFLOW.md`
- **THEN** the worker attempt fails with a workflow validation error and the orchestrator records the failure for retry or operator action

### Requirement: Bounded Runtime Observability
The system SHALL expose bounded runtime state for running items, retry queues, recent failures, and aggregate token/runtime totals for each Symphony Project.

#### Scenario: Operator inspects runtime state
- **WHEN** an operator requests Symphony project status
- **THEN** the response includes active items, retrying items, recent error context, and aggregate execution counters without mirroring the full remote board payload
