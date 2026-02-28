<!-- ITO:START -->
# Change: Improve workspace session management UX

## Why
The Workspace Sessions UI currently makes session triage difficult:

- No session start timestamp is shown, and the list is not ordered by recency.
- There is no first-class way to terminate (kill) an Active session from the UI.
- The token entry box sits in the top bar, competing with primary navigation and consuming space.
- The refresh button placement is confusing.

These issues slow down day-to-day operations when many sessions exist.

## What Changes
- Control-plane API:
  - Add `createdAt` (RFC3339) to workspace session responses, derived from Kubernetes metadata.
  - Add `DELETE /api/v1/workspace-sessions/{workspaceSessionID}` to terminate a workspace session.
- Web UI:
  - Add a Settings page to manage the bearer token (save/remember/clear) and show required scopes.
  - Move token entry out of the Topbar.
  - Update sidebar navigation with section headers and icons, and add Settings navigation.
  - Improve the Workspace Sessions page:
    - Show a Started column (createdAt).
    - Sort sessions newest-first.
    - Add a Kill action for sessions in phase `Active`.
    - Move the Refresh button to the table header (next to polling status).
    - Improve the Provision Session form layout.

## Impact
- Affected specs: `control-plane-api`, `web-ui`.
- Affected code:
  - `internal/controlplaneapi/api.go`, `internal/controlplaneapi/openapi.go`
  - `web/src/ui/pages/SessionsPage.tsx`, `web/src/ui/pages/SettingsPage.tsx`, navigation components
- Backward compatibility:
  - `createdAt` is additive and optional for older clients.
  - Workspace session delete is a new endpoint gated by existing auth scope checks.
<!-- ITO:END -->
