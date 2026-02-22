# Project Context

## Purpose

kocao is a Kubernetes-native coding agent orchestration platform. It runs short-lived "harness" pods to execute work
against a repo, with a control-plane API + operator coordinating sessions, runs, attach, and egress policy.

## Tech Stack

- Go (control-plane API + Kubernetes operator)
- Kubernetes (CRDs, controller-runtime, RBAC, NetworkPolicy)
- React + TypeScript (Vite) web UI (`web/`)
- Docker (local images; harness runtime image)
- Tooling via `make` (see `Makefile`)

## Project Conventions

### Code Style

- Go: `gofmt` is required; `go vet` runs in `make lint`.
- Prefer small, explicit helpers over clever abstractions.
- Avoid introducing new dependencies without a clear justification.
- Keep security-sensitive code (authn/authz, attach, token handling) easy to audit.

### Architecture Patterns

- Control-plane API serves `/api/v1/*` and uses bearer tokens with scope checks.
- Operator reconciles CRDs (`Session`, `HarnessRun`) and manages dependent resources (pods, PVCs, NetworkPolicy).
- Attach uses websockets to stream an interactive exec session into the harness pod.
- Egress enforcement is intended to be deny-by-default with allowlists for required endpoints.

### Testing Strategy

- Go: `make test` runs `go test ./...`.
- Go lint: `make lint` runs `gofmt` (check) + `go vet ./...`.
- Web: `pnpm -C web test` (vitest) and `pnpm -C web lint` (tsc).
- Prefer integration-style tests over heavy mocking when feasible.

### Git Workflow

- Keep commits small and focused.
- Avoid committing secrets or local env files.
- For security modules, prefer follow-on changes that are small and independently verifiable.

## Domain Context

- A "Session" is a top-level unit of work; "HarnessRuns" are per-session executions.
- Attach is the highest-risk feature: it is effectively remote shell access into the harness container.
- Egress policy is part of the security contract; "restricted" mode is deny-by-default with explicit allowlists.

## Important Constraints

- Treat the private network as hostile: require authn/authz and defense-in-depth.
- Do not put secrets (tokens, credentials) in URLs or logs.
- Enforce least-privilege RBAC for in-cluster components.
- Prefer safe defaults (restricted egress, conservative attach/websocket behavior).

## External Dependencies

- Kubernetes API server (in-cluster)
- Optional ingress/TLS termination (environment-specific)
- Git providers (e.g., GitHub) for repo clone/fetch during runs
