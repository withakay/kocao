<!-- ITO:START -->
## Why

The harness runtime image now ships a comprehensive language toolchain (002-04) but lacks the coding agent CLIs that actually drive harness runs. Without the agent binaries pre-installed, the operator would need to bootstrap them per-run — adding latency, network dependency, and fragility. The PRD specifies three agent harnesses: Claude Code, OpenCode, and OpenAI Codex CLI.

## What Changes

- Add a Dockerfile layer that installs three agent CLIs via `npm install -g`:
  - `@anthropic-ai/claude-code` (Claude Code)
  - `opencode-ai` (OpenCode)
  - `@openai/codex` (OpenAI Codex CLI)
- Pin exact versions in the Dockerfile for reproducibility.
- Add all three CLIs to `runtime-matrix.json` so the smoke test validates their presence at build time.
- Update `smoke.sh` validation — the existing tool-check loop already handles this since the CLIs will be added to the `tools` section of the matrix.

## Capabilities

### New Capabilities

_(none — enhances existing harness-runtime capability)_

### Modified Capabilities

- `harness-runtime`: Add requirement that agent CLI binaries are pre-installed and available on PATH.

## Impact

- **Docker/build**: One additional `RUN npm install -g` layer in `build/Dockerfile.harness`. Updated `build/harness/runtime-matrix.json`.
- **Image size**: ~50-100 MB increase (Node.js packages with bundled dependencies).
- **Existing behavior preserved**: All runtimes, tools, entrypoint contract, security hardening unchanged.
<!-- ITO:END -->
