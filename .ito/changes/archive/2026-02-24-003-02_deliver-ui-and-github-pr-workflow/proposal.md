## Why
The MVP value is realized through a user-facing workflow that starts runs, monitors progress, and captures pull request outcomes. Without UI and PR flow integration, orchestration remains an internal-only primitive.

## What Changes
- Build the initial web console for sessions and runs with lifecycle visibility.
- Add flow for creating sessions, launching runs, and opening run details.
- Integrate GitHub PR outcome reporting so users can see branch/PR results in platform UI.

## Capabilities

### New Capabilities
- `workflow-ui-github`: user workflow for session/run management and GitHub PR outcome visibility.

### Modified Capabilities
- None.

## Impact
- Affects frontend routes/components, control-plane client APIs, and GitHub status mapping.
- Defines the first end-to-end user experience from session creation to PR result.
