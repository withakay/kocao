<!-- ITO:START -->
## Context

`build/Dockerfile.harness` currently produces one monolithic Ubuntu-based harness image that installs every runtime, every agent CLI, and the sandbox-agent contract in a single build path. That image preserves the module 007 runtime contract, but it is too large for responsive dev-cluster pulls and live demo workflows.

Task 1.1 defines the split strategy without changing the runtime contract yet. The output of this task must remove ambiguity for later implementation waves: which profiles exist, what each profile contains, which layers are shared, how the images are tagged and built, and which sandbox-agent invariants every profile must preserve.

## Goals / Non-Goals

- Goals:
  - Define a concrete four-profile harness family: `base`, `go`, `web`, and `full`.
  - Keep `full` as the compatibility baseline so the current image behavior has a direct mapped successor.
  - Separate shared sandbox-agent and agent CLI layers from workload-specific runtime layers.
  - Make the future build/export workflow deterministic enough that task 2.1 can implement it directly.
  - Make sandbox-agent compatibility requirements explicit and machine-verifiable.
- Non-Goals:
  - Rewriting `build/Dockerfile.harness` in this task.
  - Introducing API or CLI profile selection in this task.
  - Changing the live pod contract, workspace paths, or agent-session behavior.
  - Optimizing exact image sizes before the per-profile builds exist.

## Decisions

### Decision: The profile family is `base`, `go`, `web`, and `full`
- `base`: minimal interactive harness for repo inspection, planning, docs, shell workflows, and sandbox-agent-backed sessions.
- `go`: `base` plus Go compilation support and shared native build tooling.
- `web`: `base` plus JavaScript/web workload runtimes.
- `full`: compatibility profile with the current broad runtime coverage.

This keeps the user-facing profile vocabulary small while still covering the main cold-start trade-offs.

### Decision: `full` is the rollout safety profile
- The current monolithic image maps directly to `full`.
- Any workflow that cannot yet classify itself safely can continue to select `full`.
- `build/harness/profile-matrix.json` records this as `compatibilityProfile` to make the no-selection/unknown-workload safety fallback explicit.
- The matrix separately records `preferredMinimalProfile` as `base` so task 1.1 can identify the smallest general-purpose profile without implying that `base` is the fallback when selection is ambiguous.
- This allows task 2.2 to add selection logic without risking regression for unknown workloads.

### Decision: Every profile keeps the same sandbox-agent control surface
Every profile must continue to provide:

- the `kocao` user (UID/GID `10001`)
- `/workspace` as the working directory
- `/usr/local/bin/kocao-harness-entrypoint`
- `/usr/local/bin/kocao-git-askpass`
- `/usr/local/bin/kocao-harness-smoke`
- `sandbox-agent`
- `claude`, `codex`, `opencode`, and `pi`
- the same sandbox-agent health and agent-catalog behavior validated today by `build/harness/smoke.sh`

This means the split is about workload runtimes, not about changing how Kocao supervises the sandbox.

### Decision: One Dockerfile, multiple named export targets
Task 2.1 should keep a single `build/Dockerfile.harness` and refactor it into reusable stages rather than maintaining one Dockerfile per profile. The target names are fixed now so later implementation can wire them directly into build automation:

- `harness-profile-base`
- `harness-profile-go`
- `harness-profile-web`
- `harness-profile-full`

Expected tags from those targets:

- `kocao/harness-runtime:<tag>-base`
- `kocao/harness-runtime:<tag>-go`
- `kocao/harness-runtime:<tag>-web`
- `kocao/harness-runtime:<tag>-full`

### Decision: Shared layers are split by responsibility
Task 2.1 should factor the Dockerfile into these logical layers and compose each exported target from those named stages/artifacts:

1. `os-common`
   - Ubuntu base image
   - common apt tooling
   - `kocao` user, workspace paths, `tini`
2. `agent-runtime`
   - the single Node runtime needed to install npm-based agent binaries
   - `sandbox-agent`, `opencode`, `codex`, `pi`, and `claude`
   - agent state directories used by the current sidecar/secret sync flow
3. `contract`
   - `runtime-matrix.json`
   - future profile metadata file
   - entrypoint, askpass helper, smoke script
4. `native-build`
   - shared compiler/build packages that are only needed for profiles that compile code
5. profile-specific runtime layers
   - `go-toolchain`
   - `web-toolchain`
   - `full-extra-toolchains`

The important rule is that the expensive, high-churn workload runtimes move out of the common sandbox-agent path. Profile relationships are therefore expressed as layer composition, not parent-image inheritance.

## Profile Matrix

The machine-readable source of truth for this task is `build/harness/profile-matrix.json`.

### Base
- Purpose: lowest-latency profile for sandbox-agent sessions, repo cloning, shell workflows, docs, planning, and lightweight edits.
- Includes:
  - `os-common`
  - `agent-runtime`
  - `contract`
- Excludes:
  - Go, Bun, Rust, .NET, Zig, and other heavyweight workload runtimes
- Compatibility note: still fully capable of hosting sandbox-agent-backed Claude/OpenCode/Codex/Pi sessions.

### Go
- Purpose: backend and CLI workloads that need Go builds/tests plus native compilation support.
- Composed from layers:
  - `os-common`
  - `agent-runtime`
  - `contract`
  - `native-build`
  - `go-toolchain`
- Excludes:
  - Bun and the full polyglot runtime set

### Web
- Purpose: JavaScript/TypeScript web workflows.
- Composed from layers:
  - `os-common`
  - `agent-runtime`
  - `contract`
  - `web-toolchain`
- Excludes:
  - Go-native compile toolchain and the full polyglot set

### Full
- Purpose: compatibility profile for unknown or broad polyglot workloads.
- Composed from layers:
  - `os-common`
  - `agent-runtime`
  - `contract`
  - `native-build`
  - `go-toolchain`
  - `web-toolchain`
  - `full-extra-toolchains`
- This is the only profile that preserves the current broad runtime footprint.

## Build Strategy

### Runtime metadata strategy
Task 2.1 should replace the single monolithic runtime metadata source with per-profile inputs while keeping the contract file path stable inside the image.

- Planned input layout:
  - `build/harness/profiles/<profile>.mise.toml`
  - `build/harness/profiles/<profile>.runtime-matrix.json`
- Stable in-image output paths:
  - `/etc/kocao/runtime-matrix.json`
  - `/etc/kocao/harness-profile.json`

This lets `kocao-harness-smoke` keep reading stable file locations while each exported profile validates only the runtimes it actually ships.

### Smoke strategy
Every exported profile must run the existing sandbox-agent contract checks plus its profile-specific runtime matrix checks during image build.

- `base` smoke: contract checks only
- `go` smoke: contract checks + Go runtime/toolchain checks
- `web` smoke: contract checks + web runtime checks
- `full` smoke: contract checks + the full runtime matrix

### Build/export order
Task 2.1 should build in this order so shared layers are maximally reusable:

1. `base`
2. `go`
3. `web`
4. `full`

`full` should be assembled from the same reusable Docker stages that provide `native-build`, `go-toolchain`, and `web-toolchain`, plus `full-extra-toolchains`, rather than using a dual-parent profile inheritance model.

### CI strategy
Task 2.1 should add a profile-aware build job rather than replacing the existing Go/Web test jobs.

- Build each named target from `build/Dockerfile.harness`
- Tag/publish each profile separately
- Run the profile-specific smoke command for each built image
- Treat `full` as the release parity gate until profile selection ships in task 1.2 / 2.2

## Risks / Trade-offs

- `base` is intentionally opinionated and may not satisfy every repo shape. That is acceptable because explicit or inferred selection can still choose `go`, `web`, or `full`.
- Keeping agent CLIs in every profile slightly increases the size of `base`, but removing them would break the sandbox-agent compatibility requirement.
- The main implementation risk is duplicated runtime installation logic across profile stages. The staged layer plan above is intended to prevent that.

## Verification Plan

- Keep the design source of truth in `build/harness/profile-matrix.json`.
- Add Go tests that validate the planned matrix, target names, build order, and sandbox-agent invariants.
- Do not treat `mock` as part of the preserved cross-profile compatibility contract; smoke coverage may still use it internally, but the profile family only guarantees the supported agent set already documented above.
- Keep `docs/sandbox-agent-integration.md` aligned with the rule that every profile preserves the same sandbox-agent surface.

## Follow-up For Task 1.2

Task 1.2 should build on this design by defining how callers request or infer one of the four profiles and how the selected profile is reported back in run/session status.
<!-- ITO:END -->
