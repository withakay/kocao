# Coding Agent Orchestration Platform (MVP PRD)

Status: Draft
Last updated: 2026-02-21
Related: `docs/planning/coding-agent-orchestrator-architecture.md`

## 1. Overview

### 1.1 Problem

Teams want to use coding agents to go from issue/context to a high-quality pull request, but need a secure, self-hosted way to run agent toolchains in Kubernetes with durable sessions, controlled network access, and GitHub integration.

### 1.2 Proposed Solution

Build a Kubernetes-native orchestration platform with a Go control plane and operator that provisions ephemeral "harness pods" (OpenCode initially), persists session artifacts externally, and provides a UI-driven workflow that can create PRs in GitHub.

### 1.3 Assumptions (until confirmed)

- Deployment is self-hosted in a single Kubernetes cluster.
- Initial users are an internal engineering org (not a public SaaS).
- One company/org per cluster (single-tenant cluster), but multi-user.
- Platform provides authentication and RBAC.
- GitHub is the only SCM for MVP.
- Sessions are durable; harness pods are ephemeral (rehydrate on resume).
- Default outbound egress from harness pods is restricted to GitHub; users can opt into broader egress per run.

### 1.4 Key Technical Choices (MVP)

- Harness: OpenCode.
- SDKs: OpenCode SDK for Go (control plane) and OpenCode SDK for TypeScript (UI/client).

## 2. Goals and Non-Goals

### 2.1 Goals (MVP)

- Run coding agents in ephemeral Kubernetes pods with a toolchain-heavy base image.
- Support multiple agents/runs in parallel.
- Provide a UI-driven experience to create/manage sessions and runs.
- Integrate with GitHub using token-based auth (fine-grained PATs initially).
- Persist and restore session artifacts (treat harness state as a black-box bundle).
- Enforce default restricted egress via NetworkPolicies.
- Multi-user access with platform auth + RBAC.
- Attach/re-attach to a running harness when a user is disconnected.

### 2.2 Non-Goals (MVP)

- GitLab support.
- Multi-cluster orchestration.
- Multi-tenant isolation beyond basic RBAC.
- External secrets managers (Vault/ExternalSecrets) beyond Kubernetes Secrets.

Out of scope for MVP (unless explicitly called out elsewhere in this PRD):

- True multi-tenant isolation (hard per-org isolation within one cluster).
- Advanced policy engines for egress (OPA/Gatekeeper/etc).
- Per-project base images.

## 3. Users and Use Cases

### 3.1 Primary Personas

- Platform Admin: installs the platform, configures storage/network policies, manages base images and cluster permissions.
- Developer (Agent Runner): starts sessions/runs against GitHub repos, monitors progress/logs, reviews outputs, and opens PRs.
- Security/Admin Reviewer: validates least-privilege token handling and egress restrictions.

### 3.2 Roles (initial)

- Admin: manage system configuration, credentials, and RBAC.
- Maintainer: create/manage sessions and runs for allowed repos.
- Viewer: read-only access to allowed sessions/runs.

### 3.3 Key User Journeys

1) Issue -> Session -> Run -> PR

- User selects a repo + branch (or default), provides/chooses a GitHub token, starts a run.
- Platform creates a HarnessRun and provisions a harness pod.
- User monitors run status/logs.
- Run produces a branch and opens a PR via GitHub.

2) Resume a Session

- User opens an existing session and clicks Resume/Run.
- Platform provisions a new harness pod and rehydrates the session artifact bundle.

3) Parallel Runs

- User runs multiple sessions concurrently (different repos/branches or multiple tasks).

4) Attach to a Disconnected Harness

- A run is in progress and the user closes the UI/browser (or loses connectivity).
- User returns to the run and attaches to the running harness without restarting the pod.

Notes:

- MVP requires re-attach for disconnected clients. Attach is interactive for a single "driver" at a time, with additional read-only viewers.

## 4. Scope: Product Requirements

### 4.1 Functional Requirements

Control plane:

- Go backend providing REST API for sessions, runs, and artifact access.
- PostgreSQL persistence for platform metadata (sqlc access layer).
- Kubernetes operator (controller-runtime) managing CRDs and pod lifecycle.

Auth + RBAC:

- Users authenticate to the platform (recommended: OIDC) before accessing any API/UI.
- Enforce RBAC for sessions, runs, artifacts, and attach operations.
- Record `createdBy` and allow sharing via role bindings (user/group) scoped to session (and optionally global).
- Emit audit events for security-relevant actions (run start/stop/cancel, attach, credential use, egress overrides).

RBAC scope (MVP):

- Global: admins and system settings.
- Session: who can view/run/attach/cancel.
- Credential use: who can select/use which GitHub tokens.

Data plane:

- Harness pods created per run; pods are ephemeral.
- Token injection into harness pods to enable GitHub operations.
- Default NetworkPolicy restricting egress to GitHub; optional per-run toggle to allow broader egress.

Agent base images (MVP):

- Ubuntu 24.04 base.
- "Fat" image with common toolchains baked in (avoid per-run installs).
- Use `mise` for runtime management; bake the following versions:
  - .NET: 10
  - Rust: 1.93, 1.92, 1.91 (full toolchain)
  - Python: 3.14, 3.13, 3.12 (via `uv`)
  - Node: most recent 3 major versions at image build time (ensure includes an LTS)
  - Go: 1.26, 1.25
  - Bun: latest
  - Zig: latest
  - uv: latest
- Include developer utilities: OpenCode (via upstream GitHub installer script), Claude Code, OpenAI Codex CLI (from `openai/codex`, installed via npm: `npm install -g @openai/codex@<pinned>`), `gh`, `jq`, `yq`, `ripgrep`, `fd`, `git`, `tmux`, `nvim`.
- Include compilation prerequisites: `build-essential`, `clang`, `gcc`, `g++`.
- Additional versions may be installed at runtime via `mise` when needed.

Attach/re-attach:

- Provide an attach endpoint for a given run that proxies to the running harness.
- Re-attach works after client disconnect and does not require restarting the harness pod.
- Attach is authorized via RBAC and recorded in audit logs.

Attach semantics (MVP):

- One "driver" may attach interactively (stdin + resize).
- Additional users may attach as read-only viewers (stdout/stderr stream only).
- Driver role is transferable (explicit "Take control" action) if authorized.

Minimum attach UX (MVP):

- UI can reconnect to an in-progress run and continue streaming harness output.
- Attach uses short-lived, run-scoped credentials/tokens generated by the platform.
- Multiple viewers may attach concurrently (read-only).

Sessions and artifacts:

- On harness shutdown/completion, snapshot the harness session bundle.
- Persist bundles to Kubernetes PersistentVolumeClaims (PVCs) (implementation pluggable behind an artifact-store interface).
- On resume, restore bundle before starting harness.
- Treat OpenCode session contents as opaque (includes SQLite DB in bundle).

GitHub integration:

- Create branches and PRs from within harness.
- Capture PR URL and final run status.

OpenCode SDK usage:

- Platform uses OpenCode SDK for Go and TypeScript for harness interactions (e.g., session/run control and attach), rather than shelling out to ad-hoc scripts.

Recommended default:

- Backend uses the OpenCode Go SDK for server-side orchestration and to generate attach tokens.
- Frontend uses the OpenCode TypeScript SDK for the attached experience (streaming + optional interaction).

UI:

- Sessions list and detail views.
- Run detail view (status, timestamps, basic logs/links).
- Create session / start run flow (repo, token selection, egress toggle).

### 4.2 Non-Functional Requirements

- Security: least-privilege token handling; secrets stored in Kubernetes Secrets (pluggable later).
- Networking: restricted egress by default; explicit opt-in for broader egress.
- Reliability: runs survive control-plane restarts; sessions can be resumed after pod eviction.
- Observability: surface run state transitions; retain pod logs/events at least for debugging.
- Auditability: persist an audit trail of user actions and security-relevant changes.
- Scalability: support N concurrent runs (MVP target to be defined) without overwhelming cluster control plane.

## 5. System Design (Product-Level)

### 5.1 Core Entities (Conceptual)

- Session: durable container for harness state and metadata (repo, default branch, createdBy, timestamps).
- HarnessRun: a single execution attempt associated with a Session (status, startedAt, finishedAt, egressMode, PR URL).
- Artifact: externalized session bundle snapshot (pointer/locator + checksum + size + createdAt).
- RoleBinding: associates a user/group with a role over a scope.
- AuditEvent: immutable record of an action (who, what, when, target, outcome).

### 5.2 Kubernetes Resources

- CRDs:
  - Session
  - HarnessRun
- Controller responsibilities:
  - Create/monitor harness pods
  - Apply NetworkPolicies based on run settings
  - Update status conditions on CRDs
  - Trigger artifact snapshot/persist and restore

## 6. MVP Acceptance Criteria

- A user can create a session for a GitHub repo and start a run.
- The platform provisions a harness pod and the run reaches a terminal state (Succeeded/Failed/Cancelled).
- On success, a PR is created in GitHub and the PR URL is recorded and displayed.
- Default egress restriction is enforced for harness pods; enabling `full internet` changes applied policy.
- A completed session can be resumed: a new pod is created and the prior session bundle is restored.
- Multi-user: users must authenticate; RBAC prevents access to sessions/runs outside a user's permissions.
- A running run can be re-attached to after UI disconnect (no pod restart required).
- Attach is interactive for one driver; additional viewers are read-only.

## 7. Risks and Open Questions

- PVC lifecycle and access mode (RWO vs RWX) impacts scheduling and resume semantics.
- OIDC provider assumptions (IdP choice, group claims) impact RBAC implementation details.
- NetworkPolicy allowlisting for GitHub requires correct endpoints (github.com, api.github.com, git over HTTPS) and DNS behavior.
- Attach increases security surface area (exec/pty/log access); requires strict RBAC and auditing.

Open questions to confirm:

- Concurrency: when the driver disconnects, does the platform auto-release the driver lock immediately or after a timeout?
- OIDC: which IdP and which claim(s) define groups/roles (e.g., `groups`, `roles`)?
- GitHub credentials: per-user tokens only, or allow shared service tokens managed by admins?
- PVC strategy: per-session PVC vs per-run PVC; retention policy; snapshot/backup requirements.

## 8. Milestones (Draft)

1) Define CRDs (Session, HarnessRun)
2) Scaffold operator + reconcile loops
3) Postgres schema + sqlc layer
4) Minimal REST API
5) Harness base image + GitHub token injection
6) Artifact persist/restore (first backend)
7) NetworkPolicy templates + per-run toggle
8) Minimal React UI
9) End-to-end Issue -> PR workflow validation
