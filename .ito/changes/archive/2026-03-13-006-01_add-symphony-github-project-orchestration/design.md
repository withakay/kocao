## Context

- Kocao already provides a control-plane API, a Kubernetes operator, durable Sessions, and ephemeral HarnessRuns for isolated execution.
- The requested Symphony feature keeps the long-running orchestration policy from the Symphony spec, but replaces Linear with GitHub Projects v2 plus GitHub Issues.
- One Kocao Symphony project must be able to watch a single GitHub project board while resolving items to issues across multiple configured repositories.

## Goals / Non-Goals

- Goals:
  - Introduce a first-class Kocao project resource for Symphony orchestration.
  - Poll GitHub Projects v2, normalize issue-backed items, and claim eligible work safely.
  - Reuse existing Session and HarnessRun primitives to execute claimed work in isolated pods.
  - Load workflow policy from each target repository's `WORKFLOW.md`.
  - Expose bounded operator-visible status through the API and web UI.
  - Keep GitHub PATs and execution credentials separated and auditable.
- Non-Goals:
  - GitHub App auth in MVP.
  - GitHub webhook delivery in MVP.
  - First-class PR/comment/label write APIs in the control plane.
  - Rich multi-tenant scheduling across namespaces.

## Decisions

- Decision: Introduce a namespaced `SymphonyProject` resource instead of extending `Session`.
  - Why: `Session` is already the durable workspace anchor for one repository context, while Symphony configuration must represent a multi-repository board, source credentials, poll settings, and orchestration status.
  - Alternatives considered: Reusing `Session` would blur workspace lifecycle and orchestration lifecycle; a standalone out-of-cluster service would duplicate Kocao's existing operator responsibilities.

- Decision: Use GitHub Projects v2 as the orchestration source of truth.
  - Why: It matches the user's request for a board, supports issue-backed items, and lets one board track issues from multiple repositories.
  - Alternatives considered: Polling issues directly from repositories would lose board-specific status mapping and make cross-repository triage much harder.

- Decision: Keep orchestration in the operator and use `Session` plus `HarnessRun` as execution primitives.
  - Why: The operator is already K8s-native, long-running, and responsible for turning desired state into pods. Reusing `Session` and `HarnessRun` keeps execution consistent with the rest of Kocao.
  - Alternatives considered: A dedicated scheduler Deployment per Symphony project would add another control plane and more moving parts for MVP.

- Decision: Separate board polling credentials from worker pod credentials.
  - Why: The GitHub token used to read and optionally mutate the board should stay in the operator path only. Harness pods should receive only the per-repository `gitAuth` and `agentAuth` refs explicitly configured for the resolved repository.
  - Alternatives considered: Mounting the board PAT into worker pods would expand blast radius and make auditing harder.

- Decision: Start with polling, fixed defaults, and bounded status.
  - Why: MVP should use a fixed poll interval (default 60s with jitter), default `maxConcurrentItems = 1` per Symphony project, and compact status fields instead of mirroring the entire GitHub board.
  - Alternatives considered: Webhooks, higher concurrency defaults, and unbounded status are follow-on optimizations once the lifecycle contract is stable.

- Decision: Require issue-backed items in configured repositories.
  - Why: Kocao needs a concrete repository and issue identity before it can create a workspace, resolve `WORKFLOW.md`, and launch a HarnessRun.
  - Alternatives considered: Supporting note items, PR-backed items, or repositories outside the allowlist would complicate execution and secret routing beyond MVP.

## Risks / Trade-offs

- Operator secret access expands the trust boundary and requires explicit RBAC, audit events, and careful status redaction.
- GitHub PATs are acceptable for MVP, but rate limits and long-term secret management point toward GitHub App auth as a likely follow-on.
- Restricted egress may block package registries or LLM providers for some workflows, so the execution policy needs a documented escape hatch without making `full` the default.
- Claim idempotency, retry state, and restart recovery need clear labels and ownership rules so controller restarts do not create duplicate work.
- Session retention after success/failure needs a concrete policy so Symphony does not leak durable workspaces indefinitely.

## Migration Plan

1. Add the `SymphonyProject` API model, operator reconciliation, and GitHub source client behind a new capability set.
2. Add control-plane API read/write endpoints and wire status into the web UI.
3. Reuse existing Session/HarnessRun creation paths for Symphony-created execution and label all child resources with Symphony ownership metadata.
4. Add hardening for secrets, audit, skip reasons, and retry handling before enabling broader concurrency.

## Open Questions

- Should GitHub write-back in MVP be limited to project status changes, or should worker flows also be able to add comments and labels through a Kocao-managed tool surface?
- How long should Symphony-created Sessions be retained after terminal completion, and should that retention be configurable per project?
- Do we want UI-based secret reference management in MVP, or should the UI only reference existing Secret names created out-of-band?
