<!-- ITO:START -->
## Why

The web UI can create runs but cannot specify what should execute, so “Start Run” typically creates an idle harness pod that just waits for attach.
The control-plane API already supports passing execution parameters (args/workingDir/env), so the UI and API are out of parity and user expectations are mismatched.

## What Changes

- Add a first-class Task field to “Start Run” in the web UI.
- Map Task to harness execution via container args (default `bash -lc <task>`), preserving the harness entrypoint (checkout/safety) semantics.
- Keep Task optional: empty Task continues to create an interactive/attachable run (idle harness pod).
- Add an Advanced section to start runs with explicit args and optional execution fields, aligned with the API request.

## Capabilities

### New Capabilities

- `run-execution`: define and expose run execution inputs (task/args) across API + UI with safe defaults

### Modified Capabilities

<!-- none -->

## Impact

- Affected UI: `web/src/ui/pages/SessionDetailPage.tsx`, `web/src/ui/lib/api.ts` (start run payload)
- Affected API: `internal/controlplaneapi/api.go` (request/response contract tests/docs as needed)
- Affected operator/harness: none expected (already supports args execution via entrypoint)
<!-- ITO:END -->
