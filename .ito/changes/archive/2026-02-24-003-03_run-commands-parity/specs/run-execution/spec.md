<!-- ITO:START -->
## ADDED Requirements

### Requirement: Start run supports task execution inputs

The control-plane API SHALL accept optional execution inputs when creating a run:

- `args`: an ordered list of strings to pass as container args to the harness image.

If `args` is omitted or empty, the run SHALL default to interactive mode (the harness pod remains alive for attach).

If `args` is provided, the harness pod SHALL execute the provided args after repo checkout and then exit with the command's status.

#### Scenario: Create interactive run

- **WHEN** a client creates a run without `args`
- **THEN** the created run's harness pod remains alive for attach/exec

#### Scenario: Create non-interactive run

- **WHEN** a client creates a run with `args: ["bash", "-lc", "go test ./..."]`
- **THEN** the harness pod checks out the repo and executes the task to completion

### Requirement: Web UI exposes Task + Advanced execution

The web UI SHALL provide a Task input when creating a run.

- When Task is non-empty, the UI SHALL translate it to `args: ["bash", "-lc", <task>]`.
- When Task is empty, the UI SHALL omit `args` to create an interactive run.
- The UI SHALL provide an Advanced section to edit the args array directly.
- The UI SHALL NOT expose or set the Kubernetes container `command` field for runs.

#### Scenario: Task executes via args

- **WHEN** a user enters a non-empty Task and starts a run
- **THEN** the UI sends `args: ["bash", "-lc", <task>]` to the API

#### Scenario: Advanced args override task mapping

- **WHEN** a user sets Advanced args explicitly
- **THEN** the UI sends the Advanced args as `args` (without setting container `command`)

### Requirement: UI warns against embedding secrets in Task

The UI SHALL display a warning that Task/args are stored in Kubernetes resources and may be visible to cluster readers.

#### Scenario: User sees secret-handling warning

- **WHEN** the Start Run form is displayed
- **THEN** the UI shows a warning to avoid placing secrets in the Task string
<!-- ITO:END -->
