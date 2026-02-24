# Tasks for: 004-02_fix-audit-config-and-rbac

## Execution Notes

- **Tool**: Any (OpenCode, Codex, Claude Code)
- **Mode**: Sequential (or parallel if tool supports)
- **Template**: Enhanced task format with waves, verification, and status tracking
- **Tracking**: Prefer the tasks CLI to drive status updates and pick work

```bash
ito tasks status 004-02_fix-audit-config-and-rbac
ito tasks next 004-02_fix-audit-config-and-rbac
ito tasks start 004-02_fix-audit-config-and-rbac 1.1
ito tasks complete 004-02_fix-audit-config-and-rbac 1.1
ito tasks shelve 004-02_fix-audit-config-and-rbac 1.1
ito tasks unshelve 004-02_fix-audit-config-and-rbac 1.1
ito tasks show 004-02_fix-audit-config-and-rbac
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Fix audit config wiring (`CP_AUDIT_PATH`)

- **Files**: `internal/config/config.go`, `cmd/control-plane-api/main.go`, `internal/controlplaneapi/api.go`, `internal/controlplaneapi/audit.go`, `internal/config/config_test.go`
- **Dependencies**: None
- **Action**: Introduce `CP_AUDIT_PATH` and rename internal fields so audit storage is not mislabeled as “DB”. Define precedence and (optionally) keep `CP_DB_PATH` as a deprecated alias.
- **Verify**: `make lint && make test`
- **Done When**: Audit log path is configured via `CP_AUDIT_PATH` and tests cover precedence/defaulting.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

### Task 1.2: Split service accounts and tighten RBAC (including `pods/exec`)

- **Files**: `deploy/base/serviceaccount.yaml`, `deploy/base/operator-rbac.yaml`, `deploy/base/api-deployment.yaml`, `deploy/base/operator-deployment.yaml`, `deploy/base/kustomization.yaml`, `deploy/overlays/dev-kind/*`
- **Dependencies**: Task 1.1
- **Action**: Create distinct service accounts for API and operator, assign least-privilege Roles, and ensure the API role includes `pods/exec` as required for attach.
- **Verify**: `kubectl apply --dry-run=client -k deploy/overlays/dev-kind`
- **Done When**: Manifests validate cleanly and RBAC is least-privilege with attach exec functional.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Document operational migration and defaults

- **Files**: `README.md`, `docs/security/posture.md`
- **Dependencies**: None
- **Action**: Document new env var (`CP_AUDIT_PATH`), deprecated alias behavior (if retained), and required RBAC expectations for attach.
- **Verify**: `make lint && make test`
- **Done When**: Docs clearly describe required env/RBAC for real deployments.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

______________________________________________________________________

## Checkpoints

### Checkpoint: Review Implementation

- **Type**: checkpoint (requires human approval)
- **Dependencies**: All Wave 2 tasks
- **Action**: Review the implementation before proceeding
- **Done When**: User confirms implementation is correct
- **Updated At**: 2026-02-23
- **Status**: [-] shelved
