<!-- ITO:START -->
## Why

The harness runtime image (002-04) ships a comprehensive language toolchain but lacks the coding agent CLIs that drive harness runs AND has no mechanism to inject agent credentials. Without both the binaries and auth, harness runs cannot invoke any agent. The PRD specifies three agent harnesses: Claude Code, OpenCode, and OpenAI Codex CLI. Users need to bring their subscriptions (Anthropic Pro/Max, OpenAI Plus/Pro, GitHub Copilot) via OAuth tokens — not just API keys.

## What Changes

### Agent CLI Installation

- Install **Claude Code** via the official install script (`https://claude.ai/install.sh`) — it ships as a standalone binary, not an npm package.
- Install **OpenCode** (`opencode-ai`) and **OpenAI Codex CLI** (`@openai/codex`) via `npm install -g` with pinned versions.
- Add all three to `runtime-matrix.json` for smoke test validation.

### Credential Injection from Kubernetes Secrets

- **Tier 1 — API Keys** (env vars): Create a Secret `kocao-agent-api-keys` injected via `envFrom`. Supports: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`, `GITHUB_TOKEN`, `OPENROUTER_API_KEY`.
- **Tier 2 — OAuth Tokens** (file mounts): Create a Secret `kocao-agent-oauth` with auth file contents, volume-mounted at the paths each CLI expects:
  - `opencode-auth.json` → `/home/kocao/.local/share/opencode/auth.json`
  - `codex-auth.json` → `/home/kocao/.codex/auth.json`
- Update the **operator** to mount both Secrets onto harness pods when present (optional — pods start fine without them).
- Update the **entrypoint** to fix ownership/permissions (0600) on mounted auth files.

### Auth Research Findings

| CLI | API Key Env Var | OAuth Env Var | OAuth File Path | Format |
|-----|----------------|---------------|-----------------|--------|
| Claude Code | `ANTHROPIC_API_KEY` | `CLAUDE_CODE_OAUTH_TOKEN` | _(env var only)_ | — |
| OpenCode | `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GITHUB_TOKEN`, `OPENROUTER_API_KEY` | — | `~/.local/share/opencode/auth.json` | Per-provider: `{"type":"oauth","refresh":"...","access":"...","expires":N}` |
| Codex CLI | `OPENAI_API_KEY` | — | `~/.codex/auth.json` | `{"tokens":{"access_token":"...","refresh_token":"...","id_token":"..."},"last_refresh":"..."}` |

## Capabilities

### New Capabilities

- `agent-credentials`: Kubernetes-native credential injection for agent CLIs. API keys via env vars, OAuth tokens via file mounts. One-time auth setup persists across all harness runs.

### Modified Capabilities

- `harness-runtime`: Add requirement that agent CLI binaries are pre-installed and available on PATH.

## Impact

- **Docker/build**: New install layers in `build/Dockerfile.harness`. Updated `build/harness/runtime-matrix.json`.
- **Operator**: Must support optional Secret mounts on harness pods.
- **Entrypoint**: Permission fixup for mounted auth files.
- **Deploy**: New Secret manifests and RBAC for Secret access.
- **Image size**: ~200-300 MB increase (Claude Code binary + npm packages).
- **Existing behavior preserved**: All runtimes, tools, entrypoint contract, security hardening unchanged. Secrets are optional — pods start without them.
<!-- ITO:END -->
