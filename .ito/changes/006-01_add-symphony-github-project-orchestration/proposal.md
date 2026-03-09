<!-- ITO:START -->
## Why

Kocao can already run isolated harness pods, but operators still need a repeatable way to pull queued GitHub work and drive it forward without manually creating runs issue-by-issue. Adding Symphony as a first-class Kocao capability turns a GitHub Projects v2 board into an orchestrated queue, lets one project board cover issues across multiple repositories, and keeps workflow policy in each target repo's `WORKFLOW.md`.

## What Changes

- Add a new Symphony Project model in Kocao for configuring a GitHub Projects v2 board, active and terminal states, supported repositories, secret refs, and runtime defaults.
- Add a GitHub project source adapter that polls issue-backed project items, normalizes GitHub issue data, and skips unsupported items or repositories with operator-visible reasons.
- Add Symphony orchestration logic for claim, retry, reconcile, deterministic per-issue execution, strict `WORKFLOW.md` loading, and bounded runtime status.
- Reuse Kocao Session and HarnessRun execution primitives so Symphony claims create isolated worker pods against the repository referenced by each issue.
- Extend the control-plane API and web UI with create, list, detail, pause, and refresh surfaces for Symphony projects and live runtime state.
- Extend security and audit rules so GitHub PATs remain secret-scoped, repository execution stays allowlisted, and Symphony claim/result transitions are auditable.

## Capabilities

### New Capabilities

- `symphony-projects`: project-level configuration and lifecycle for a GitHub-backed Symphony orchestration target.
- `github-project-source`: GitHub Projects v2 polling, issue normalization, repository allowlisting, and item eligibility rules.
- `symphony-orchestration`: poll, reconcile, retry execution, workflow loading, worker session lifecycle, and runtime observability.

### Modified Capabilities

- `control-plane-api`: add Symphony project CRUD, runtime state, and operator control endpoints.
- `operator-reconcile`: reconcile Symphony projects and create or track child Sessions and HarnessRuns.
- `security-posture`: extend secret handling, audit, and default egress rules for Symphony-managed work.
- `web-ui`: add Symphony project management and runtime status views in Kocao.

## Impact

- Affected code: `internal/controlplaneapi/`, `internal/operator/`, `internal/controlplanecli/`, `web/`, and new Symphony/GitHub integration packages.
- Affected infrastructure: Kubernetes CRDs/controllers, Secrets/RBAC, GitHub API integration, and HarnessRun label/ownership contracts.
- Dependencies: GitHub GraphQL/REST client support for Projects v2 and issue metadata, plus shared status/cache helpers required by the operator.
<!-- ITO:END -->
