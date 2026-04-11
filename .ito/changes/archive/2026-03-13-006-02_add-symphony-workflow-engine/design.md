## Context

- `006-01` established a Kocao-native GitHub Projects orchestration MVP.
- The remaining Symphony scope is the workflow engine itself: repository `WORKFLOW.md` contracts, prompt rendering, coding-agent session management, and richer execution observability.
- This change should build on the existing `SymphonyProject` model rather than replacing the GitHub-backed orchestration surface.

## Goals / Non-Goals

- Goals:
  - Load workflow policy from `WORKFLOW.md` in the target repository for each claimed issue.
  - Render strict prompts from workflow config plus normalized issue data.
  - Start and manage Codex app-server sessions within the claimed issue workspace.
  - Support continuation turns, retries, timeouts, and session telemetry.
  - Expose richer execution detail through Kocao status surfaces.
- Non-Goals:
  - Replacing GitHub Projects as the tracker source.
  - Adding a separate external Symphony daemon outside Kocao.
  - Building the optional standalone HTTP dashboard from the original language-agnostic spec.

## Decisions

- Decision: Keep `SymphonyProject` as the orchestration root and layer the workflow engine behind it.
  - Why: `006-01` already provides the source-of-truth project object, repo allowlist, and Kocao execution linkage.
  - Alternatives considered: introducing a second orchestration CRD would split responsibility and make status harder to reason about.

- Decision: Add a dedicated workflow-contract package separate from GitHub source loading.
  - Why: repository workflow parsing is orthogonal to issue polling and should be testable without GitHub or Kubernetes concerns.
  - Alternatives considered: embedding `WORKFLOW.md` parsing inside the controller would make validation, reload behavior, and tests too coupled.

- Decision: Add a dedicated agent-runner package around Codex app-server.
  - Why: the app-server lifecycle, JSON line protocol, timeouts, approvals, and usage extraction are a subsystem with their own tests and failure modes.
  - Alternatives considered: shelling out from the controller directly would make retry and telemetry handling brittle.

- Decision: Keep worker execution inside the per-issue workspace/session model introduced in `006-01`.
  - Why: it preserves deterministic workspace reuse and lets retries continue from the same durable Kocao context.
  - Alternatives considered: creating a fresh session every turn would discard useful context and break the Symphony continuation model.

- Decision: Expose richer status through the existing API/UI instead of adding a separate Symphony-specific HTTP server.
  - Why: Kocao already has an authenticated control-plane API and Symphony pages; reusing them keeps operator UX consistent.
  - Alternatives considered: mirroring the original standalone HTTP server would duplicate state presentation and auth concerns.

## Risks / Trade-offs

- `WORKFLOW.md` files can contain trusted hook scripts, so the execution trust boundary must stay explicit and auditable.
- Codex app-server protocol drift or schema variation could make the runner fragile without good fixture coverage.
- Long-running turns and continuation retries increase complexity around cancellation, shutdown, and resource cleanup.
- Richer runtime telemetry can become noisy if status payloads are not kept bounded.

## Migration Plan

1. Implement workflow contract parsing and tests independent of Kubernetes execution.
2. Implement the agent-runner with protocol fixtures and timeout coverage.
3. Integrate both into Symphony orchestration and enrich runtime status.
4. Surface the new telemetry in API/UI and document the trust posture.

## Open Questions

- Should `WORKFLOW.md` reload behavior be controller-driven, worker-driven, or both for missed file-watch events?
- Do we need to persist any session metadata beyond CRD status, or is bounded in-memory-plus-status state sufficient for this phase?
- Which approval and sandbox defaults should Kocao declare for Codex-managed Symphony runs in MVP?
