## Context

- Symphony projects already store `spec.source.tokenSecretRef`, but the current UI exposes that low-level detail directly.
- Users are now naturally pasting raw GitHub PATs into the Secret-name field, which fails and leaks confusing error details.

## Goals / Non-Goals

- Goals:
  - Let operators create/update Symphony projects from the UI using a write-only GitHub PAT field.
  - Derive a stable Secret name from Symphony project name and GitHub owner.
  - Keep PAT values out of stored API responses, audit metadata, and user-visible errors.
- Non-Goals:
  - Replacing `tokenSecretRef` in the CRD.
  - Supporting arbitrary Secret naming from the normal UI path.
  - Adding GitHub App auth in this change.

## Decisions

- Decision: Extend the API request model with an optional write-only `githubToken` field under `source`.
  - Why: It keeps token input close to the existing GitHub source config while preserving backward-compatible CRD storage.

- Decision: Derive Secret names as `symphony-<project-name>-<github-owner>-token` using Kubernetes-safe normalization.
  - Why: Names are predictable, deterministic, and tied to the owning configuration.

- Decision: On create/update, the API creates or updates the backing Secret before persisting the `SymphonyProject`.
  - Why: This keeps the user workflow atomic from the UI perspective.

- Decision: PAT input is write-only and optional on update.
  - Why: Leaving it blank preserves the current Secret reference; providing a value rotates the stored token.

## Risks / Trade-offs

- API RBAC must be widened carefully to manage Secrets without overexposing other secret operations.
- Deterministic Secret naming can collide if normalization is too naive, so the derivation must be stable and bounded.
- Validation must distinguish between a genuine Secret name and a PAT-looking string to prevent accidental misuse.

## Migration Plan

1. Add API/UI request fields and secret-name derivation helpers.
2. Update create/update handlers to create or patch the backing Secret.
3. Update the UI to use PAT input and display the managed Secret reference read-only when editing.
4. Add regression tests for PAT redaction and secret lifecycle behavior.

## Open Questions

- None for this MVP; deterministic Secret naming and write-only PAT input are sufficient.
