<!-- ITO:START -->
# Tasks for: 009-03_split-harness-image-profiles

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 009-03_split-harness-image-profiles
ito tasks next 009-03_split-harness-image-profiles
ito tasks start 009-03_split-harness-image-profiles 1.1
ito tasks complete 009-03_split-harness-image-profiles 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Define harness profile matrix and build strategy

- **Files**: `build/Dockerfile.*`, `build/harness/*`, design docs, CI build config
- **Dependencies**: None
- **Action**: Define the base/full/profile matrix, shared layers, and build pipeline needed to produce the image family.
- **Verify**: profile build plan review or profile build smoke commands
- **Done When**: There is a concrete, testable build plan for base/go/web/full profiles with preserved sandbox-agent compatibility.
- **Requirements**: harness-image-profiles:profile-based-harness-images, harness-runtime:sandbox-agent-compatibility-across-profiles
- **Updated At**: 2026-04-14
- **Status**: [x] complete

### Task 1.2: Define profile selection contract

- **Files**: `internal/controlplaneapi/*`, `internal/controlplanecli/*`, `cmd/kocao/*`, docs/specs
- **Dependencies**: Task 1.1
- **Action**: Define how explicit, policy-driven, or inferred profile selection works and where the chosen profile is surfaced.
- **Verify**: API/CLI contract review and tests
- **Done When**: The profile selection contract is documented and implementable without ambiguity.
- **Requirements**: harness-image-profiles:deterministic-profile-selection, control-plane-api:image-profile-selection-surface
- **Updated At**: 2026-04-14
- **Status**: [x] complete

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Implement profile-based harness builds

- **Files**: `build/Dockerfile.*`, `build/harness/*`, build scripts, CI pipeline config
- **Dependencies**: None
- **Action**: Build the base and profile images, keeping sandbox-agent compatibility across all supported profiles.
- **Verify**: profile build/test workflow and harness smoke verification
- **Done When**: Multiple harness image profiles build successfully and pass smoke coverage.
- **Requirements**: harness-image-profiles:profile-based-harness-images, harness-runtime:sandbox-agent-compatibility-across-profiles
- **Updated At**: 2026-04-14
- **Status**: [x] complete

### Task 2.2: Implement API/CLI profile selection and reporting

- **Files**: `internal/controlplaneapi/*`, `internal/controlplanecli/*`, `cmd/kocao/*`, tests
- **Dependencies**: None
- **Action**: Add profile selection fields/flags, defaulting behavior, and selected-profile reporting in run status.
- **Verify**: `go test ./internal/controlplaneapi/... ./internal/controlplanecli/... ./cmd/kocao/...`
- **Done When**: Clients can request or observe harness image profiles reliably.
- **Requirements**: harness-image-profiles:deterministic-profile-selection, control-plane-api:image-profile-selection-surface
- **Updated At**: 2026-04-14
- **Status**: [x] complete

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Add dev-cluster pre-pull workflows

- **Files**: deployment manifests, cluster scripts, docs, demo workflows
- **Dependencies**: None
- **Action**: Add repeatable pre-pull support for common profiles on Kind/MicroK8s/dev clusters.
- **Verify**: documented cluster prep workflow and smoke validation
- **Done When**: Dev clusters can be primed with common profiles before live demos or tests.
- **Requirements**: harness-image-profiles:development-cluster-prepull-support
- **Updated At**: 2026-04-14
- **Status**: [x] complete

### Task 3.2: Add startup performance instrumentation and demos

- **Files**: API metrics/reporting code, demos, docs, test harnesses
- **Dependencies**: None
- **Action**: Capture pull/readiness/first-prompt metrics and update demos/docs to show the improved startup path.
- **Verify**: metrics test coverage and live/dev-cluster measurement workflow
- **Done When**: Runtime profile decisions can be evaluated with real startup data.
- **Requirements**: run-execution:startup-performance-metrics
- **Updated At**: 2026-04-13
- **Status**: [ ] pending
<!-- ITO:END -->
