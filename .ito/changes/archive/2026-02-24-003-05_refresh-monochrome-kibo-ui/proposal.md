## Why
The current web console works functionally but feels visually inconsistent and too rough for fast expert workflows. We need a full-product refresh to deliver a clean monochrome dark interface with stronger identity and scanability, without changing backend or API behavior.

## What Changes
- Refresh the entire web console UX (sessions, runs, run details, and attach) into a unified monochrome-dark system with one subtle accent.
- Adopt Kibo UI components wherever an equivalent primitive exists, with custom components only where Kibo has no direct fit.
- Rework layout, hierarchy, spacing, and component states for high-density operator workflows and quick lifecycle scanning.
- Tighten labels and helper copy to a sharp technical tone aimed at expert users.
- Keep all existing API contracts and workflow semantics unchanged (UI/UX-only scope).

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `workflow-ui-github`: extend workflow-console requirements to mandate monochrome dark styling, Kibo-based composition, and full-route visual consistency.
- `attach-session`: extend browser attach UX requirements so attach has visual and interaction parity with the refreshed workflow console.

## Impact
- Affects `web/` route views, shared UI components, styling tokens, and frontend tests.
- Introduces Kibo-compatible frontend setup and dependencies needed by this Vite React application.
- Requires test updates for refreshed UX while preserving existing run/session/attach behavior and backend integration.
