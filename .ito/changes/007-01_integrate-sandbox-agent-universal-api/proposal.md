<!-- ITO:START -->
## Why
Kocao currently owns too much provider-specific harness behavior itself. Integrating `rivet-dev/sandbox-agent` gives us one in-sandbox API for OpenCode, Claude, Codex, and Pi, which reduces bespoke adapter code while aligning the product around a provider-neutral runtime that works in Kubernetes and can be driven consistently from the UI and the API.

## What Changes
- Add a new sandbox-agent-backed runtime contract for generic Kocao harness execution.
- Run `sandbox-agent` inside harness pods and make supported agents (`opencode`, `claude`, `codex`, `pi`) available through that single server API.
- Add control-plane API support for creating agent sessions, sending prompts/messages, replaying/streaming events, and managing agent lifecycle through Kocao-authenticated endpoints.
- Persist normalized sandbox-agent session metadata/events outside the sandbox so reconnect, audit, and resume flows survive client disconnects and pod churn.
- Add UI flows to choose an agent, start a sandbox-backed run, interact with the running agent, and stop/resume/reconnect to it.
- Tighten the security contract so sandbox-agent remains control-plane-mediated rather than directly exposed to end users.
- Explicitly keep Symphony-specific worker migration out of the first slice unless needed to support the generic Kocao run path.

## Capabilities

### New Capabilities
- `sandbox-agent-runtime`: provider-neutral runtime contract for launching and interacting with supported coding agents through sandbox-agent.
- `agent-session-ui`: browser workflow for selecting an agent, interacting with a live session, and managing lifecycle state.

### Modified Capabilities
- `control-plane-api`: add Kocao API surfaces for sandbox-agent session creation, messaging, event replay/streaming, and lifecycle control.
- `harness-runtime`: package and supervise sandbox-agent in the harness image alongside supported agent binaries and readiness behavior.
- `session-durability`: extend workspace/run durability to cover sandbox-agent session metadata and normalized event history.
- `security-posture`: require sandbox-agent access to stay behind Kocao authn/authz, audit, and credential-boundary rules.

## Impact
- Affected code: `build/Dockerfile.harness`, `build/harness/*`, `internal/operator/api/v1alpha1/*`, `internal/operator/controllers/*`, `internal/controlplaneapi/*`, `web/src/ui/*`, and associated tests/docs.
- APIs: new or expanded `/api/v1/*` endpoints for sandbox-backed agent session management and event streaming.
- Dependencies: introduces `sandbox-agent` as a pinned runtime dependency and likely generated/handwritten clients against its HTTP/OpenAPI contract.
- Operations: harness pods must start a sandbox-agent server, expose readiness/lifecycle state to Kocao, and preserve transcript continuity across reconnect/resume flows.
<!-- ITO:END -->
