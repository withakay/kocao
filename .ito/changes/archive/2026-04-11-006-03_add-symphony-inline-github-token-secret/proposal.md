<!-- ITO:START -->
## Why

Creating a Symphony project currently requires operators to manually create a Kubernetes Secret first and then paste its name into the UI, which is easy to get wrong and encourages people to paste raw PATs into the wrong field. Kocao should handle GitHub PAT capture in the UI and create the backing Secret automatically so project setup is safer and smoother.

## What Changes

- Add a write-only GitHub PAT input to the Symphony project UI and stop asking normal users to provide a raw Secret name during create/edit flows.
- Extend the Symphony project API so create/update requests can include an inline GitHub PAT, derive a deterministic Secret name from the project name and GitHub owner, and create or update the backing Secret automatically.
- Preserve `spec.source.tokenSecretRef` as the internal stored reference, but make it operator-managed for the common UI path.
- Reject PAT-shaped values where a Secret name is expected and avoid echoing raw token values in API errors, audit metadata, or UI surfaces.

## Capabilities

### New Capabilities

<!-- none -->

### Modified Capabilities

- `symphony-projects`: Symphony project creation and update now support operator-managed GitHub token secret creation.
- `control-plane-api`: Symphony create/update endpoints accept write-only token input and manage backing Secrets.
- `security-posture`: Symphony token handling must remain secret-safe across validation, audit, and error paths.
- `web-ui`: Symphony project forms capture a GitHub PAT directly and no longer require manual Secret-name entry for the standard flow.

## Impact

- Affected code: `internal/controlplaneapi/`, `internal/operator/api/v1alpha1/`, `deploy/base/api-rbac.yaml`, `web/src/ui/`, and related tests.
- Affected systems: Kubernetes Secret lifecycle for Symphony board credentials, API request/response handling, and Symphony project UX.
- No new external dependencies.
<!-- ITO:END -->
