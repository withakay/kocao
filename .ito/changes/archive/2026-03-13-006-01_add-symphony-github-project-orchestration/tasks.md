<!-- ITO:START -->
# Tasks for: 006-01_add-symphony-github-project-orchestration

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 006-01_add-symphony-github-project-orchestration
ito tasks next 006-01_add-symphony-github-project-orchestration
ito tasks start 006-01_add-symphony-github-project-orchestration 1.1
ito tasks complete 006-01_add-symphony-github-project-orchestration 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Define Symphony project API and CRD types

- **Files**: `internal/operator/api/v1alpha1/types.go`, `internal/operator/api/v1alpha1/deepcopy.go`, generated CRD manifests, `internal/operator/controllers/constants.go`
- **Dependencies**: None
- **Action**: Add the `SymphonyProject` resource, repository configuration model, status shape, labels, and ownership metadata needed for orchestration.
- **Verify**: `go test ./internal/operator/...`
- **Done When**: The new resource types compile, generated artifacts are updated, and unit tests cover basic defaults/validation.
- **Updated At**: 2026-03-09
- **Status**: [x] complete

### Task 1.2: Add RBAC and secret-access contracts for Symphony

- **Files**: `deploy/base/operator-rbac.yaml`, `internal/operator/controllers/`, `internal/controlplaneapi/auth.go`, audit wiring files
- **Dependencies**: Task 1.1
- **Action**: Grant the operator the minimum access needed to read referenced GitHub/project secrets, and document or encode the new authz and audit scope requirements.
- **Verify**: `go test ./internal/operator/... ./internal/controlplaneapi/...`
- **Done When**: Secret access is least-privilege, audit hooks compile, and new policy paths are covered by tests.
- **Updated At**: 2026-03-09
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Implement GitHub Projects v2 source client

- **Files**: new GitHub client package under `internal/`, `internal/config/`, source normalization tests
- **Dependencies**: None
- **Action**: Implement GitHub Projects v2 polling, issue-backed item normalization, repo allowlist evaluation, and skip-reason reporting for unsupported items.
- **Verify**: `go test ./internal/...`
- **Done When**: The source client can load candidate items, normalize issue data, and distinguish eligible vs skipped items in tests.
- **Updated At**: 2026-03-09
- **Status**: [x] complete

### Task 2.2: Implement Symphony orchestration controller loop

- **Files**: `internal/operator/controllers/`, new Symphony orchestration package(s), controller tests
- **Dependencies**: Task 2.1
- **Action**: Add polling, claim, retry, reconciliation, and bounded status updates for `SymphonyProject` resources inside the operator.
- **Verify**: `go test ./internal/operator/...`
- **Done When**: The controller claims eligible work, avoids duplicate execution, and records retry/reconcile outcomes in tests.
- **Updated At**: 2026-03-09
- **Status**: [x] complete

### Task 2.3: Materialize claims as Session and HarnessRun execution

- **Files**: `internal/operator/controllers/`, `internal/controlplaneapi/`, existing Session/HarnessRun helpers and tests
- **Dependencies**: Task 2.2
- **Action**: Map each claimed issue to a durable Session plus per-attempt HarnessRun, including repository-specific auth, repo revision, labels, and lifecycle cleanup.
- **Verify**: `go test ./internal/operator/... ./internal/controlplaneapi/...`
- **Done When**: Claimed work creates or reuses the right Session, launches HarnessRuns, and links child resources back to the Symphony project in tests.
- **Updated At**: 2026-03-10
- **Status**: [x] complete

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Add Symphony control-plane API endpoints and scopes

- **Files**: `internal/controlplaneapi/api.go`, `internal/controlplaneapi/openapi.go`, `internal/controlplaneapi/api_test.go`, CLI client files under `internal/controlplanecli/`
- **Dependencies**: None
- **Action**: Add list/create/get/control endpoints plus auth scopes and client bindings for Symphony projects.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/controlplanecli/...`
- **Done When**: API and CLI paths expose Symphony project CRUD/control behavior with authz and serialization tests.
- **Updated At**: 2026-03-10
- **Status**: [x] complete

### Task 3.2: Expose runtime detail and observability payloads

- **Files**: `web/src/`, route definitions, UI tests, related API client bindings
- **Dependencies**: None
- **Action**: Add navigation, list views, and create/edit flows for Symphony project configuration in the web app.
- **Verify**: `pnpm -C web test && pnpm -C web lint`
- **Done When**: Operators can navigate to Symphony, create a project, and edit configuration with passing UI tests.
- **Updated At**: 2026-03-10
- **Status**: [x] complete

### Task 3.3: Build Symphony runtime detail and operator controls UI

- **Files**: `web/src/`, API hooks/clients, UI tests
- **Dependencies**: Task 3.2
- **Action**: Add project detail screens showing active/retrying work, skip reasons, child run links, and pause/resume/refresh controls.
- **Verify**: `pnpm -C web test && pnpm -C web lint`
- **Done When**: The Symphony detail view renders runtime state and operator controls correctly with passing tests.
- **Updated At**: 2026-03-10
- **Status**: [x] complete

______________________________________________________________________

## Wave 4

- **Depends On**: Wave 3

### Task 4.1: Expose runtime detail and observability payloads

- **Files**: `internal/controlplaneapi/`, `internal/operator/controllers/`, API/OpenAPI tests
- **Dependencies**: None
- **Action**: Add bounded runtime detail payloads for active items, retries, recent errors, linked child objects, and aggregate counters.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/operator/...`
- **Done When**: Symphony detail responses include the agreed runtime fields and remain covered by API/controller tests.
- **Updated At**: 2026-03-10
- **Status**: [x] complete

______________________________________________________________________

## Wave 5

- **Depends On**: Wave 4

### Task 5.1: Harden audit, secret redaction, and egress defaults

- **Files**: `internal/controlplaneapi/`, `internal/operator/controllers/`, security/audit tests, deployment manifests
- **Dependencies**: None
- **Action**: Finish security hardening for Symphony secrets, audit events, skip-reason visibility, and restricted-by-default worker egress.
- **Verify**: `make test && make lint`
- **Done When**: Symphony security-sensitive paths are covered by tests and default deployment posture remains least-privilege.
- **Updated At**: 2026-03-10
- **Status**: [x] complete

### Task 5.2: Add end-to-end validation and docs for GitHub-backed orchestration

- **Files**: integration tests, docs, example configuration manifests, `README` or operator docs as needed
- **Dependencies**: Task 5.1
- **Action**: Add integration coverage and operator documentation for configuring a Symphony project, secrets, repository mappings, and expected runtime behavior.
- **Verify**: `make test && pnpm -C web test && pnpm -C web lint`
- **Done When**: Documentation covers setup and the end-to-end test suite validates the primary GitHub-backed Symphony flow.
- **Updated At**: 2026-03-10
- **Status**: [x] complete

______________________________________________________________________

## Wave Guidelines

- Waves group tasks that can run in parallel within the wave
- Wave N depends on all prior waves completing
- Task dependencies within a wave are fine; cross-wave deps use the wave dependency
- Checkpoint waves require human approval before proceeding
<!-- ITO:END -->
