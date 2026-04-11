<!-- ITO:START -->
# Tasks for: 006-03_add-symphony-inline-github-token-secret

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 006-03_add-symphony-inline-github-token-secret
ito tasks next 006-03_add-symphony-inline-github-token-secret
ito tasks start 006-03_add-symphony-inline-github-token-secret 1.1
ito tasks complete 006-03_add-symphony-inline-github-token-secret 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Add API request and secret management support

- **Files**: `internal/controlplaneapi/symphony.go`, `internal/controlplaneapi/api_test.go`, `deploy/base/api-rbac.yaml`
- **Dependencies**: None
- **Action**: Extend Symphony create/update requests with write-only GitHub token input, derive Secret names, and create/update the backing Secret with redaction-safe validation and errors.
- **Verify**: `go test ./internal/controlplaneapi/...`
- **Done When**: API create/update flows manage Secrets automatically and tests cover creation, rotation, and token-safe validation failures.
- **Updated At**: 2026-03-14
- **Status**: [x] complete

### Task 1.2: Update Symphony UI for managed PAT input

- **Files**: `web/src/ui/components/SymphonyProjectForm.tsx`, `web/src/ui/lib/api.ts`, `web/src/ui/workflow.test.tsx`
- **Dependencies**: Task 1.1
- **Action**: Replace manual Secret-name entry in the standard UI flow with a write-only GitHub PAT input and derived/read-only Secret reference display.
- **Verify**: `pnpm -C web test && pnpm -C web lint`
- **Done When**: Operators can create/edit Symphony projects with an inline PAT without seeing or typing the underlying Secret name during normal entry.
- **Updated At**: 2026-03-14
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Validate and refresh local deploy path

- **Files**: docs/tests/manifests as needed
- **Dependencies**: None
- **Action**: Run full verification, ensure local kind deployment works with the new secret-management flow, and update any user-facing docs touched by the new UX.
- **Verify**: `make test && make lint && pnpm -C web test && pnpm -C web lint`
- **Done When**: The change is verified end-to-end and any touched docs remain accurate.
- **Updated At**: 2026-03-14
- **Status**: [x] complete

______________________________________________________________________

## Wave Guidelines

- Waves group tasks that can run in parallel within the wave
- Wave N depends on all prior waves completing
- Task dependencies within a wave are fine; cross-wave deps use the wave dependency
- Checkpoint waves require human approval before proceeding
<!-- ITO:END -->
