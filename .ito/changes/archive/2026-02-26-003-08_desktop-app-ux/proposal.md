<!-- ITO:START -->
## Why

The UI works but still feels like a web page. It needs to feel like a desktop IDE application — Cursor-inspired dark aesthetic, resizable panels with drag handles, collapsible sections, inspector panels for detail views, and a polished command palette. These are the finishing touches that make the tool feel professional and native.

## What Changes

- Upgrade command palette to Cursor-style dark aesthetic with glassmorphism, larger type, keyboard hint badges, and category icons
- Add resizable panel system with drag handles (pointer-event-based) for terminal, sidebar, and inspector panels
- Add collapsible/foldable sections in detail pages (session detail, run detail) with smooth animation
- Add inspector panel pattern for run/session detail views (slide-in from right, keyboard dismissable)
- Tighten overall spacing, typography, and visual hierarchy for desktop-app density
- Wire fullscreen terminal toggle through shared context so command palette can trigger it

## Capabilities

### Modified Capabilities

- `workflow-ui-github`: visual polish, collapsible sections, inspector panels
- `ide-terminal-experience`: resizable terminal panel, fullscreen context wiring, command palette upgrade

## Impact

- Affected UI: `web/src/ui/components/CommandPalette.tsx`, `web/src/ui/components/Shell.tsx`, all page components
- New UI: resizable panel primitive, collapsible section primitive, inspector panel pattern
- Affected state: fullscreen context shared between command palette and attach page
<!-- ITO:END -->
