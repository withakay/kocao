## Context

The current naming encourages an incorrect runtime model where a run is interpreted as "start OpenCode interactively."
In practice, the platform starts a harness pod and executes a configured command against a checked-out repository.
Attach provides optional human interaction with that same pod and should not redefine what a run is.

## Goals / Non-Goals

- Goals:
  - Make run semantics explicit: non-interactive command execution by default.
  - Separate durable workspace identity from per-attempt execution identity.
  - Align terminology across API, schema, UI, and documentation.
  - Remove ambiguous lifecycle wording by requiring object-qualified labels.
- Non-Goals:
  - Redesign attach transport/protocol mechanics.
  - Introduce compatibility aliases for old naming.
  - Change scheduler/resource policy behavior beyond terminology/contract alignment.

## Decisions

- Decision: Define `Harness Run` as one harness pod execution attempt that runs configured command/args to completion.
  - Rationale: Matches operator behavior and expected CI/test command workflows.
  - Alternative considered: Keep run concept partially interactive by default. Rejected due to ambiguity and mismatch with actual execution model.

- Decision: Keep attach as a separate optional capability on an existing running Harness Run.
  - Rationale: Preserves interactive workflows while keeping execution semantics deterministic.
  - Alternative considered: Model attach as a separate interactive run mode. Rejected to avoid two conflicting meanings of run.

- Decision: Establish glossary split as `Workspace Session` (repo anchor + workspace PVC + policy) and `Harness Run` (single execution attempt).
  - Rationale: Disambiguates long-lived workspace context from ephemeral execution units.

- Decision: Use a strict language split by surface while preserving canonical nouns across both.
  - Contract-facing surfaces MUST use exact canonical contract vocabulary (`Workspace Session`, `Harness Run`, `Workspace Session Lifecycle`, `Harness Run Lifecycle`).
  - Product-facing UX copy SHOULD keep those nouns and may add explanatory helper text without inventing alternate object names.
  - Rationale: Preserves precision for integrations while keeping UI language understandable.

- Decision: Require qualified lifecycle language only (`Workspace Session Lifecycle`, `Harness Run Lifecycle`) and forbid bare `lifecycle`.
  - Rationale: Prevents overloaded state labels in UI and API docs.

- Decision: Apply hard-cutover naming updates with no compatibility aliases.
  - Rationale: Avoids prolonged dual terminology and contract drift.

## Risks / Trade-offs

- Breaking contract risk for existing clients that still use legacy term names.
  - Mitigation: Provide release notes, changelog entries, and explicit migration examples before rollout.

- Incomplete rename risk across API/UI/docs causing mixed vocabulary.
  - Mitigation: Add a proposal task to inventory and update all user-visible/session-run contract surfaces.

- Ambiguity risk if lifecycle labels are reintroduced without qualifier in future work.
  - Mitigation: Add naming checks in docs/UI review checklist and contract tests for label strings.

## Migration Plan

1. Update canonical glossary and contract definitions in spec deltas.
2. Rename API/schema/UI/docs surfaces from legacy session/run wording to Workspace Session/Harness Run wording.
3. Enforce qualified lifecycle labels on all exposed state surfaces.
4. Publish breaking-change migration notes with old-to-new term mapping.

### Legacy to Canonical Term Matrix

| Legacy term | Canonical replacement | Surface rule |
| --- | --- | --- |
| Session | Workspace Session | Required in API/schema/events/UI labels |
| Run | Harness Run | Required in API/schema/events/UI labels |
| Lifecycle | Workspace Session Lifecycle or Harness Run Lifecycle | Never unqualified |

## Open Questions

- None.
