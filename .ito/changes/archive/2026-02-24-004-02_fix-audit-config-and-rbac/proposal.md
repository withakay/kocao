<!-- ITO:START -->
## Why

The control-plane API currently treats `CP_DB_PATH` as an audit log file path, which is confusing and likely broken intent.
Additionally, attach relies on Kubernetes `pods/exec`, but the in-cluster RBAC does not grant `pods/exec`, making attach
non-functional in a realistic deployment.

## What Changes

- Separate audit log configuration from any future DB configuration (introduce `CP_AUDIT_PATH`; deprecate the misuse of `CP_DB_PATH`).
- Fix Kubernetes RBAC so attach `pods/exec` works.
- Reduce blast radius by separating service accounts/roles for API vs operator (least privilege).

## Capabilities

### New Capabilities

- `audit-log`: audit persistence and configuration contract
- `k8s-rbac`: least-privilege Kubernetes RBAC for control-plane components

### Modified Capabilities

<!-- none (no existing specs yet) -->

## Impact

- Affected code: `internal/config/config.go`, `cmd/control-plane-api/main.go`, `internal/controlplaneapi/api.go`, `internal/controlplaneapi/audit.go`
- Affected manifests: `deploy/base/*` + `deploy/overlays/dev-kind/*`
- Operational impact: new env var `CP_AUDIT_PATH`; RBAC/service accounts change
<!-- ITO:END -->
