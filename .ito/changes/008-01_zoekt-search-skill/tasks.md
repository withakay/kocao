<!-- ITO:START -->
# Tasks for: 008-01_zoekt-search-skill

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 008-01_zoekt-search-skill
ito tasks next 008-01_zoekt-search-skill
ito tasks start 008-01_zoekt-search-skill 1.1
ito tasks complete 008-01_zoekt-search-skill 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Install zoekt binaries and verify they work

- **Files**: (none — external tooling install)
- **Dependencies**: None
- **Action**: Run `go install github.com/sourcegraph/zoekt/cmd/zoekt-index@latest` and `go install github.com/sourcegraph/zoekt/cmd/zoekt@latest`. Verify both binaries are on PATH and functional by indexing a small directory and running a test query.
- **Verify**: `which zoekt-index && which zoekt && zoekt-index --help && zoekt --help`
- **Done When**: `zoekt-index` and `zoekt` are installed, on PATH, and respond to `--help` without error.
- **Requirements**: zoekt-wrapper-cli:index-subcommand, zoekt-wrapper-cli:search-subcommand
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 1.2: Add `.git/zoekt` to `.gitignore`

- **Files**: `.gitignore`
- **Dependencies**: None
- **Action**: Append `.git/zoekt/` to the project `.gitignore` so the zoekt index is never committed.
- **Verify**: `grep -q '.git/zoekt' .gitignore`
- **Done When**: `.git/zoekt/` pattern is present in `.gitignore`.
- **Requirements**: zoekt-wrapper-cli:default-index-location
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Create `agent-zoekt` wrapper — `index` subcommand with tests

- **Files**: `cmd/agent-zoekt/main.go`, `cmd/agent-zoekt/main_test.go`
- **Dependencies**: None
- **Action**: Write failing tests first (TDD red). Implement the `index` subcommand that shells out to `zoekt-index -index <index-dir> <target-dir>`, defaulting index-dir to `.git/zoekt` and target-dir to `.`. Support `--index-dir` and `--dir` flags. Handle missing binary and non-git-repo errors. Make tests green, then refactor.
- **Verify**: `go test ./cmd/agent-zoekt/... -run TestIndex`
- **Done When**: `agent-zoekt index` invokes `zoekt-index` with correct flags, defaults to `.git/zoekt`, handles errors gracefully, and all tests pass.
- **Requirements**: zoekt-wrapper-cli:index-subcommand, zoekt-wrapper-cli:default-index-location, zoekt-wrapper-cli:stable-agent-contract
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 2.2: Create `agent-zoekt` wrapper — `search` subcommand with tests

- **Files**: `cmd/agent-zoekt/main.go`, `cmd/agent-zoekt/main_test.go`
- **Dependencies**: Task 2.1
- **Action**: Write failing tests first (TDD red). Implement the `search` subcommand that shells out to `zoekt -index_dir <index-dir> -jsonl <query>`, defaulting index-dir to `.git/zoekt`. Support `--index-dir` and `-n` (result limit) flags. Handle missing binary, missing index, and non-git-repo errors. Make tests green, then refactor.
- **Verify**: `go test ./cmd/agent-zoekt/... -run TestSearch`
- **Done When**: `agent-zoekt search <query>` invokes `zoekt` with correct flags, outputs JSONL, handles errors gracefully, and all tests pass.
- **Requirements**: zoekt-wrapper-cli:search-subcommand, zoekt-wrapper-cli:default-index-location, zoekt-wrapper-cli:stable-agent-contract
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Create agent skill SKILL.md

- **Files**: `.agents/skills/zoekt-search/SKILL.md`
- **Dependencies**: None
- **Action**: Write the skill file with: trigger descriptions (when to use zoekt vs grep/glob), search workflow (query construction, result interpretation), index freshness guidance, and usage examples. Ensure the skill follows the AgentSkills specification for portability across OpenCode, Claude Code, and Codex.
- **Verify**: Skill appears in `skills_list()` output when loaded in an OpenCode session in this repo.
- **Done When**: `.agents/skills/zoekt-search/SKILL.md` exists, has correct trigger descriptions, workflow guidance, and result interpretation sections.
- **Requirements**: zoekt-agent-skill:skill-trigger-conditions, zoekt-agent-skill:search-workflow-guidance, zoekt-agent-skill:index-freshness-awareness, zoekt-agent-skill:cross-tool-portability
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 3.2: Create OpenCode reindex plugin

- **Files**: `.opencode/plugins/zoekt-reindex/index.js`
- **Dependencies**: None
- **Action**: Write an OpenCode ESM plugin that registers hooks for `file.edited` / `file.watcher.updated` (debounced, default 30s) and `session.idle`. On trigger, run `agent-zoekt index` in the background. Log errors as warnings. Follow the existing plugin pattern in `.opencode/plugins/ito-skills.js`.
- **Verify**: Plugin loads without error when OpenCode starts. Manual test: edit a file, wait for debounce, confirm index is updated.
- **Done When**: Plugin auto-reindexes on file changes (debounced) and session idle, runs non-blocking, logs failures without interrupting agent.
- **Requirements**: zoekt-opencode-plugin:debounced-auto-reindex, zoekt-opencode-plugin:session-idle-reindex, zoekt-opencode-plugin:non-blocking-reindex, zoekt-opencode-plugin:plugin-location-and-structure
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

______________________________________________________________________

## Wave 4

- **Depends On**: Wave 3

### Task 4.1: Integration test — index, search, verify results

- **Files**: `cmd/agent-zoekt/integration_test.go`
- **Dependencies**: None
- **Action**: Write an integration test that: (1) creates a temp directory with known Go source files, (2) runs `agent-zoekt index`, (3) runs `agent-zoekt search` for a known pattern, (4) verifies JSONL output contains expected file paths and line numbers. Tag with `//go:build integration`.
- **Verify**: `go test ./cmd/agent-zoekt/... -tags integration -run TestIntegration`
- **Done When**: End-to-end index→search→verify cycle passes with real zoekt binaries.
- **Requirements**: zoekt-wrapper-cli:index-subcommand, zoekt-wrapper-cli:search-subcommand, zoekt-wrapper-cli:stable-agent-contract
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 4.2: Showboat demo document

- **Files**: `docs/agents/zoekt-search-demo.md`
- **Dependencies**: None
- **Action**: Write a markdown demo document showing: (1) installing zoekt, (2) indexing a repo, (3) searching for patterns, (4) interpreting results, (5) how the skill and plugin work together. Include example commands and sample output.
- **Verify**: Document renders correctly in markdown preview.
- **Done When**: Demo document exists and accurately describes the full workflow.
- **Requirements**: zoekt-agent-skill:search-workflow-guidance
- **Updated At**: 2026-04-13
- **Status**: [ ] pending
<!-- ITO:END -->
