<!-- ITO:START -->
# Tasks for: 002-04_fat-harness-image-with-mise

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 002-04_fat-harness-image-with-mise
ito tasks next 002-04_fat-harness-image-with-mise
ito tasks start 002-04_fat-harness-image-with-mise 1.1
ito tasks complete 002-04_fat-harness-image-with-mise 1.1
```

______________________________________________________________________

## Wave 1: Base image and mise foundation

- **Depends On**: None

### Task 1.1: Rebase Dockerfile onto Ubuntu 24.04 with mise

- **Files**: `build/Dockerfile.harness`, `build/harness/mise.toml`
- **Dependencies**: None
- **Action**: Rewrite `Dockerfile.harness` to use `ubuntu:24.04` as the base. Install `mise` via the official installer script. Create `build/harness/mise.toml` with pinned versions for all runtimes (use current latest stable at authoring time). Set `MISE_DATA_DIR`, update `PATH` to include mise shims. Preserve the `kocao` user (UID 10001), `/workspace` workdir, and tini entrypoint.
- **Verify**: `docker build -f build/Dockerfile.harness -t kocao/harness-runtime:dev-fat --target base-with-mise . && docker run --rm kocao/harness-runtime:dev-fat mise --version`
- **Done When**: Image builds from Ubuntu 24.04, mise is installed and functional, `mise --version` succeeds, kocao user exists.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 1.2: Install compilation prerequisites and system packages

- **Files**: `build/Dockerfile.harness`
- **Dependencies**: Task 1.1
- **Action**: Add a `RUN` layer installing: `build-essential`, `clang`, `gcc`, `g++`, `pkg-config`, `libssl-dev`, `cmake`, `curl`, `git`, `openssh-client`, `bash`, `tmux`, `jq`. Use a single `apt-get install` to minimize layers.
- **Verify**: `docker run --rm kocao/harness-runtime:dev-fat bash -c "gcc --version && clang --version && make --version && pkg-config --version"`
- **Done When**: All compilation prerequisites are available in the image.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Wave 2: Language runtimes via mise

- **Depends On**: Wave 1

### Task 2.1: Install Go runtimes (latest + N-1)

- **Files**: `build/harness/mise.toml`, `build/Dockerfile.harness`
- **Dependencies**: None
- **Action**: Add Go latest stable and N-1 to `mise.toml`. Run `mise install` in the Dockerfile. Set the latest as the global default.
- **Verify**: `docker run --rm kocao/harness-runtime:dev-fat bash -c "go version && mise ls go"`
- **Done When**: Two Go versions installed, latest is default, both selectable via `mise use`.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 2.2: Install Rust toolchains (latest + N-1 + N-2)

- **Files**: `build/harness/mise.toml`, `build/Dockerfile.harness`
- **Dependencies**: None
- **Action**: Add Rust latest stable, N-1, N-2 to `mise.toml`. Install each with `rustup component add rustfmt clippy rust-analyzer`. Set latest as global default. Note: mise's rust plugin uses rustup under the hood — ensure components are added per-version.
- **Verify**: `docker run --rm kocao/harness-runtime:dev-fat bash -c "rustc --version && rustfmt --version && clippy-driver --version && rust-analyzer --version && mise ls rust"`
- **Done When**: Three Rust versions installed, each with rustfmt/clippy/rust-analyzer, latest is default.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 2.3: Install Python runtimes via uv (latest + N-1 + N-2)

- **Files**: `build/harness/mise.toml`, `build/Dockerfile.harness`
- **Dependencies**: None
- **Action**: Install `uv` (latest) via mise. Install Python latest stable, N-1, N-2 minor versions via mise (using uv backend where supported). Set latest as global default. Ensure `python3`, `pip`, and `uv` are all on PATH.
- **Verify**: `docker run --rm kocao/harness-runtime:dev-fat bash -c "python3 --version && uv --version && mise ls python"`
- **Done When**: Three Python versions installed, uv available, latest is default.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 2.4: Install Node.js runtimes (latest 3 majors incl. LTS)

- **Files**: `build/harness/mise.toml`, `build/Dockerfile.harness`
- **Dependencies**: None
- **Action**: Add Node.js latest 3 major versions to `mise.toml`, ensuring at least one is an active LTS. Set the LTS as the global default. npm ships with Node.
- **Verify**: `docker run --rm kocao/harness-runtime:dev-fat bash -c "node --version && npm --version && mise ls node"`
- **Done When**: Three Node.js majors installed, LTS is default, npm available.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 2.5: Install .NET, Bun, and Zig

- **Files**: `build/harness/mise.toml`, `build/Dockerfile.harness`
- **Dependencies**: None
- **Action**: Add .NET latest stable SDK, Bun latest, and Zig latest to `mise.toml`. Run `mise install`. Set each as global default for its tool.
- **Verify**: `docker run --rm kocao/harness-runtime:dev-fat bash -c "dotnet --version && bun --version && zig version"`
- **Done When**: .NET SDK, Bun, and Zig are installed and report correct versions.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

______________________________________________________________________

## Wave 3: Developer utilities and smoke test

- **Depends On**: Wave 2

### Task 3.1: Install remaining developer utilities

- **Files**: `build/Dockerfile.harness`
- **Dependencies**: None
- **Action**: Install `gh` (GitHub CLI, via official apt repo or mise), `yq` (via binary download or mise), `fd-find` (apt or mise), `nvim` (apt or AppImage). Ensure `rg` (ripgrep) is still present (may already be via apt). Verify all tools are on PATH for the kocao user.
- **Verify**: `docker run --rm kocao/harness-runtime:dev-fat bash -c "gh --version && yq --version && rg --version && fd --version && nvim --version && tmux -V"`
- **Done When**: All developer utilities from the spec are available and functional.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 3.2: Update runtime-matrix.json and expand smoke test

- **Files**: `build/harness/runtime-matrix.json`, `build/harness/smoke.sh`
- **Dependencies**: Task 3.1
- **Action**: Rewrite `runtime-matrix.json` to enumerate every baked runtime (with versions) and tool. Expand `smoke.sh` to read the matrix and validate each entry — check binary existence and version output. Exit non-zero with a clear report on any mismatch. Add the smoke test as a Dockerfile `RUN` layer so builds fail fast.
- **Verify**: `docker build -f build/Dockerfile.harness -t kocao/harness-runtime:dev-fat . && docker run --rm kocao/harness-runtime:dev-fat /usr/local/bin/kocao-harness-smoke`
- **Done When**: `runtime-matrix.json` covers all tools, smoke test validates every entry, build passes with smoke test layer.
- **Updated At**: 2026-02-24
- **Status**: [x] complete

### Task 3.3: Final integration build and Kind cluster test

- **Files**: `Makefile` (if targets need updating)
- **Dependencies**: Task 3.2
- **Action**: Run `make images` to build the final image. Load into Kind with `make kind-load-images`. Restart operator/API deployments. Create a test harness run and verify the pod starts with the full toolchain available.
- **Verify**: `make images && make kind-load-images && kubectl rollout restart deployment/control-plane-operator -n kocao-system && kubectl rollout status deployment/control-plane-operator -n kocao-system --timeout=60s`
- **Done When**: Harness run pod starts successfully with the fat image, smoke test passes inside the pod, no regressions in existing entrypoint/workspace/credential behavior.
- **Updated At**: 2026-02-24
- **Status**: [x] complete
<!-- ITO:END -->
