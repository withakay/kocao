<!-- ITO:START -->
## Why

AI coding agents working on the Kocao codebase need fast, structured code search that works across the full project without reading every file. Current agent tools (grep, glob) are sequential and don't understand code structure. Zoekt provides trigram-indexed, rank-ordered search with symbol awareness — the same engine that powers GitHub and Sourcegraph code search. Adding a thin CLI wrapper, agent skill, and auto-indexing plugin gives every agent instant, high-quality code search with zero configuration.

## What Changes

- New `agent-zoekt` wrapper binary with `index` and `search` subcommands that shells out to `zoekt-index` and `zoekt` CLI, hiding flag mismatches and giving every agent the same stable contract.
- New agent skill at `.agents/skills/zoekt-search/SKILL.md` with triggers, workflows, and result interpretation guidance for AI coding agents.
- New OpenCode plugin at `.opencode/plugins/zoekt-reindex/` that auto-reindexes on file changes or session idle via debounced hooks.
- Index stored at `.git/zoekt` per repo (gitignored, not committed). Uses `zoekt-index` (filesystem-based) rather than `zoekt-git-index` so agents can search uncommitted edits.
- Zoekt binaries installed via `go install` from `github.com/sourcegraph/zoekt`.

## Capabilities

### New Capabilities

- `zoekt-wrapper-cli`: Thin CLI wrapper (`agent-zoekt`) with `index` and `search` subcommands providing a stable contract for agents, hiding the zoekt flag mismatch (`-index` vs `-index_dir`) and defaulting the index path to `.git/zoekt`.
- `zoekt-agent-skill`: Agent skill (`.agents/skills/zoekt-search/`) with triggers, workflows, and result interpretation guidance so AI coding agents know when and how to use zoekt search.
- `zoekt-opencode-plugin`: OpenCode plugin (`.opencode/plugins/zoekt-reindex/`) that auto-reindexes on file changes via debounced hooks, keeping the zoekt index fresh without manual intervention.

### Modified Capabilities

(none)

## Impact

- New Go binary: `cmd/agent-zoekt/` (thin wrapper, no complex logic)
- New skill files: `.agents/skills/zoekt-search/SKILL.md`
- New plugin: `.opencode/plugins/zoekt-reindex/`
- Dependencies: `github.com/sourcegraph/zoekt` (Go, well-maintained, Apache-2.0) — installed via `go install`, not added to the project `go.mod`
- `.gitignore`: add `.git/zoekt/` pattern
- No changes to existing code, APIs, or Kubernetes resources
<!-- ITO:END -->
