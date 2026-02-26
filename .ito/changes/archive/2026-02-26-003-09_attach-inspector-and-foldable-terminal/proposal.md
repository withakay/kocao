<!-- ITO:START -->
## Why

The attach experience is close to IDE quality but still misses two high-value workflows: a dedicated inspector surface and a foldable terminal activity area.
Adding these closes the gap for debugging and quick state inspection without forcing users to leave the terminal context.

## What Changes

- Add a right-side Attach Inspector panel for session/run/connection metadata
- Add a foldable terminal activity panel (expand/collapse below terminal)
- Add command palette actions for inspector and activity panel toggles
- Add attach-specific keyboard shortcuts for inspector and activity panel
- Keep state synchronized across AttachPage, Shell shortcuts, and CommandPalette

## Capabilities

### Modified Capabilities

- `ide-terminal-experience`
- `workflow-ui-github`

## Impact

- Affected UI: `web/src/ui/pages/AttachPage.tsx`, `web/src/ui/components/CommandPalette.tsx`, `web/src/ui/components/Shell.tsx`
- Affected state: `web/src/ui/lib/useLayoutState.ts`
- New UI primitive: `web/src/ui/components/InspectorPanel.tsx`
- New tests: `web/src/ui/workflow.test.tsx`
<!-- ITO:END -->
