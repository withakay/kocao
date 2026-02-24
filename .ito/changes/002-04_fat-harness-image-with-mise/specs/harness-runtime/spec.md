<!-- ITO:START -->
## ADDED Requirements

### Requirement: Mise-Managed Multi-Version Runtimes
The harness image SHALL use `mise` as the polyglot version manager and SHALL bake multiple versions of each language runtime into the image at build time. A `mise.toml` file SHALL be committed alongside the Dockerfile to pin all runtime versions reproducibly.

#### Scenario: Mise is available and functional
- **WHEN** a harness pod starts from the published image
- **THEN** `mise --version` succeeds and `mise ls` reports all expected runtimes as installed

#### Scenario: Multiple Go versions available
- **WHEN** a harness pod starts
- **THEN** Go latest stable and N-1 are installed via mise and selectable with `mise use go@<version>`

#### Scenario: Multiple Rust versions with full toolchain
- **WHEN** a harness pod starts
- **THEN** Rust latest stable, N-1, and N-2 are installed via mise, each with rustfmt, clippy, and rust-analyzer components

#### Scenario: Multiple Python versions via uv
- **WHEN** a harness pod starts
- **THEN** Python latest stable, N-1, and N-2 minor versions are installed via mise and uv, and `uv` is available as a standalone tool

#### Scenario: Multiple Node.js versions including LTS
- **WHEN** a harness pod starts
- **THEN** Node.js latest 3 major versions are installed via mise, with at least one being an active LTS release

#### Scenario: .NET runtime available
- **WHEN** a harness pod starts
- **THEN** the latest stable .NET SDK is installed via mise and `dotnet --version` succeeds

#### Scenario: Bun runtime available
- **WHEN** a harness pod starts
- **THEN** the latest stable Bun is installed via mise and `bun --version` succeeds

#### Scenario: Zig runtime available
- **WHEN** a harness pod starts
- **THEN** the latest stable Zig is installed via mise and `zig version` succeeds

### Requirement: Compilation Prerequisites
The harness image SHALL include standard compilation toolchains so that native extensions and C/C++ dependencies can be built without per-run package installation.

#### Scenario: C/C++ compilation toolchain present
- **WHEN** a harness pod starts
- **THEN** `gcc`, `g++`, `clang`, `make`, and `pkg-config` are available and functional

#### Scenario: Build-essential packages installed
- **WHEN** a harness pod starts
- **THEN** the `build-essential` metapackage (or equivalent) is installed, providing standard build tools

### Requirement: Developer Utilities
The harness image SHALL include a standard set of developer utilities required for common coding-agent workflows.

#### Scenario: Required CLI tools available
- **WHEN** a harness pod starts
- **THEN** the following tools are available and report expected versions: `gh`, `jq`, `yq`, `rg` (ripgrep), `fd`, `tmux`, `nvim`, `git`, `curl`, `openssh-client`, `bash`

### Requirement: Runtime Matrix Accuracy
The `runtime-matrix.json` file SHALL enumerate every baked runtime and CLI tool with its expected version, and the smoke test SHALL validate the matrix against the actual image contents.

#### Scenario: Smoke test validates all matrix entries
- **WHEN** the smoke test runs against the built image
- **THEN** every runtime and tool listed in `runtime-matrix.json` is present and reports a version matching the matrix entry

#### Scenario: Smoke test fails on missing tool
- **WHEN** a tool listed in `runtime-matrix.json` is missing from the image
- **THEN** the smoke test exits non-zero and reports the missing tool

### Requirement: Ubuntu 24.04 Base Image
The harness image SHALL use Ubuntu 24.04 as the base OS to align with the architecture specification and provide broad package compatibility.

#### Scenario: Base OS is Ubuntu 24.04
- **WHEN** a harness pod starts
- **THEN** `/etc/os-release` reports Ubuntu 24.04

## MODIFIED Requirements

### Requirement: Reproducible Harness Runtime Image
The system SHALL provide a versioned harness image that includes the approved runtime matrix and required coding-agent tooling for MVP workloads. The image SHALL use `mise` for runtime version management, bake all runtimes specified in `mise.toml`, and include compilation prerequisites and developer utilities. The image SHALL be built from Ubuntu 24.04.

#### Scenario: Harness pod starts with required toolchain availability
- **WHEN** a run pod starts from the published harness image
- **THEN** the required language runtimes (Go, Rust, Python, Node.js, .NET, Bun, Zig) and core CLI tools (gh, jq, yq, rg, fd, tmux, nvim, git, curl, uv) are available and report expected versions
<!-- ITO:END -->
