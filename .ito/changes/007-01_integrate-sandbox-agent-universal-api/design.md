## Context
Kocao already orchestrates Workspace Sessions and Harness Runs in Kubernetes, but the in-harness agent control path is fragmented around individual CLIs and provider-specific assumptions. `sandbox-agent` offers a single HTTP/SSE control plane inside the sandbox for OpenCode, Claude, Codex, and Pi, which matches the direction of treating the harness as a generic execution environment while keeping Kocao responsible for tenancy, auth, policy, audit, and UX.

The external acceptance criteria for this change are clear: operators must be able to run Kocao in Kubernetes, launch a sandbox-agent-backed instance from the UI and API, choose OpenCode/Claude/Codex/Pi, interact with it, and manage lifecycle. The design therefore focuses on the generic Kocao run path rather than Symphony-specific worker migration.

## Goals / Non-Goals
- Goals:
  - Make `sandbox-agent` the only generic in-sandbox agent API Kocao talks to.
  - Preserve Kocao as the public API, authn/authz, and audit boundary.
  - Support OpenCode, Claude, Codex, and Pi from the same run/session flow.
  - Persist enough normalized event and session metadata to support reconnect and resume semantics.
  - Expose a first-class browser experience for launch, transcript viewing, follow-up prompts, and lifecycle control.
- Non-Goals:
  - Replacing Kubernetes orchestration with sandbox-agent.
  - Exposing sandbox-agent directly to browsers or external clients.
  - Migrating every Symphony worker path in the same change unless a generic abstraction extraction naturally enables it.
  - Solving long-term billing/accounting across upstream model providers in this slice.

## Decisions
- Decision: Kocao remains the external API façade; sandbox-agent runs only inside the harness pod.
  - Why: the sandbox-agent docs recommend a backend hop for auth, session affinity, and persistence; Kocao already owns those responsibilities.
- Decision: Use one sandbox-agent server per Harness Run and map Kocao Workspace Session/Harness Run identities to sandbox-agent session IDs.
  - Why: this fits current pod-per-run orchestration and keeps provider execution isolated.
- Decision: Kocao reaches sandbox-agent over an internal pod-local or pod-network endpoint exposed only inside the cluster, with readiness gated by sandbox-agent health before a session becomes interactable.
  - Why: this preserves the control-plane boundary and gives the operator a concrete readiness signal.
- Decision: Persist normalized sandbox-agent events/metadata in Kocao-managed durability surfaces rather than relying on sandbox-agent memory.
  - Why: sandbox-agent explicitly states sessions are in-memory only and must be persisted by the integrating platform.
- Decision: Keep provider selection additive to current run creation flows and publish a least-common-denominator capability contract for supported agents.
  - Why: it lets us land the universal API without forcing an immediate product-wide rename or destroying current advanced workflows, while making unsupported provider-specific features explicit.

## Proposed Architecture
1. **Harness runtime**
   - Bake/pin `sandbox-agent` into `build/Dockerfile.harness`.
   - Ensure supported agent binaries/auth mount points remain available for OpenCode, Claude, Codex, and Pi.
   - Update the harness entrypoint/supervision contract so the pod can start sandbox-agent reliably, surface health/readiness, and keep the server internal to the pod.
2. **Control-plane mediation**
   - Extend the Kocao API with run-scoped agent-session endpoints for create/start, prompt/message, event list/stream, inspect, stop, and resume/reconnect.
   - Maintain Kocao bearer-token auth, scope checks, and audit records on every control operation.
   - Translate Kocao request/response models to sandbox-agent calls and normalized event payloads.
3. **Durability/lifecycle**
   - Persist sandbox-agent session IDs, selected agent metadata, offsets, and normalized events outside the sandbox.
   - Tie that state to Workspace Session + Harness Run so reconnects and resumed runs can reconstruct a coherent transcript/history.
4. **UI**
   - Add agent selection to run launch.
   - Add a run interaction surface showing session state, transcript/events, prompt input, and lifecycle controls.
   - Allow reconnect/reload without losing the displayed transcript.

## Risks / Trade-offs
- `sandbox-agent` is still an external dependency and experimental in some compatibility areas; version pinning and contract tests will matter.
- Pi/OpenCode/Claude/Codex do not all expose identical features, so Kocao must define a least-common-denominator contract and surface unsupported capabilities explicitly.
- Transcript persistence adds backend state-management complexity, but not doing it would violate reconnect/resume expectations.
- There is a tension between universal API purity and existing raw-shell attach flows. This design keeps Kocao attach/security controls intact while introducing a separate sandbox-backed agent interaction path.

## Migration Plan
1. Add the new runtime and API contracts behind additive request/response fields and endpoints.
2. Prove the Kubernetes happy path with one running Harness Run hosting sandbox-agent.
3. Ship UI support for launch + interaction against the new API.
4. Once the new path is stable, follow up on removing or collapsing provider-specific internal abstractions that are made redundant.

## Verification Plan
- Backend: `go test ./internal/controlplaneapi/... ./internal/operator/... ./internal/harnessruntime/...`
- Web: `pnpm -C web test && pnpm -C web lint`
- Runtime: `make harness-smoke`
- Cluster/API happy path: local kind deploy + create Workspace Session/Harness Run + create sandbox-backed agent session + send prompt + receive events + stop/resume.

## Open Questions
- Which Kocao durability surface should own normalized event persistence first (audit store extension, API store, or new dedicated backing store)?
- Should run creation create the sandbox-agent session eagerly, or should Kocao separate pod start from agent-session start?
- Do we preserve raw attach as an orthogonal debugging feature, or fold all end-user interaction into the sandbox-agent-backed UI flow over time?
