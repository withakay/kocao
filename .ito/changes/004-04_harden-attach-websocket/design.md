## Context

Attach is implemented as a websocket endpoint that multiplexes clients and streams to/from `pods/exec`.
We need to harden the upgrade request and runtime behavior.

## Goals / Non-Goals

- Goals: origin checks, message limits, timeouts; safer browser token transport; audit attach usage.
- Non-Goals: multi-tenant isolation or a full auth provider.

## Decisions

- Add `Origin` validation with an explicit allowlist config. Default strict in prod.
- Add `SetReadLimit`, read deadlines, and ping/pong handling on websocket connections.
- Prefer an HttpOnly cookie-based attach token for browser flows (cookies are sent during websocket handshake).

## Risks / Trade-offs

- Cookie-based token transport requires careful origin checking to avoid CSWSH.
- Dev UX: must allow localhost origins when developing UI separately.

## Migration Plan

- Keep existing attach-token JSON response for non-browser clients.
- Add a browser-specific flow (cookie issuance) and update the UI to use it.

## Open Questions

- Should attach tokens be single-use to reduce replay risk?
