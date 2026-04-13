<!-- ITO:START -->
## Why

AI coding agents working on the Kocao codebase need fast, structured code search that works across the full project without reading every file. Current agent tools (grep, glob) are sequential and don't understand code structure. Zoekt provides trigram-indexed, rank-ordered search with symbol awareness — the same engine that powers GitHub and Sourcegraph code search. Adding an agent skill with bundled wrapper scripts and an auto-indexing plugin gives every agent instant, high-quality code search with zero configuration.

## What Changes

- New agent skill at `.agents/skills/zoekt-search/` with SKILL.md, triggers, workflows, and result interpretation guidance for AI coding agents.
- Self-bootstrapping binary management: `scripts/install-zoekt.sh` downloads pre-built zoekt binaries from GitHub releases (darwin/linux, arm64/amd64), falling back to `go install` if downloads fail. Binaries are installed to skill-local `bin/` directory.
- Skill-bundled wrapper scripts (`scripts/zoekt-index.sh`, `scripts/zoekt-search.sh`) that shell out to `zoekt-index` and `zoekt` CLI binaries, hiding flag mismatches and providing a stable contract for agents. Scripts auto-trigger install if binaries are not found.
- New OpenCode plugin at `.opencode/plugins/zoekt-reindex/` that auto-reindexes on file changes or session idle via debounced hooks.
- Index stored at `.git/zoekt` per repo (gitignored, not committed). Uses `zoekt-index` (filesystem-based) rather than `zoekt-git-index` so agents can search uncommitted edits.
- Zoekt binaries managed by `scripts/install-zoekt.sh`: pre-built binaries from `github.com/sourcegraph/zoekt/releases` preferred, `go install` as fallback. No system PATH pollution — binaries live in `.agents/skills/zoekt-search/bin/`.

## Capabilities

### New Capabilities

- `zoekt-agent-skill`: Agent skill (`.agents/skills/zoekt-search/`) with triggers, workflows, result interpretation guidance, self-bootstrapping binary management (`scripts/install-zoekt.sh`), and bundled wrapper scripts (`scripts/zoekt-index.sh`, `scripts/zoekt-search.sh`) that provide a stable contract for agents. The install script downloads pre-built binaries or falls back to `go install`. The wrapper scripts auto-trigger installation if binaries are not found. Scripts hide the zoekt flag mismatch (`-index` vs `-index_dir`) and default the index path to `.git/zoekt`.
- `zoekt-opencode-plugin`: OpenCode plugin (`.opencode/plugins/zoekt-reindex/`) that auto-reindexes on file changes via debounced hooks, keeping the zoekt index fresh without manual intervention.

### Modified Capabilities

(none)

## Impact

- New skill files: `.agents/skills/zoekt-search/SKILL.md`, `.agents/skills/zoekt-search/scripts/install-zoekt.sh`, `.agents/skills/zoekt-search/scripts/zoekt-index.sh`, `.agents/skills/zoekt-search/scripts/zoekt-search.sh`
- New directory: `.agents/skills/zoekt-search/bin/` (gitignored, holds platform-specific zoekt binaries)
- New plugin: `.opencode/plugins/zoekt-reindex/`
- Dependencies: `github.com/sourcegraph/zoekt` (Go, well-maintained, Apache-2.0) — pre-built binaries from GitHub releases preferred, `go install` as fallback. Not added to the project `go.mod`.
- `.gitignore`: add `.git/zoekt/` and `.agents/skills/zoekt-search/bin/` patterns
- Supported platforms: darwin-arm64, darwin-amd64, linux-arm64, linux-amd64 (no Windows)
- No compiled wrapper binary, no `cmd/` directory, no Go build step
- No changes to existing code, APIs, or Kubernetes resources
<!-- ITO:END -->
