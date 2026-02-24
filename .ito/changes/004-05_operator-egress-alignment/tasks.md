# Tasks for: 004-05_operator-egress-alignment

## Execution Notes

- **Tool**: Any (OpenCode, Codex, Claude Code)
- **Mode**: Sequential (or parallel if tool supports)
- **Template**: Enhanced task format with waves, verification, and status tracking
- **Tracking**: Prefer the tasks CLI to drive status updates and pick work

```bash
ito tasks status 004-05_operator-egress-alignment
ito tasks next 004-05_operator-egress-alignment
ito tasks start 004-05_operator-egress-alignment 1.1
ito tasks complete 004-05_operator-egress-alignment 1.1
ito tasks shelve 004-05_operator-egress-alignment 1.1
ito tasks unshelve 004-05_operator-egress-alignment 1.1
ito tasks show 004-05_operator-egress-alignment
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Align API egress override request with operator enforcement

- **Files**: `internal/controlplaneapi/api.go`
- **Dependencies**: None
- **Action**: Update the egress override endpoint to reject `allowedHosts` (or any other unenforced fields) with a clear 400 error.
- **Verify**: `make lint && make test`
- **Done When**: API does not accept unenforced egress controls.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

### Task 1.2: Validate GitHub CIDR configuration robustly

- **Files**: `internal/operator/controllers/egress_policy.go`, `internal/operator/controllers/*_test.go`
- **Dependencies**: None
- **Action**: Parse `CP_GITHUB_EGRESS_CIDRS` using CIDR parsing, and make invalid entries visible (log/condition/audit).
- **Verify**: `make lint && make test`
- **Done When**: Invalid CIDRs are not silently ignored and tests cover parsing behavior.
- **Updated At**: 2026-02-23
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Document required egress configuration

- **Files**: `docs/security/posture.md`, `deploy/overlays/dev-kind/config.env`
- **Dependencies**: None
- **Action**: Document how to configure GitHub CIDRs and explain restricted vs full mode behavior.
- **Verify**: `make lint && make test`
- **Done When**: Operators can configure egress correctly without trial-and-error.
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
