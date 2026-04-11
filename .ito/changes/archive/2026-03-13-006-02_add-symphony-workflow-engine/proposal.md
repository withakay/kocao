<!-- ITO:START -->
## Why

`006-01` gives Kocao a GitHub-project-backed orchestration MVP, but it still stops short of the core Symphony promise: loading workflow policy from each repository and running a coding-agent session against the claimed issue. `006-02` closes that gap so Symphony can execute repo-defined work instead of only materializing Kocao Sessions and HarnessRuns.

## What Changes

- Add a repository-owned `WORKFLOW.md` contract loader with front matter parsing, strict template rendering, validation errors, and runtime reload behavior for future worker attempts.
- Add a Symphony agent runner that starts and manages Codex app-server sessions, handles continuation turns, enforces timeouts, and records worker-level session metadata.
- Extend Symphony orchestration to use workflow-derived prompts, per-issue workspace preparation, continuation retry behavior, and aggregate runtime/token accounting.
- Extend the control-plane API and operator-visible status model with richer worker/session detail needed to inspect workflow execution.
- Extend security posture documentation and enforcement for workflow-supplied hooks, app-server approval posture, and agent-facing secret boundaries.

## Capabilities

### New Capabilities

- `symphony-workflow-contract`: repository `WORKFLOW.md` discovery, parsing, config resolution, validation, and prompt rendering.
- `symphony-agent-runner`: Codex app-server lifecycle, turn streaming, continuation turns, and session telemetry extraction.

### Modified Capabilities

- `symphony-orchestration`: integrate workflow contracts and agent-runner execution into claim, retry, reconcile, and workspace lifecycle behavior.
- `control-plane-api`: expose richer Symphony runtime detail for worker sessions, retries, recent errors, and aggregate execution counters.
- `security-posture`: define and enforce the trust boundary for workflow hooks, app-server approvals, sandbox posture, and secret handling.
- `web-ui`: surface the richer Symphony runtime state needed for operators to inspect in-flight agent execution.

## Impact

- Affected code: new workflow and agent-runner packages under `internal/`, deeper changes in `internal/operator/controllers/`, `internal/controlplaneapi/`, and `web/src/ui/`.
- Affected systems: Kubernetes worker pod execution, repository workspace management, audit/event handling, and Codex app-server integration.
- Dependencies: repo checkout and file access for `WORKFLOW.md`, Codex app-server compatibility, and additional tests for long-running worker lifecycle behavior.
<!-- ITO:END -->
