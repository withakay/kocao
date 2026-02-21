# Coding Agent Orchestration Platform - Technical Findings + Discovery

Status: Draft
Last updated: 2026-02-21
Related: `docs/planning/coding-agent-orchestrator-prd.md`
Related: `docs/planning/coding-agent-orchestrator-architecture.md`

## 1. Why This Doc Exists

Capture "lego blocks" (off-the-shelf frameworks/packages) and the key technical discoveries/constraints they imply, so we minimize bespoke infrastructure.

## 2. Verified SDKs and Key External Building Blocks

### 2.0 Recommended "Lego Inventory" (MVP)

Control plane (Go):

- Operator: Kubebuilder/controller-runtime
- API: net/http + router (chi) + OpenAPI codegen (oapi-codegen or ogen)
- Auth: OIDC (coreos/go-oidc) + oauth2
- DB: Postgres + sqlc + pgx + migrations (goose/atlas)
- Cache/queue (only if needed): Redis + asynq
- Observability: zap + Prometheus + OpenTelemetry

Frontend (TS):

- React + TanStack + shadcn/ui + Kibo UI
- Attach terminal: xterm.js

Harness:

- OpenCode server + SDKs (`@opencode-ai/sdk` and `github.com/sst/opencode-sdk-go`)

### 2.1 OpenCode SDKs (Harness Integration)

JS/TS SDK (official):

- Package: `@opencode-ai/sdk`
- Install: `npm install @opencode-ai/sdk`
- Create server+client: `createOpencode()`
- Client-only mode: `createOpencodeClient({ baseUrl })`
- Supports SSE events: `client.event.subscribe()`
- Docs: https://opencode.ai/docs/sdk/

Go SDK (REST client):

- Repo: https://github.com/anomalyco/opencode-sdk-go
- Import path: `github.com/sst/opencode-sdk-go` (as shown in the repo README)
- License: MIT (per repo)
- Notes: generated REST API client; good fit for control plane calling an OpenCode server API

Implication:

- For MVP, interactive attach is not something we should assume is "built into" the OpenCode SDKs; design attach around Kubernetes pod I/O and/or OpenCode server streaming endpoints.

### 2.2 Vercel AI SDK (Future/Optional Agent Orchestration)

- Core package: `ai` (`npm install ai`)
- Provider packages: `@ai-sdk/anthropic`, `@ai-sdk/openai`, etc.
- Includes an "agents" abstraction (e.g., `ToolLoopAgent`) and streaming utilities.
- Docs: https://ai-sdk.dev/docs
- Repo: https://github.com/vercel/ai

Recommended stance:

- Use Vercel AI SDK as the "agent framework" for any TypeScript-based orchestration/services we add later (e.g., non-OpenCode harnesses, Claude SDK integrations, tool-loop services).
- For the Kubernetes/OpenCode harness MVP, it is optional; OpenCode itself already provides an agent runtime.

## 3. Kubernetes Operator + Control Plane (Go)

### 3.1 Operator Framework

Recommended:

- Kubebuilder + controller-runtime (standard operator stack in Go)

Why:

- Scaffolding for CRDs/controllers/webhooks; aligns with controller patterns and k8s API machinery.

### 3.2 Interactive Attach: "Single Driver + Viewers"

Recommended approach:

- Browser connects to platform backend via WebSocket.
- Backend bridges WebSocket <-> Kubernetes exec/attach streams using `client-go` remotecommand (SPDY) to the API server.
- Enforce attach lock in platform (driver lease) and broadcast output to viewers.

Key primitives:

- `k8s.io/client-go/tools/remotecommand` for exec/attach streaming (kubectl-like behavior)
- WebSocket server library (pick one):
  - `nhooyr.io/websocket` (modern, minimal)
  - `github.com/gorilla/websocket` (ubiquitous)

Borrow patterns from:

- Kubernetes Dashboard terminal/exec implementation (WebSocket -> remotecommand)

Why this matters:

- Kubernetes only meaningfully supports a single stdin stream for an interactive TTY; implementing "driver + viewers" is naturally a platform concern.

## 4. API, Auth, and RBAC (Go)

### 4.1 HTTP + OpenAPI

Recommended pattern:

- REST API with OpenAPI-first schema and generated clients.

Common Go tooling options (choose one and standardize):

- `oapi-codegen` (OpenAPI -> Go types/handlers)
- `ogen` (OpenAPI -> performant server/client)

Optional supporting libs:

- `github.com/getkin/kin-openapi` for OpenAPI parsing/validation in tooling

### 4.2 OIDC Authentication

Recommended:

- OIDC with PKCE for the UI and JWT validation in the API.
- Go: `github.com/coreos/go-oidc/v3/oidc` + `golang.org/x/oauth2`

### 4.3 Authorization (RBAC)

MVP approach:

- Store role bindings in Postgres (`user`/`group` -> role -> scope), enforce in API.
- Optional "lego" policy engine:
  - Casbin (policy-based RBAC/ABAC) if we want a mature evaluation engine instead of rolling our own evaluator.

Audit:

- Append-only audit events in Postgres for: run start/stop/cancel, attach driver changes, credential use, egress override.

## 5. Persistence: Postgres + sqlc + Migrations

Recommended baseline:

- Postgres
- `sqlc` for query/code generation
- Driver/pool: `pgx` (with `pgxpool`)
- Migrations: pick one
  - `goose`
  - `atlas`

## 6. Cache/Queue/Backplane (Only If Needed)

When Redis is useful:

- Cross-replica fanout for attach viewers (pub/sub) if the API is horizontally scaled.
- Work queues for async tasks (artifact upload, GitHub API retries, long-running reconciler side-jobs).

Common "lego" options:

- Redis client: `github.com/redis/go-redis/v9`
- Queue: `asynq` (Redis-backed)

## 7. Artifacts (Session Bundles)

Storage backends to consider:

- Kubernetes PVCs (backed by whatever the cluster provides: block, file, or object via CSI)

PVC-first recommendation:

- One PVC per Session.
- Harness pod mounts the PVC and reads/writes the session bundle in-place.
- Resume = new pod mounts the same PVC (sessions durable; pods ephemeral).

Abstraction note:

- Keep an internal artifact-store interface so we can later implement S3/object storage (or export/import) without rewriting orchestration.

Related Kubernetes APIs to leverage later:

- VolumeSnapshot (backup/restore)
- StorageClass parameters (encryption, reclaim policy, performance)

## 7.1 GitHub Integration

Likely "lego" blocks:

- GitHub API client for Go: `github.com/google/go-github/v66` + `golang.org/x/oauth2`

Notes:

- Even if PR creation happens inside the harness, the platform still benefits from a GitHub client for validation, repo discovery, and status checks.

## 7.2 Agent Base Images (MVP)

Goal:

- Make harness pods productive immediately. Prefer a "fat" base image over per-run bootstrap.

Base OS:

- Ubuntu 24.04.

Build strategy:

- Use `mise` as the multi-language version manager.
- Bake these runtimes into the image (pinned):
  - .NET: 10 only
  - Rust: 1.93, 1.92, 1.91
  - Python: 3.14, 3.13, 3.12 (via `uv`)
  - Node: most recent 3 major releases at build time (ensure at least 1 active LTS)
  - Go: 1.26, 1.25
  - Bun: latest
  - Zig: latest
  - uv: latest

Rust toolchain extras:

- Install the full Rust toolchain + commonly expected components (e.g., rustfmt, clippy, rust-analyzer) so agents can lint/format/build without extra downloads.

Preinstalled dev tooling:

- OpenCode (via upstream GitHub installer script)
- Claude Code
- OpenAI Codex CLI (from `openai/codex`, installed via npm: `npm install -g @openai/codex@<pinned>`)
- `gh`, `git`
- `jq`, `yq`
- `ripgrep`, `fd`
- `tmux`, `nvim`

Build dependencies:

- Install compiler toolchains and common build prereqs (at minimum: `build-essential`, `clang`, `gcc`, `g++`).

Optional base image:

- Consider building FROM a GitHub Actions runner base image (Ubuntu 24.04) if it reduces maintenance (toolchain availability, CA certs, common packages). Keep versions pinned and the build reproducible.

Operational notes:

- Expect large image sizes; mitigate with registry layer caching and infrequent rebuilds.
- Pin runtime versions in a `mise.toml` committed alongside the Dockerfile.
- Prefer non-root user in the final image; only install packages as root during build.
- Add a smoke-test layer (version checks) so CI fails fast when upstream downloads change.

## 8. Frontend (React/TanStack/shadcn/Kibo) + Attach UI

Terminal UI:

- `xterm` (xterm.js)
- Addons: `@xterm/addon-fit`, `xterm-addon-attach`

Data + state:

- `@tanstack/react-query` for API data
- `@tanstack/router` (if desired) for routing
- `zod` for runtime validation

Attach UX model:

- "Driver" sees an interactive terminal and can type.
- "Viewers" see a read-only terminal (no key handling) but same stream.
- "Take control" becomes a mutation that acquires a driver lease.

Auth in SPA:

- OIDC PKCE in the browser; exchange for API session/token.

## 9. Discovery: Decisions We Still Need to Make

High-impact decisions:

- PVC strategy: per-session PVC vs per-run PVC; retention policy; access mode (RWO vs RWX); snapshot/backup requirements.
- Attach lease semantics: when driver disconnects, immediate release vs timeout; admin override.
- Credential strategy: per-user GitHub tokens only vs shared org tokens managed by admins.
- Multi-replica story for attach: sticky sessions vs Redis pub/sub backplane.
- Whether we proxy OpenCode server APIs through the platform API (recommended) vs exposing OpenCode directly.

Nice-to-have discovery:

- Whether OpenCode SSE `event.subscribe()` can be used for some run streaming in addition to pod logs.
- Whether we want a generic "agent runtime" service later (Vercel AI SDK based) alongside OpenCode.
