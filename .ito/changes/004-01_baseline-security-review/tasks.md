# Tasks for: 004-01_baseline-security-review

## Execution Notes

- **Tool**: Any (OpenCode, Codex, Claude Code)
- **Mode**: Sequential (or parallel if tool supports)
- **Template**: Enhanced task format with waves, verification, and status tracking
- **Tracking**: Prefer the tasks CLI to drive status updates and pick work

```bash
ito tasks status 004-01_baseline-security-review
ito tasks next 004-01_baseline-security-review
ito tasks start 004-01_baseline-security-review 1.1
ito tasks complete 004-01_baseline-security-review 1.1
ito tasks shelve 004-01_baseline-security-review 1.1
ito tasks unshelve 004-01_baseline-security-review 1.1
ito tasks show 004-01_baseline-security-review
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Write the security review report (findings + priorities)

- **Files**: `docs/security/review-2026-02-22.md`
- **Dependencies**: None
- **Action**: Document architecture, threat model, findings (broken behavior, quality issues, security gaps), and a prioritized remediation list that maps to the follow-on changes in module 004.
- **Verify**: `make test`
- **Done When**: Review report exists, is actionable, and includes concrete file/endpoint references.
- **Updated At**: 2026-02-22
- **Status**: [x] complete

### Task 1.2: Fill in Ito project context for consistent future work

- **Files**: `.ito/project.md`
- **Dependencies**: None
- **Action**: Replace placeholders with stack, architecture overview, conventions, and security/testing expectations used in this repo.
- **Verify**: `ito audit validate`
- **Done When**: `.ito/project.md` reflects real project context (no placeholder sections remain).
- **Updated At**: 2026-02-22
- **Status**: [x] complete

### Task 1.3: Add user-facing security posture docs

- **Files**: `SECURITY.md`, `docs/security/posture.md`
- **Dependencies**: None
- **Action**: Document supported deployment modes, threat assumptions, and operator guidance for tokens, RBAC, and egress configuration.
- **Verify**: `make lint`
- **Done When**: Security docs exist and match the `security-posture` requirements.
- **Updated At**: 2026-02-22
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Add lightweight security checks to CI (lint/static)

- **Files**: `Makefile`, `.github/workflows/**` (if added), `docs/security/review-2026-02-22.md`
- **Dependencies**: None
- **Action**: Add/standardize baseline checks (at minimum `make lint` + `make test`). Optionally evaluate `gosec` and document justification if not adopted.
- **Verify**: `make lint && make test`
- **Done When**: CI path is documented and repeatable; security checks are tracked and visible.
- **Updated At**: 2026-02-22
- **Status**: [x] complete

______________________________________________________________________

## Checkpoints

### Checkpoint: Review Implementation

- **Type**: checkpoint (requires human approval)
- **Dependencies**: All Wave 2 tasks
- **Action**: Review the implementation before proceeding
- **Done When**: User confirms implementation is correct
- **Updated At**: 2026-02-22
- **Status**: [x] complete
