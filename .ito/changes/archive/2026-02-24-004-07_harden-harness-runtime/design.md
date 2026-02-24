## Context

The harness is a general-purpose runtime image intended to execute agent workflows. It must be flexible, but it should
still avoid obvious safety footguns and run with the least privilege reasonable for its purpose.

## Goals / Non-Goals

- Goals: prevent accidental `rm -rf` outside workspace; prevent reserved env overrides; harden git invocation; improve pod security context.
- Non-Goals: sandbox untrusted code strongly (would require a broader isolation design).

## Decisions

- Add explicit path validation in the entrypoint before any destructive operations.
- Treat certain env vars as reserved; operator enforces this at reconciliation time.
- Add `--` separators to `git` commands to avoid option injection.
- Add pod/container security context defaults that work with the current harness image.

## Risks / Trade-offs

- Running as non-root may require ensuring writable directories (`/workspace`, `$HOME`) are owned appropriately.

## Open Questions

- Do we want to default to `sleep infinity` for interactive attach in all environments, or require an explicit flag?
