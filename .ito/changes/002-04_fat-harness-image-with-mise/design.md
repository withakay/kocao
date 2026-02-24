<!-- ITO:START -->
## Context

The current harness image (`build/Dockerfile.harness`) is a minimal Debian bookworm-slim image with Go 1.23.2 (copied from the golang image), Node 22.11.0 (from the node base image), and Python 3.11 (apt). It carries a handful of CLI tools (bash, curl, git, jq, rg, openssh-client). There is no version manager — each runtime is baked via multi-stage copy or apt.

The PRD, architecture doc, and technical findings all specify a "fat" image philosophy: pre-install many language runtimes and tools to avoid slow/flaky per-run bootstrapping. The version manager is `mise`, pinned via a committed `mise.toml`.

The entrypoint contract (`kocao-harness-entrypoint.sh`), git-askpass helper, smoke test, and security hardening (non-root user, workspace path safety) are already in place and must be preserved.

## Goals / Non-Goals

**Goals:**
- Rebase onto Ubuntu 24.04 for broader package compatibility and alignment with PRD.
- Install `mise` and use it to manage all language runtimes with pinned versions in `mise.toml`.
- Bake multi-version runtimes: Go (2 versions), Rust (3 versions + full toolchain), Python (3 versions via uv), Node.js (3 majors incl. LTS), .NET (latest stable), Bun (latest), Zig (latest), uv (latest).
- Install compilation prerequisites (`build-essential`, `clang`, `gcc`, `g++`).
- Install developer utilities (`gh`, `jq`, `yq`, `rg`, `fd`, `tmux`, `nvim`, `git`, `curl`, `openssh-client`).
- Expand the smoke test to validate every entry in `runtime-matrix.json`.
- Keep the non-root `kocao` user and existing entrypoint/askpass/hardening contracts.

**Non-Goals:**
- Agent CLIs (OpenCode, Claude Code, Codex CLI) — separate follow-up change.
- Runtime isolation or sandboxing — out of scope.
- Image size optimization (layer squashing, multi-stage tricks) — fat image is intentional; optimize later if needed.
- Nushell, zsh, or other alternative shells — bash only per decision.

## Decisions

### Base image: Ubuntu 24.04 over Debian bookworm-slim
- PRD and architecture docs mandate Ubuntu 24.04.
- Broader package availability (especially for .NET, gh, and newer toolchains).
- Trade-off: larger base layer (~80MB vs ~30MB). Acceptable for a fat image.

### Version manager: mise
- Single tool manages Go, Rust, Python, Node, .NET, Bun, Zig.
- `mise.toml` committed alongside Dockerfile provides reproducible pinning.
- `mise install` in Dockerfile bakes everything; `mise use` at runtime switches versions.
- Alternative considered: asdf — mise is a faster, Rust-based drop-in with the same plugin ecosystem.
- Alternative considered: manual multi-stage copies — doesn't scale to 7+ languages with multiple versions.

### Python via uv
- `uv` is installed as a standalone tool (via mise or pipx).
- Python versions installed via mise use uv as the backend where possible.
- Provides fast pip-compatible package management inside harness runs.

### Rust toolchain components
- Each Rust version installs with `rustup component add rustfmt clippy rust-analyzer`.
- These are the three components most commonly expected by coding agents and CI.

### Smoke test expansion
- `smoke.sh` reads `runtime-matrix.json` and validates every entry.
- JSON schema: `{ "runtimes": { "<name>": { "versions": [...], "check": "<command>" } }, "tools": { "<name>": "<expected-version-prefix>" } }`.
- Failure is non-zero exit with a clear report of missing/mismatched tools.
- Runs as a Dockerfile `RUN` layer so CI catches breakage at build time.

### Non-root user preserved
- Keep `kocao` user (UID 10001) from the existing image.
- mise installs to `$HOME/.local/share/mise` under the kocao user.
- All runtime binaries are owned by kocao and live under mise's shim path.
- `MISE_DATA_DIR` and `PATH` set in the Dockerfile so runtimes are available without shell profile sourcing.

### Layer strategy
- Single `RUN` block for apt packages to minimize layers.
- Separate `RUN` for mise install (allows Docker cache reuse when only mise.toml changes).
- Smoke test as final `RUN` before entrypoint — build fails fast on any regression.

## Risks / Trade-offs

- **Image size ~3-5 GB**: Accepted trade-off per fat image philosophy. Monitor and optimize later if pull times become a problem (pre-pull to nodes, use image caching).
- **Build time ~10-20 min**: Rust compilation of multiple toolchains is slow. Mitigate with Docker layer caching. Consider pre-built mise cache volume in CI.
- **Upstream version churn**: mise plugins pull from upstream release channels. Pin exact versions in `mise.toml` to prevent surprise breakage. Smoke test catches drift.
- **mise shim PATH ordering**: Multiple runtime versions means the default must be explicit. Set `mise use --global <lang>@<latest>` for each language so the default is predictable.
- **.NET on Ubuntu 24.04**: Microsoft provides official apt feed for Ubuntu 24.04. If the feed breaks, fall back to mise's dotnet-core plugin.

## Open Questions

- Should the image include a pre-warmed `$CARGO_HOME` with commonly used crates, or leave that to per-run install?
- Should `mise.toml` pin exact patch versions (e.g., `go = "1.24.2"`) or minor versions (e.g., `go = "1.24"`)? Recommendation: exact patch for reproducibility.
<!-- ITO:END -->
