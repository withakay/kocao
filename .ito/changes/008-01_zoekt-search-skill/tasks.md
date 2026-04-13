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
- **Requirements**: zoekt-agent-skill:index-script, zoekt-agent-skill:search-script
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 1.2: Add `.git/zoekt` to `.gitignore`

- **Files**: `.gitignore`
- **Dependencies**: None
- **Action**: Append `.git/zoekt/` to the project `.gitignore` so the zoekt index is never committed.
- **Verify**: `grep -q '.git/zoekt' .gitignore`
- **Done When**: `.git/zoekt/` pattern is present in `.gitignore`.
- **Requirements**: zoekt-agent-skill:default-index-location
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Create skill scaffold and `scripts/zoekt-index.sh` wrapper script

- **Files**: `.agents/skills/zoekt-search/scripts/zoekt-index.sh`
- **Dependencies**: None
- **Action**: Create the skill directory structure at `.agents/skills/zoekt-search/scripts/`. Write `zoekt-index.sh` that shells out to `zoekt-index -index <index-dir> <target-dir>`, defaulting index-dir to `.git/zoekt` and target-dir to `.`. Support `--index-dir` and `--dir` flags. Handle missing binary and non-git-repo errors. Mark executable.
- **Verify**: `bash .agents/skills/zoekt-search/scripts/zoekt-index.sh --help` (or run against a temp dir)
- **Done When**: `scripts/zoekt-index.sh` invokes `zoekt-index` with correct flags, defaults to `.git/zoekt`, handles errors gracefully.
- **Requirements**: zoekt-agent-skill:index-script, zoekt-agent-skill:default-index-location, zoekt-agent-skill:stable-agent-contract
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 2.2: Create `scripts/zoekt-search.sh` wrapper script

- **Files**: `.agents/skills/zoekt-search/scripts/zoekt-search.sh`
- **Dependencies**: Task 2.1
- **Action**: Write `zoekt-search.sh` that shells out to `zoekt -index_dir <index-dir> -jsonl <query>`, defaulting index-dir to `.git/zoekt`. Support `--index-dir` and `-n` (result limit) flags. Handle missing binary, missing index, and non-git-repo errors. Mark executable.
- **Verify**: `bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "func main"` (after indexing)
- **Done When**: `scripts/zoekt-search.sh <query>` invokes `zoekt` with correct flags, outputs JSONL, handles errors gracefully.
- **Requirements**: zoekt-agent-skill:search-script, zoekt-agent-skill:default-index-location, zoekt-agent-skill:stable-agent-contract
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Write SKILL.md with triggers, workflows, examples

- **Files**: `.agents/skills/zoekt-search/SKILL.md`
- **Dependencies**: None
- **Action**: Write the skill file with: trigger descriptions (when to use zoekt vs grep/glob), search workflow (query construction, result interpretation), index freshness guidance, and usage examples referencing the bundled scripts. Ensure the skill follows the AgentSkills specification for portability across OpenCode, Claude Code, and Codex.
- **Verify**: Skill appears in `skills_list()` output when loaded in an OpenCode session in this repo.
- **Done When**: `.agents/skills/zoekt-search/SKILL.md` exists, has correct trigger descriptions, workflow guidance, and result interpretation sections.
- **Requirements**: zoekt-agent-skill:skill-trigger-conditions, zoekt-agent-skill:search-workflow-guidance, zoekt-agent-skill:index-freshness-awareness, zoekt-agent-skill:cross-tool-portability
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 3.2: Create OpenCode reindex plugin

- **Files**: `.opencode/plugins/zoekt-reindex/index.js`
- **Dependencies**: None
- **Action**: Write an OpenCode ESM plugin that registers hooks for `file.edited` / `file.watcher.updated` (debounced, default 30s) and `session.idle`. On trigger, run `scripts/zoekt-index.sh` in the background. Log errors as warnings. Follow the existing plugin pattern in `.opencode/plugins/`.
- **Verify**: Plugin loads without error when OpenCode starts. Manual test: edit a file, wait for debounce, confirm index is updated.
- **Done When**: Plugin auto-reindexes on file changes (debounced) and session idle, runs non-blocking, logs failures without interrupting agent.
- **Requirements**: zoekt-opencode-plugin:debounced-auto-reindex, zoekt-opencode-plugin:session-idle-reindex, zoekt-opencode-plugin:non-blocking-reindex, zoekt-opencode-plugin:plugin-location-and-structure
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

______________________________________________________________________

## Wave 4

- **Depends On**: Wave 3

### Task 4.1: Integration test — index, search, verify results

- **Files**: (test script or manual verification)
- **Dependencies**: None
- **Action**: Run an integration test that: (1) creates a temp directory with known source files, (2) runs `scripts/zoekt-index.sh`, (3) runs `scripts/zoekt-search.sh` for a known pattern, (4) verifies JSONL output contains expected file paths and line numbers.
- **Verify**: End-to-end index → search → verify cycle passes with real zoekt binaries.
- **Done When**: End-to-end index→search→verify cycle passes.
- **Requirements**: zoekt-agent-skill:index-script, zoekt-agent-skill:search-script, zoekt-agent-skill:stable-agent-contract
- **Updated At**: 2026-04-13
- **Status**: [ ] pending

### Task 4.2: Showboat demo document

- **Files**: `docs/agents/zoekt-search-demo.md`
- **Dependencies**: None
- **Action**: Write a markdown demo document showing: (1) installing zoekt, (2) indexing a repo, (3) searching for patterns, (4) interpreting results, (5) how the skill and plugin work together. Include example commands and sample output. Reference the skill scripts, not a compiled binary.
- **Verify**: Document renders correctly in markdown preview.
- **Done When**: Demo document exists and accurately describes the full workflow.
- **Requirements**: zoekt-agent-skill:search-workflow-guidance
- **Updated At**: 2026-04-13
- **Status**: [ ] pending
<!-- ITO:END -->
