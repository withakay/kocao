## Why
The current language makes "run" sound like "start OpenCode and hand it a task," which does not match the actual runtime contract. We need terminology that matches the architecture so API users, UI users, and operators share the same mental model.

## What Changes
- Define a Harness Run as one harness pod execution attempt that runs configured command/args against the checked-out repo and exits when complete.
- Define interactivity as optional attach behavior on the same harness pod, not the default run behavior.
- Rename session/run terminology across API, schema, docs, and UI to distinguish Workspace Session (durable anchor) from Harness Run (execution attempt).
- Require qualified lifecycle labels (Harness Run Lifecycle, Workspace Session Lifecycle) and disallow bare "lifecycle".
- **BREAKING**: perform hard-cutover naming updates with no compatibility aliases for legacy terms.

### Terminology by Surface
- **Contract-facing surfaces (API/schema/CRD/events/tests/docs specs)** MUST use canonical terms: `Workspace Session`, `Harness Run`, `Workspace Session Lifecycle`, and `Harness Run Lifecycle`.
- **Product-facing UX copy (page titles/table columns/badges)** SHOULD use the same canonical nouns for consistency; explanatory helper text is allowed (for example, "durable workspace context" for Workspace Session and "single execution attempt" for Harness Run).
- Bare `lifecycle` is not allowed on any surface.

### Migration Matrix (Hard Cutover)
| Legacy term | Canonical replacement | Notes |
| --- | --- | --- |
| Session | Workspace Session | Durable repo + workspace PVC + policy anchor |
| Run | Harness Run | One pod execution attempt for configured command/args |
| Lifecycle | Workspace Session Lifecycle or Harness Run Lifecycle | Always object-qualified |

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `harness-runtime`: clarify command-execution semantics for runs and attach behavior boundaries.
- `session-durability`: clarify workspace session identity and lifecycle terminology across surfaces.

## Impact
- Affects API contracts, CRD/status naming, operator/controller state surfaces, and UI glossary.
- Requires coordinated updates to docs/tests and release notes for breaking terminology changes.
- Requires copy-level QA so UX labels remain readable while preserving canonical object names.
- Reduces ambiguity between durable workspace context and per-attempt command execution.
