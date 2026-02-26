<!-- ITO:START -->
## Why

The attach UI currently uses a single terminal implementation, which limits experimentation and user choice as terminal rendering needs diverge.
Adding an interchangeable engine toggle now enables faster iteration on terminal UX while preserving a stable fallback path.

## What Changes

- Add dual terminal engine support in the attach UI using `xterm.js` and `ghostty-web`.
- Add a per-session terminal engine toggle in the UI so users can switch engines immediately while attached.
- Mark `ghostty-web` as experimental in the toggle UI; keep `xterm.js` as the baseline option.
- Persist the selected engine in a cookie so the same workspace session restores the prior choice on reload.
- Keep attach transport semantics unchanged so engine choice only affects rendering/input behavior in the browser.

## Capabilities

### New Capabilities

- `terminal-engine-selection`: define how the UI exposes, persists, and applies interchangeable terminal engines for attach sessions

### Modified Capabilities

<!-- none -->

## Impact

- Affected UI: `web/src/ui/pages/AttachPage.tsx` (toggle, renderer host, per-session persistence)
- Affected UI internals: new terminal adapter layer under `web/src/ui` for engine interchangeability
- Affected dependencies: add `ghostty-web` and `xterm.js` integration/runtime assets to web package
- Affected tests: `web/src/ui/workflow.test.tsx` (or attach-focused tests) for toggle behavior, persistence, and attach continuity
<!-- ITO:END -->
