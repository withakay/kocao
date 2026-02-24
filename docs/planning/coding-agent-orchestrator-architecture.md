# Coding Agent Orchestration Platform

## Architecture Decisions & Next Steps

Generated: 2026-02-21T12:28:32.001021 UTC

------------------------------------------------------------------------

# 1. Vision

Build a self-hosted Kubernetes-native orchestration platform capable of:

-   Spinning up coding agent harness pods (OpenCode initially)
-   Managing multiple agents in parallel
-   Interacting with GitHub (initially)
-   Persisting session artifacts
-   Supporting UI-driven and future event-driven execution

------------------------------------------------------------------------

# 2. Architecture Overview

## Control Plane

-   Backend: Go
-   Kubernetes Operator (controller-runtime)
-   REST API
-   PostgreSQL
-   Kustomize deployment
-   NetworkPolicies for security

## Data Plane

-   Ephemeral Harness Pods
-   Toolchain-heavy "fat" agent base image (MVP)
-   Token-based GitHub authentication
-   Restricted outbound networking by default

### Agent Base Images (MVP)

- Base OS: Ubuntu 24.04
- Philosophy: "fat" image with common dev toolchains preinstalled to avoid slow, flaky per-run bootstrap.
- Version management: `mise` installs language runtimes; bake multiple versions up front.
- Initial toolchains (baked):
  - .NET: 10 only
  - Rust: 1.93, 1.92, 1.91 (full toolchain + components)
  - Python: 3.14, 3.13, 3.12 (via `uv`)
  - Node: most recent 3 major releases at build time (ensure at least 1 is an active LTS)
  - Go: 1.26, 1.25
  - Bun: latest
  - Zig: latest
  - uv: latest
- Common dev tooling: OpenCode (installed via upstream GitHub shell script), Claude Code, Codex, `gh`, `git`, `jq`, `yq`, `ripgrep`, `fd`, `tmux`, `nvim`.
- Common dev tooling: OpenCode (installed via upstream GitHub shell script), Claude Code, OpenAI Codex CLI (from `openai/codex`, installed via npm: `npm install -g @openai/codex@<pinned>`), `gh`, `git`, `jq`, `yq`, `ripgrep`, `fd`, `tmux`, `nvim`.
- Build deps: `build-essential`, `clang`, `gcc`, `g++` (and other common compilation prerequisites).
- Optional: build FROM a GitHub Actions runner base image (Ubuntu 24.04) if it meaningfully reduces maintenance, but keep our image build reproducible and pinned.

## Frontend

-   React
-   TanStack
-   shadcn/ui
-   Kibo UI
-   Vite
-   pnpm

------------------------------------------------------------------------

# 3. Core Decisions

  Category             Decision
  -------------------- --------------------------------------
  Backend              Go
  Operator Framework   controller-runtime
  Deployment           Kustomize
  Database             PostgreSQL
  DB Access            sqlc
  Architecture Style   Modular Monolith / Vertical Slice
  Git Strategy         GitHub tokens (fine-grained)
  Secrets              Kubernetes Secrets (pluggable later)
  Networking           NetworkPolicies

------------------------------------------------------------------------

# 4. Session Strategy

-   Treat harness session data as a black-box bundle
-   For OpenCode: SQLite DB included in session artifact
-   On shutdown:
    -   Snapshot session bundle
    -   Persist externally
-   On resume:
    -   Start new pod
    -   Rehydrate bundle

Sessions are durable; harness runs are ephemeral.

------------------------------------------------------------------------

# 5. Security Model

Default harness egress: - GitHub only

Optional pre-run toggle: - Full internet access

Future: - Whitelist policies - Denylists - External policy engines

------------------------------------------------------------------------

# 6. Event Model

Initial: - UI-driven orchestration

Future: - GitHub webhook triggers - Issue label automation

------------------------------------------------------------------------

# 7. MVP Scope

Includes: - GitHub integration - Session + HarnessRun CRDs - Pod
lifecycle management - PR creation - Session artifact persistence -
Default restricted egress

------------------------------------------------------------------------

# 8. Immediate Next Steps

1.  Define CRDs
2.  Scaffold Go operator
3.  Design Postgres schema
4.  Implement sqlc layer
5.  Define NetworkPolicy templates
6.  Build harness base image
7.  Implement GitHub token injection
8.  Build minimal REST API
9.  Build React UI
10. Run full Issue → PR workflow test

------------------------------------------------------------------------

# 9. Future Enhancements

-   GitLab support
-   External secrets integration
-   Multi-tenant isolation
-   Per-project base images
-   Attach-to-running-harness mode
-   Multi-cluster support

------------------------------------------------------------------------

# 10. Cluster Web Edge (Caddy + Scalar)

- The control-plane deployment runs a same-pod Caddy edge in front of the API container.
- Caddy serves static pages at `/` and `/scalar`.
- Scalar loads the live API schema from `/openapi.json` (no bundled spec file).
- Caddy proxies `/api/v1/*`, `/openapi.json`, `/healthz`, `/readyz`, and attach websocket upgrades to `127.0.0.1:8080`.
- Dev-kind is the first target for rollout and smoke validation.
- Optional Tailscale exposure is planned as an explicit opt-in overlay path.

------------------------------------------------------------------------

End of Document
