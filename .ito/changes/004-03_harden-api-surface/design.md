## Context

The API uses custom routing and per-handler scope checks. Several low-effort defaults (size/time limits) are missing.
We want to harden without redesigning the API.

## Goals / Non-Goals

- Goals: add body size limits; tighten timeouts; make “forgot to add authz” less likely.
- Non-Goals: introduce a full framework, gateway, or external auth provider.

## Decisions

- Add a request size limit in the JSON decoder path.
- Add additional server timeouts (read/write/idle) in `http.Server`.
- Keep existing scope-based authz, but improve patterns to fail closed where practical.

## Risks / Trade-offs

- Some clients/tests may need to adjust for stricter parsing and limits.

## Open Questions

- Should we add a simple rate limiter (per token / per IP) at the API layer, or rely on ingress/service mesh?
