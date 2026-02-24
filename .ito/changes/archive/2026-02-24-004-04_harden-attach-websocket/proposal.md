<!-- ITO:START -->
## Why

Attach is one of the highest-risk surfaces: it upgrades to websockets and bridges into `pods/exec`.
Current defaults are permissive (accept any Origin; no websocket read limits/timeouts) and the browser
workflow passes attach tokens via the URL query string.

## What Changes

- Add websocket origin validation and make it configurable for dev vs prod.
- Add websocket message size limits, deadlines, and ping/pong handling to reduce abuse.
- Improve token transport so browser attach does not require putting secrets in URLs.
- Expand auditing around attach lifecycle and control/stdin events.

## Capabilities

### New Capabilities

- `attach-session`: hardened attach websocket behavior and token transport

### Modified Capabilities

<!-- none (no existing specs yet) -->

## Impact

- Affected code: `internal/controlplaneapi/attach.go`, `internal/controlplaneapi/api.go`, `web/src/ui/pages/AttachPage.tsx`
- Operational impact: new config for allowed websocket origins; stricter attach behavior
<!-- ITO:END -->
