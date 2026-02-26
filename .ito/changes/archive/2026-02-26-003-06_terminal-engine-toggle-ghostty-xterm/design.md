<!-- ITO:START -->
## Context

The attach page currently renders terminal output in a single built-in UI path. Product direction now requires two interchangeable engines (`xterm.js` and `ghostty-web`) so users can switch rendering technology without changing backend attach behavior.

This change is UI-centric and sits in module 003 (Product Experience). It must preserve existing websocket attach semantics, keep xterm as the stable baseline, and introduce `ghostty-web` as an experimental option.

## Goals / Non-Goals

**Goals:**

- Add interchangeable terminal engines in attach UI: `xterm.js` and `ghostty-web`.
- Provide a per-session toggle that applies immediately in an active attach session.
- Persist per-session engine choice in a cookie and restore it on reload.
- Keep attach transport/auth/protocol behavior unchanged.

**Non-Goals:**

- Redesign attach protocol, auth, or backend runtime behavior.
- Add global user preferences for terminal engine in this iteration.
- Guarantee pixel-identical behavior between engines for every escape-sequence edge case.

## Decisions

- **Use a terminal engine adapter boundary in UI.**
  - Decision: define a narrow adapter interface (mount, write, resize, dispose, onInput) and implement one adapter per engine.
  - Rationale: keeps attach transport logic independent from renderer choice and makes engine toggling low-risk.
  - Alternative considered: branch directly in `AttachPage.tsx`; rejected because it couples transport/UI state to renderer internals.

- **Keep one attach websocket transport regardless of engine.**
  - Decision: engine switches reuse the same active websocket session and stream pipeline.
  - Rationale: satisfies immediate switching requirement and avoids reconnect side effects.
  - Alternative considered: reconnect on switch; rejected because it breaks session continuity.

- **Per-session cookie persistence.**
  - Decision: store selected engine in a cookie keyed by workspace session ID.
  - Rationale: matches requested per-session behavior and survives reload.
  - Alternative considered: global localStorage preference; rejected due user preference for per-session cookie persistence.

- **Mark ghostty as experimental in UI copy.**
  - Decision: expose ghostty in selector with explicit experimental label; keep xterm default.
  - Rationale: enables adoption while setting stability expectations.

## Risks / Trade-offs

- [Ghostty browser/runtime incompatibilities] -> Mitigation: keep xterm default, preserve fast fallback via toggle.
- [Hot-switch lifecycle bugs (double listeners, leaked nodes)] -> Mitigation: strict adapter dispose lifecycle and focused tests.
- [Behavior divergence between engines] -> Mitigation: define minimum parity contract for input/output/resize and test against it.
- [Cookie key growth across many sessions] -> Mitigation: deterministic key format and overwrite behavior per session ID.

## Migration Plan

- Add new web dependencies and adapter layer behind attach UI toggle.
- Deploy as backward-compatible UI-only change; backend unaffected.
- Rollback by reverting web build or forcing selector to xterm-only path.

## Open Questions

- Should the UI surface an explicit warning/tooltip when ghostty fails initialization and auto-falls back to xterm?
- Should we normalize copy/paste and keyboard shortcut behavior across engines in this change or follow-up?
<!-- ITO:END -->
