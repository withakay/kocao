<!-- ITO:START -->
## Why

The current harness image is too large for responsive development workflows, and cold pulls are already hurting live Kocao demos and cluster operations. Once the sandbox-agent session path is reliable, the next biggest leverage point is splitting the monolithic image into smaller runtime profiles while preserving the module 007 runtime contract.

## What Changes

- Split the harness image into a minimal base plus optional runtime profiles (for example `base`, `go`, `web`, and `full`).
- Define profile-selection rules so CLI/API/policy can choose an appropriate image automatically for a requested task or repo shape.
- Add pre-pull support for common images on dev clusters so frequent flows avoid cold-start penalties.
- Add metrics for image pull duration, time-to-ready, and time-to-first-prompt so image-profile decisions are measurable.
- Preserve the sandbox-agent runtime contract from module 007 while reducing startup latency and cluster bandwidth cost.

## Capabilities

### New Capabilities

- `harness-image-profiles`: profile-based harness image family and selection contract.

### Modified Capabilities

- `harness-runtime`: build and supervise sandbox-agent within multiple image flavors instead of one monolithic image.
- `run-execution`: harness run creation must select the right image/profile automatically and surface startup timing metrics.
- `control-plane-api`: run creation APIs and policies may accept or derive image-profile intent.

## Impact

- Affected code: `build/Dockerfile.*`, `build/harness/*`, `internal/controlplaneapi/*`, `internal/operator/*`, CLI flag selection paths, deployment manifests, CI/build pipelines, and docs/demos.
- Operations: reduces cold pull time, node bandwidth usage, and time-to-ready for agent sessions.
- Metrics: introduces pull/readiness/first-prompt timing visibility for runtime tuning.
- Module relationship: this change optimizes the module 007 sandbox-agent runtime rather than replacing it; it should preserve compatibility with the lifecycle guarantees from `009-01`.
<!-- ITO:END -->
