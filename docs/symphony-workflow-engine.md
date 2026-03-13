# Symphony Workflow Engine

This document covers the workflow-driven execution layer introduced by `006-02_add-symphony-workflow-engine`.

## What It Adds

- Repository `WORKFLOW.md` discovery and parsing
- Strict prompt rendering from normalized issue data
- Codex app-server execution in an isolated per-issue workspace
- Continuation retries and richer worker telemetry in Symphony status

## Repository Configuration

Each Symphony repository entry may include local development fields used by the controller-side workflow engine:

```yaml
repositories:
  - owner: withakay
    name: kocao
    repoURL: https://github.com/withakay/kocao
    localPath: /absolute/path/to/local/checkout
    workflowPath: .kocao/WORKFLOW.md # optional, defaults to WORKFLOW.md at repo root
```

`localPath` is intended for local/dev or tightly controlled in-cluster execution contexts where the controller can read repository contents directly.

## Security Posture

- Workflow hooks in `WORKFLOW.md` are currently rejected for Symphony execution.
- If the workflow omits Codex posture fields, Kocao applies these defaults:
  - `approval_policy: untrusted`
  - `thread_sandbox: workspace-write`
  - `turn_sandbox_policy: workspace-write`
- Secret-looking values in worker messages are redacted before storage.
- Worker metadata redacts local workspace and workflow paths before exposing them through run annotations or project status.

## Verification

Relevant commands for the workflow engine path:

```bash
go test ./internal/symphony/workflow ./internal/symphony/runner ./internal/operator/controllers
make test
make lint
```
