<!-- ITO:START -->
# Change: Restore web-ui and control-plane-api spec requirements

## Why
The previous retrospective spec update for workspace session management replaced the entire `web-ui` and `control-plane-api` spec files, unintentionally dropping existing requirements.

This change restores the full set of requirements in both specs by rewriting the spec documents to include both the pre-existing requirements and the newly added session-management requirements.

## What Changes
- Rewrite the `web-ui` spec file to include:
  - Cluster dashboard navigation and config visibility requirements.
  - Workspace session triage, termination, and settings/auth requirements.
- Rewrite the `control-plane-api` spec file to include:
  - Cluster overview and pod log endpoints.
  - Workspace session `createdAt` field and delete endpoint.

## Impact
- Affected specs: `web-ui`, `control-plane-api`.
- No code changes.
<!-- ITO:END -->
