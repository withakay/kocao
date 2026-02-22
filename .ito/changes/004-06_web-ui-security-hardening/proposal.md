<!-- ITO:START -->
## Why

The web UI currently stores the API token in `localStorage` and performs attach by embedding an attach token in the
websocket URL query string. Under defense-in-depth assumptions, we should reduce token exposure and make “safe by default”
UX decisions.

## What Changes

- Make token persistence opt-in (default to non-persistent/session storage).
- Ensure the UI avoids secrets in URLs and avoids accidental token logging.
- Improve UX around auth failures (clear token on 401/403, explicit logout/clear).

## Capabilities

### New Capabilities

- `web-ui`: UI security requirements around token handling and attach flow

### Modified Capabilities

<!-- none (no existing specs yet) -->

## Impact

- Affected code: `web/src/ui/auth.tsx`, `web/src/ui/pages/AttachPage.tsx`, `web/src/ui/lib/api.ts`, `web/src/ui/workflow.test.tsx`
- UX impact: token handling changes; attach flow changes (may depend on backend attach hardening)
<!-- ITO:END -->
