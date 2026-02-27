# Tasks for: 003-11_cluster-observability-dashboard

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 003-11_cluster-observability-dashboard
ito tasks next 003-11_cluster-observability-dashboard
ito tasks start 003-11_cluster-observability-dashboard 1.1
ito tasks complete 003-11_cluster-observability-dashboard 1.1
```

______________________________________________________________________

## Wave 1 - Backend Observability Endpoints

- **Depends On**: None

### Task 1.1: Add cluster overview endpoint

- **Files**: `internal/controlplaneapi/api.go`, `internal/controlplaneapi/cluster.go`
- **Dependencies**: None
- **Action**: Add `GET /api/v1/cluster-overview` returning namespace summary, deployments, pods, and non-secret runtime config indicators.
- **Verify**: `go test ./internal/controlplaneapi`
- **Done When**: endpoint returns stable JSON schema and requires authenticated read scope.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 1.2: Add pod log tail endpoint

- **Files**: `internal/controlplaneapi/api.go`, `internal/controlplaneapi/cluster.go`
- **Dependencies**: None
- **Action**: Add `GET /api/v1/pods/{podName}/logs` with optional `container` and `tailLines` query params and safe bounds checks.
- **Verify**: `go test ./internal/controlplaneapi`
- **Done When**: endpoint returns log text payload, validates inputs, and handles unavailable log backend safely.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 1.3: Update API RBAC for read-only cluster visibility

- **Files**: `deploy/base/api-rbac.yaml`
- **Dependencies**: None
- **Action**: Grant read-only access for `pods`, `pods/log`, `deployments`, and `configmaps` in namespace to `control-plane-api` Role.
- **Verify**: `kustomize build deploy/base`
- **Done When**: manifests render and API can list required resources without privilege escalation.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

______________________________________________________________________

## Wave 2 - Frontend Dashboard

- **Depends On**: Wave 1

### Task 2.1: Add cluster API client methods and types

- **Files**: `web/src/ui/lib/api.ts`
- **Dependencies**: None
- **Action**: Add typed client methods for cluster overview and pod logs endpoints.
- **Verify**: `pnpm -C web tsc --noEmit`
- **Done When**: frontend can request and render cluster data with type safety.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 2.2: Add Cluster page route and shell navigation

- **Files**: `web/src/ui/App.tsx`, `web/src/ui/components/Shell.tsx`, `web/src/ui/components/CommandPalette.tsx`
- **Dependencies**: None
- **Action**: Add `/cluster` route, sidebar link, and command palette navigation action.
- **Verify**: `pnpm -C web tsc --noEmit`
- **Done When**: users can navigate to Cluster dashboard through sidebar and command palette.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 2.3: Implement Cluster dashboard UI

- **Files**: `web/src/ui/pages/ClusterPage.tsx` (new)
- **Dependencies**: None
- **Action**: Render namespace summary metrics, deployment and pod tables, config indicators, and selectable pod log tail viewer.
- **Verify**: `pnpm -C web test`
- **Done When**: dashboard displays cluster state and can fetch logs for selected pod/container.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

______________________________________________________________________

## Wave 3 - Verification

- **Depends On**: Wave 1, Wave 2

### Task 3.1: Add/update tests for dashboard APIs and UI

- **Files**: `internal/controlplaneapi/api_test.go`, `web/src/ui/workflow.test.tsx`
- **Dependencies**: None
- **Action**: Add tests covering overview/log endpoint behavior and Cluster page render flow.
- **Verify**: `go test ./internal/controlplaneapi && pnpm -C web test`
- **Done When**: new tests pass and protect key dashboard behaviors.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 3.2: Full verification and change validation

- **Files**: all modified files
- **Dependencies**: Task 3.1
- **Action**: Run full checks: `go test`, `web tsc/test/build`, `kustomize`, and `ito validate`.
- **Verify**: `go test ./... && pnpm -C web tsc --noEmit && pnpm -C web test && pnpm -C web build && kustomize build deploy/base && ito validate 003-11_cluster-observability-dashboard --strict`
- **Done When**: all checks pass.
- **Updated At**: 2026-02-27
- **Status**: [x] complete
