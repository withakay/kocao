<!-- ITO:START -->
## Why

The harness runtime and entrypoint scripts execute with high leverage: they clone repos, check out revisions,
and default to keeping the pod alive for interactive exec. Today, the entrypoint can perform dangerous operations
(`rm -rf` of a configurable repo dir) and the harness container runs as root by default. Even in single-tenant mode,
these are sharp edges and weaken defense-in-depth.

## What Changes

- Harden the harness entrypoint against path-based footguns (prevent deleting outside the workspace).
- Prevent user-provided env from overriding reserved harness variables.
- Make `git` operations safer (`--` separators; option-injection hardening).
- Harden the harness pod/container security context (run as non-root where possible; drop capabilities; read-only rootfs where possible).

## Capabilities

### New Capabilities

- `harness-runtime`: runtime hardening requirements for the harness image + entrypoint

### Modified Capabilities

<!-- none (no existing specs yet) -->

## Impact

- Affected files: `build/harness/kocao-harness-entrypoint.sh`, `build/harness/kocao-git-askpass.sh`, `build/Dockerfile.harness`, `internal/operator/controllers/pod.go`
- Behavioral impact: stricter safety checks; reduced privilege; fewer surprising failure modes
<!-- ITO:END -->
