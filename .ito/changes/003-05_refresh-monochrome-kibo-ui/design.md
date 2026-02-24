## Context

The existing React console is functional but visually fragmented: route-level layouts feel uneven, styling is hand-rolled in one global stylesheet, and accent usage is louder than desired for an operator-focused tool. The requested direction is a full UX rethink toward a clean monochrome dark interface with a single subtle accent, plus broad adoption of Kibo UI components.

This change is intentionally frontend-only. API contracts, polling behavior, auth semantics, and attach transport behavior remain unchanged.

## Goals / Non-Goals

**Goals:**
- Refresh all user-facing console routes into a cohesive monochrome-dark visual system.
- Use Kibo UI components wherever they map to existing primitives (buttons, inputs, cards, tables, navigation, status surfaces).
- Improve scan speed for technical users through clearer hierarchy, tighter spacing, and stronger state affordances.
- Align copy and labels to a concise technical tone.
- Preserve current workflow behavior and backend integration.

**Non-Goals:**
- Adding or changing backend/API endpoints.
- Changing auth, token, websocket, or attach protocol semantics.
- Reworking business logic for workspace session or harness run orchestration.

## Decisions

- Decision: Introduce a shared monochrome token system with one accent and enforce it across all routes.
  - Rationale: A tokenized system gives consistent contrast, state styling, and maintainable future iteration.
  - Alternatives considered:
    - Keep route-local styling tweaks only: faster short term but continues inconsistency.
    - Use multi-accent palette: more expressive but conflicts with monochrome brief.

- Decision: Adopt Kibo UI components wherever an equivalent primitive exists; keep local wrappers for app-specific behavior.
  - Rationale: Kibo speeds delivery of modern, composable components while wrappers preserve application semantics and testability.
  - Alternatives considered:
    - Keep all custom components: maximal control but slower and less consistent.
    - Full direct Kibo usage without wrappers: less code initially but harder to keep app-specific behavior centralized.

- Decision: Refresh layout architecture first (shell/topbar/navigation/grid) before page-specific migrations.
  - Rationale: Shared layout primitives reduce repeated effort and visual drift.
  - Alternatives considered:
    - Page-by-page isolated redesign: simpler sequencing but higher risk of inconsistent UX.

- Decision: Keep attach page in the same design system while preserving strict viewer/driver interaction constraints.
  - Rationale: Attach is a core operator surface and must feel integrated, but security/role semantics cannot regress.

## Risks / Trade-offs

- [Kibo setup introduces tooling churn in a non-Tailwind codebase] -> Isolate setup in a dedicated foundation task and lock verification with lint/build/test gates.
- [Visual refresh unintentionally reduces readability in dense tables/forms] -> Define minimum contrast rules and validate scan-heavy pages first.
- [Copy changes introduce ambiguity for existing operators] -> Keep labels technical and behavior-specific; avoid changing action semantics.
- [Attach UX polish could accidentally alter role gating behavior] -> Add explicit tests for viewer read-only and driver send-control interactions.

## Migration Plan

1. Add Kibo-compatible frontend setup and baseline design tokens.
2. Build shared shell/topbar/navigation primitives with monochrome styling.
3. Migrate list/workflow pages (sessions and runs) to new component system.
4. Migrate detail and attach pages with parity in status, actions, and error surfaces.
5. Update tests for route rendering, key actions, and attach role behavior.
6. Verify with `pnpm -C web lint`, `pnpm -C web test`, and `pnpm -C web build`.

## Open Questions

- None.
