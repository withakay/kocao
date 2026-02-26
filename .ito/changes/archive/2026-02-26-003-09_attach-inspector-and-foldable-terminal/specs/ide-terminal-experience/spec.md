<!-- ITO:START -->
## MODIFIED Requirements

### Requirement: Attach view provides inspector and activity surfaces

The attach view SHALL provide a right-side inspector panel and a foldable bottom activity panel while keeping terminal interaction primary.

Inspector and activity panel state SHALL be controllable from the page controls, command palette, and keyboard shortcuts.

#### Scenario: User toggles inspector and activity panel in attach

- **WHEN** a user toggles inspector or activity panel from any entrypoint (button, Cmd+K action, shortcut)
- **THEN** the attach UI updates immediately and all entrypoints reflect the same shared state

### Requirement: Attach command palette exposes context-aware layout actions

On attach routes, the command palette SHALL include actions to toggle fullscreen terminal, inspector panel, and activity panel.

#### Scenario: Cmd+K shows attach-specific actions

- **WHEN** a user opens command palette while on an attach route
- **THEN** the palette shows toggle actions for fullscreen, inspector, and activity panel
<!-- ITO:END -->
