<!-- ITO:START -->
## Why

The harness runtime image ships a minimal toolchain (Go 1.23.2, Node 22.11.0, Python 3.11, and a handful of CLI tools) but the PRD and architecture docs call for a "fat" image with mise-managed multi-version runtimes and comprehensive developer tooling. Coding agents targeting real-world repos cannot function without the expected language runtimes, compilers, and utilities being present. Every missing tool triggers a slow, flaky per-run bootstrap or outright failure.

## What Changes

- **BREAKING**: Replace the Debian bookworm-slim base with Ubuntu 24.04 to match the PRD.
- Install `mise` as the polyglot version manager; commit a `mise.toml` alongside the Dockerfile that pins all runtime versions.
- Bake multi-version language runtimes via mise (latest stable + N-1 + N-2 at image build time):
  - **.NET**: latest 1 stable version (currently .NET 9; .NET 10 when GA)
  - **Rust**: latest 3 stable versions with full toolchain (rustfmt, clippy, rust-analyzer)
  - **Python**: latest 3 stable minor versions, installed via `uv`
  - **Node.js**: latest 3 major versions (ensuring at least 1 active LTS)
  - **Go**: latest 2 stable minor versions
  - **Bun**: latest stable
  - **Zig**: latest stable
  - **uv**: latest stable
- Install compilation prerequisites: `build-essential`, `clang`, `gcc`, `g++`.
- Install developer utilities: `gh`, `jq`, `yq`, `ripgrep`, `fd`, `tmux`, `nvim`, `git`, `curl`, `openssh-client`.
- Update `runtime-matrix.json` to reflect the full toolchain inventory.
- Add a comprehensive smoke-test layer that validates every baked runtime and tool version so CI fails fast on upstream breakage.
- Preserve existing hardening: non-root user, workspace path safety, reserved env vars, restrictive security context.

## Capabilities

### New Capabilities

_(none — this enhances the existing harness-runtime capability)_

### Modified Capabilities

- `harness-runtime`: Add requirements for mise-managed multi-version runtimes, compilation prerequisites, and developer utilities. The existing hardening requirements (workspace path safety, reserved env vars, non-root container) are preserved unchanged.

## Impact

- **Docker/build**: `build/Dockerfile.harness` rewritten; new `build/harness/mise.toml` added; `build/harness/runtime-matrix.json` updated; `build/harness/smoke.sh` expanded.
- **Image size**: Will increase significantly (fat image philosophy — accepted trade-off per architecture docs). Expect 3-5 GB.
- **CI**: Image build time will increase. Smoke test layer catches version drift.
- **Existing behavior preserved**: Entrypoint contract, workspace mounts, credential injection, git-askpass, security context — all unchanged.
<!-- ITO:END -->
