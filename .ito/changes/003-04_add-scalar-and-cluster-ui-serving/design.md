## Context

The control-plane API already exposes a live OpenAPI document at `/openapi.json`, while the workflow UI is built as a separate web artifact. Today there is no single in-cluster edge that serves both UI and API documentation from one endpoint, and there is no documented path for optional private-network exposure via Tailscale.

This change defines a cluster-native serving model with Caddy and Scalar, scoped for dev-kind first and with Tailscale integration planned but not fully rolled out.

## Goals / Non-Goals

**Goals:**
- Serve workflow UI and Scalar from the same control-plane pod.
- Keep OpenAPI docs live by sourcing Scalar from `/openapi.json` at runtime.
- Preserve existing API/auth behavior by reverse proxying API/websocket routes through Caddy.
- Define a dev-kind-first deployment and verification plan.
- Define optional Tailscale integration architecture and manifests as plan-ready artifacts.

**Non-Goals:**
- Full production Tailscale rollout in this change.
- Replacing control-plane API authn/authz with a new auth system.
- Changing harness execution behavior or CRD contracts unrelated to serving topology.

## Decisions

- Decision: Use Caddy as same-pod web edge in front of the control-plane API.
  - Rationale: Caddy provides simple static file serving plus HTTP/WebSocket reverse proxy in one lightweight runtime, minimizing custom Go edge code.
  - Alternatives considered:
    - Serve UI/Scalar directly from the Go API binary: fewer containers, but mixes edge/static concerns into API process and increases release coupling.
    - Separate web deployment: cleaner isolation, but adds inter-service routing complexity and diverges from "same pod" requirement.

- Decision: Serve Scalar from Caddy and load spec from live `/openapi.json`.
  - Rationale: avoids stale bundled spec files and keeps docs aligned with deployed API.
  - Alternatives considered:
    - Static bundled spec: simpler at runtime but drifts unless tightly regenerated on every API change.
    - Dual live+static fallback: more resilient but additional complexity not needed for initial rollout.

- Decision: Route model is unified at Caddy with path-based forwarding.
  - Rationale: one external endpoint for UI/docs/API; internal API remains private to pod network.
  - Routing targets:
    - `/` and UI assets -> static web build
    - `/scalar` -> Scalar app
    - `/api/v1/*`, `/openapi.json`, `/healthz`, `/readyz` -> control-plane API container
    - attach websocket path -> proxied upgrade to control-plane API

- Decision: Tailscale is plan-only in this change.
  - Rationale: request prioritizes architecture and cluster serving plan now; rollout can follow after validation.
  - Plan scope:
    - define optional overlay and sidecar shape on the control-plane deployment
    - define auth and network posture expectations
    - document operator steps and production readiness checklist

## Risks / Trade-offs

- [Proxy misconfiguration breaks attach websocket upgrades] -> Add explicit websocket route tests and a dev-kind smoke test for attach.
- [Caddy and API health semantics diverge] -> Keep readiness probing explicit for both edge and API paths.
- [Live OpenAPI endpoint unintentionally exposed beyond intended boundary] -> Preserve existing auth allowlist rules and document exposure model per environment.
- [Tailscale plan drifts before rollout] -> Record concrete acceptance criteria and follow-up implementation tasks in this change.

## Migration Plan

1. Add Caddy config and container wiring in deployment manifests.
2. Add Scalar static app wiring that fetches live `/openapi.json`.
3. Update dev-kind overlay to expose Caddy as the user-facing endpoint.
4. Add docs/runbook for routing, health checks, and local verification.
5. Add plan-only Tailscale overlay design and operational checklist (disabled by default).
6. Validate via automated tests plus dev-kind smoke verification; keep rollback as deployment manifest revert to API-only exposure.

## Open Questions

- None.
