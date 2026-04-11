<!-- ITO:START -->
## Context

The harness runtime image (002-04) has mise-managed language runtimes and developer tooling but no agent CLIs and no credential injection mechanism. Users need to bring their subscriptions (Anthropic Pro/Max, OpenAI Plus/Pro, GitHub Copilot) via OAuth tokens — not just API keys.

The operator's `buildHarnessPod()` already supports Secret-backed volume mounts for git auth (SecretVolumeSource + env vars). Agent credential injection follows the same pattern.

## Goals / Non-Goals

**Goals:**
- Install Claude Code, OpenCode, and Codex CLI into the harness image.
- Support Tier 1 (API key env vars) and Tier 2 (OAuth token file mounts) credential injection from Kubernetes Secrets.
- Add `spec.agentAuth` to the HarnessRun CRD with `apiKeySecretName` and `oauthSecretName` fields.
- Update the operator's pod builder to mount these Secrets when referenced.
- Add smoke test coverage for all three CLIs.

**Non-Goals:**
- CLI for importing auth (`kocao auth import-*`) — follow-up change.
- Agent configuration (AGENTS.md, model settings) — runtime concern, not build-time.
- Token refresh orchestration — the CLIs handle this themselves using refresh tokens.
- Web UI for credential management — follow-up.

## Auth Research Findings

### Claude Code
- **API key**: `ANTHROPIC_API_KEY` env var.
- **OAuth**: `CLAUDE_CODE_OAUTH_TOKEN` env var (discovered from anthropics/claude-code-action source). No file-based credential store needed.
- **Install method**: Standalone binary via `curl -fsSL https://claude.ai/install.sh | bash` (not npm).

### OpenCode
- **API keys**: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, `GITHUB_TOKEN`, `OPENROUTER_API_KEY` env vars.
- **OAuth**: File at `~/.local/share/opencode/auth.json` (mode 0600). Per-provider schema:
  ```json
  {
    "<provider>": {
      "type": "oauth",
      "refresh": "sk-ant-ort01-...",
      "access": "sk-ant-oat01-...",
      "expires": 1771997312887
    }
  }
  ```
  Supports: anthropic, openai, github-copilot, openrouter. Refresh tokens enable auto-renewal.
- **Install method**: `npm install -g opencode-ai`.

### Codex CLI
- **API key**: `OPENAI_API_KEY` env var.
- **OAuth**: File at `~/.codex/auth.json` (mode 0600). Schema:
  ```json
  {
    "tokens": {
      "id_token": "eyJ...",
      "access_token": "eyJ...",
      "refresh_token": "rt_...",
      "account_id": "..."
    },
    "last_refresh": "2026-02-14T13:32:34.407273Z"
  }
  ```
  Refresh tokens enable auto-renewal. `CODEX_HOME` env var overrides `~/.codex` base path.
- **Install method**: `npm install -g @openai/codex`.

## Decisions

### Two-tier Secret model
- **Tier 1 (API keys)**: A single Secret with env var keys (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`, `GITHUB_TOKEN`, `OPENROUTER_API_KEY`). Injected via `envFrom: secretRef`. Simplest path for users with API keys.
- **Tier 2 (OAuth tokens)**: A single Secret with auth file contents as data keys. Projected as files at the paths each CLI expects. For subscription users who OAuth'd locally and want to bring those tokens to the cluster.

### Both Secrets are optional
- If `spec.agentAuth` is nil, the pod starts normally — no credential injection.
- If only `apiKeySecretName` is set, only env vars are injected.
- If only `oauthSecretName` is set, only files are mounted.
- Both can be set simultaneously (env vars + file mounts).

### File permissions via fsGroup + defaultMode
- The harness pod already runs as UID/GID 10001 with `FSGroup: 10001`.
- OAuth Secret volume uses `defaultMode: 0600` so files are owner-readable only.
- No entrypoint changes needed — Kubernetes handles the permissions via the security context.

### Claude Code install via official script, not npm
- Claude Code is a compiled binary (Mach-O on macOS, ELF on Linux) distributed via `https://claude.ai/install.sh`.
- The Dockerfile runs the installer with the target version.
- The npm package `@anthropic-ai/claude-code` may exist but the official install method is the script.

### CRD extension: spec.agentAuth
```yaml
spec:
  agentAuth:
    apiKeySecretName: "kocao-agent-api-keys"   # optional
    oauthSecretName: "kocao-agent-oauth"        # optional
```
Both fields are optional strings referencing Secrets in the same namespace.

### OAuth file mount paths
| Secret Key | Mount Path | CLI |
|---|---|---|
| `opencode-auth.json` | `/home/kocao/.local/share/opencode/auth.json` | OpenCode |
| `codex-auth.json` | `/home/kocao/.codex/auth.json` | Codex CLI |

Claude Code doesn't need a file mount — `CLAUDE_CODE_OAUTH_TOKEN` env var covers it.

## Risks / Trade-offs

- **OAuth token expiry**: Refresh tokens have long (but not infinite) lifetimes. If a refresh token expires, the user must re-auth locally and update the Secret. The CLIs handle access token refresh automatically.
- **Secret sprawl**: Two Secrets per namespace. Acceptable for MVP; a unified Secret with subpath projections could consolidate later.
- **Claude Code binary size**: ~100-200 MB. Accepted for fat image philosophy.
- **RBAC**: Operator needs `get` on Secrets in the run namespace. Already needed for git auth Secrets.

## Open Questions

_(none — all resolved by research)_
<!-- ITO:END -->
