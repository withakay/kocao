<!-- ITO:START -->
## ADDED Requirements

### Requirement: Skill Trigger Conditions

The zoekt search skill SHALL be loaded when an agent needs to find code patterns, function definitions, type declarations, usages of a symbol, or navigate unfamiliar parts of the codebase. The skill MUST define clear trigger descriptions so OpenCode, Claude Code, and Codex can auto-load it.

- **Requirement ID**: zoekt-agent-skill:skill-trigger-conditions

#### Scenario: Agent needs structural code search

- **WHEN** an agent is asked to find all implementations of an interface, usages of a function, or files matching a code pattern
- **THEN** the skill is loaded and the agent uses `scripts/zoekt-search.sh` instead of sequential grep/glob

#### Scenario: Agent explores unfamiliar codebase area

- **WHEN** an agent needs to understand a subsystem it has not read files from
- **THEN** the skill guides the agent to search for relevant types, functions, and packages before reading individual files

### Requirement: Search Workflow Guidance

The skill SHALL provide a structured workflow for when and how to use zoekt search, including query construction tips, result interpretation, and follow-up actions.

- **Requirement ID**: zoekt-agent-skill:search-workflow-guidance

#### Scenario: Skill provides query construction guidance

- **WHEN** the skill is loaded
- **THEN** the agent has access to guidance on constructing effective zoekt queries (literal strings, regex patterns, file filters, symbol searches)

#### Scenario: Skill provides result interpretation

- **WHEN** the agent receives JSONL search results
- **THEN** the skill provides guidance on interpreting fields (file path, line number, matched content, score) and how to use results for next steps

### Requirement: Install Script

The skill SHALL bundle a `scripts/install-zoekt.sh` script that manages zoekt binary installation. The script MUST detect the current platform (darwin/linux) and architecture (arm64/amd64). It MUST first attempt to download pre-built binaries from `github.com/sourcegraph/zoekt/releases`. If the download fails (no release for the platform, network error, checksum mismatch, etc.), the script MUST fall back to building from source via `go install github.com/sourcegraph/zoekt/cmd/zoekt-index@latest` and `go install github.com/sourcegraph/zoekt/cmd/zoekt@latest`. Binaries MUST be installed to the skill-local `bin/` directory (`.agents/skills/zoekt-search/bin/`), not to any system-wide location. The script MUST support darwin-arm64, darwin-amd64, linux-arm64, and linux-amd64. The script SHOULD be idempotent — skipping download if binaries already exist and are the correct version.

- **Requirement ID**: zoekt-agent-skill:install-script

#### Scenario: Download pre-built binaries on supported platform

- **WHEN** `scripts/install-zoekt.sh` is run on darwin-arm64 and a matching release exists
- **THEN** pre-built `zoekt-index` and `zoekt` binaries are downloaded and placed in `.agents/skills/zoekt-search/bin/`

#### Scenario: Fall back to go install when download fails

- **WHEN** `scripts/install-zoekt.sh` is run and the binary download fails (network error, no release for platform, etc.)
- **THEN** the script runs `go install github.com/sourcegraph/zoekt/cmd/zoekt-index@latest` and `go install github.com/sourcegraph/zoekt/cmd/zoekt@latest`, then copies the resulting binaries to `.agents/skills/zoekt-search/bin/`

#### Scenario: Skip install when correct version already exists

- **WHEN** `scripts/install-zoekt.sh` is run and the correct version of both binaries already exists in `bin/`
- **THEN** the script exits successfully without downloading or building

#### Scenario: Unsupported platform

- **WHEN** `scripts/install-zoekt.sh` is run on an unsupported platform (e.g., Windows, freebsd)
- **THEN** the script exits with a non-zero status and prints an error listing supported platforms

#### Scenario: No Go toolchain for fallback

- **WHEN** binary download fails and `go` is not on PATH
- **THEN** the script exits with a non-zero status and prints an error indicating that Go is required for building from source

### Requirement: Index Script

The skill SHALL bundle a `scripts/zoekt-index.sh` wrapper script that invokes `zoekt-index` against the current working directory, storing the index at `.git/zoekt` by default. The script SHALL use filesystem-based indexing (not git-based) so that uncommitted edits are searchable. The script hides the zoekt `-index` flag behind a consistent `--index-dir` option. The script MUST check the skill-local `bin/` directory first for the `zoekt-index` binary, then fall back to PATH, and if neither is found, MUST auto-trigger `scripts/install-zoekt.sh` before retrying.

- **Requirement ID**: zoekt-agent-skill:index-script

#### Scenario: Index the current repository

- **WHEN** `scripts/zoekt-index.sh` is run in a git repository without arguments
- **THEN** `zoekt-index -index .git/zoekt .` is executed and the index is written to `.git/zoekt/`

#### Scenario: Index with custom path

- **WHEN** `scripts/zoekt-index.sh --index-dir /tmp/myindex` is run
- **THEN** `zoekt-index -index /tmp/myindex .` is executed and the index is written to the specified directory

#### Scenario: Index target directory

- **WHEN** `scripts/zoekt-index.sh --dir /some/path` is run
- **THEN** `zoekt-index -index .git/zoekt /some/path` is executed, indexing the specified directory

#### Scenario: zoekt-index binary not found — auto-install

- **WHEN** `scripts/zoekt-index.sh` is run and `zoekt-index` is not in the skill-local `bin/` directory or on PATH
- **THEN** the script runs `scripts/install-zoekt.sh` automatically, then retries the index operation

#### Scenario: Auto-install fails

- **WHEN** `scripts/zoekt-index.sh` triggers `scripts/install-zoekt.sh` and the install fails
- **THEN** the script exits with a non-zero status and prints the install error

### Requirement: Search Script

The skill SHALL bundle a `scripts/zoekt-search.sh` wrapper script that invokes `zoekt` against the index at `.git/zoekt` by default, passing the query and outputting results in JSONL format. The script hides the zoekt `-index_dir` flag behind a consistent `--index-dir` option. The script MUST check the skill-local `bin/` directory first for the `zoekt` binary, then fall back to PATH, and if neither is found, MUST auto-trigger `scripts/install-zoekt.sh` before retrying.

- **Requirement ID**: zoekt-agent-skill:search-script

#### Scenario: Search with default index

- **WHEN** `scripts/zoekt-search.sh "func main"` is run in a git repository
- **THEN** `zoekt -index_dir .git/zoekt -jsonl "func main"` is executed and JSONL results are printed to stdout

#### Scenario: Search with custom index directory

- **WHEN** `scripts/zoekt-search.sh --index-dir /tmp/myindex "func main"` is run
- **THEN** `zoekt -index_dir /tmp/myindex -jsonl "func main"` is executed

#### Scenario: Search with result limit

- **WHEN** `scripts/zoekt-search.sh -n 20 "pattern"` is run
- **THEN** the search returns at most 20 results

#### Scenario: No index exists

- **WHEN** `scripts/zoekt-search.sh "query"` is run but `.git/zoekt` does not exist
- **THEN** the script exits with a non-zero status and prints an error message suggesting the user run `scripts/zoekt-index.sh` first

#### Scenario: zoekt binary not found — auto-install

- **WHEN** `scripts/zoekt-search.sh "query"` is run and `zoekt` is not in the skill-local `bin/` directory or on PATH
- **THEN** the script runs `scripts/install-zoekt.sh` automatically, then retries the search operation

#### Scenario: Auto-install fails

- **WHEN** `scripts/zoekt-search.sh` triggers `scripts/install-zoekt.sh` and the install fails
- **THEN** the script exits with a non-zero status and prints the install error

### Requirement: Default Index Location

The default index location SHALL be `.git/zoekt` relative to the repository root. This location is inside `.git/` so `zoekt-index` automatically ignores it (avoiding recursive indexing) and it is not tracked by git.

- **Requirement ID**: zoekt-agent-skill:default-index-location

#### Scenario: Default index resolves to .git/zoekt

- **WHEN** `scripts/zoekt-index.sh` or `scripts/zoekt-search.sh` is run without `--index-dir`
- **THEN** the script uses `.git/zoekt` relative to the nearest git root as the index directory

#### Scenario: Not in a git repository

- **WHEN** a script is run outside a git repository without `--index-dir`
- **THEN** the script exits with a non-zero status and prints an error indicating it must be run inside a git repository or with an explicit `--index-dir`

### Requirement: Stable Agent Contract

The bundled scripts SHALL provide a stable contract that abstracts over zoekt flag naming inconsistencies. Agents SHALL only need to know `scripts/zoekt-index.sh` and `scripts/zoekt-search.sh <query>` without understanding zoekt's internal flag conventions.

- **Requirement ID**: zoekt-agent-skill:stable-agent-contract

#### Scenario: Flag names are consistent

- **WHEN** an agent uses `--index-dir` with either the index or search script
- **THEN** the flag is accepted by both scripts and maps to the correct underlying zoekt flag (`-index` for `zoekt-index`, `-index_dir` for `zoekt`)

### Requirement: Index Freshness Awareness

The skill SHALL inform agents that the index may be stale if files have been edited since the last indexing run, and SHALL provide guidance on when to re-index.

- **Requirement ID**: zoekt-agent-skill:index-freshness-awareness

#### Scenario: Agent is warned about potential staleness

- **WHEN** the skill is loaded and the agent is about to search
- **THEN** the skill notes that results reflect the last index time and suggests running `scripts/zoekt-index.sh` if recent edits may not be captured

### Requirement: Cross-Tool Portability

The skill SHALL be placed at `.agents/skills/zoekt-search/SKILL.md` so it is portable across Claude Code, OpenCode, and Codex without tool-specific configuration.

- **Requirement ID**: zoekt-agent-skill:cross-tool-portability

#### Scenario: Skill is loadable by OpenCode

- **WHEN** an OpenCode session starts in a repository containing `.agents/skills/zoekt-search/SKILL.md`
- **THEN** the skill appears in the available skills list and can be loaded on demand

#### Scenario: Skill is loadable by Claude Code

- **WHEN** a Claude Code session starts in a repository containing `.agents/skills/zoekt-search/SKILL.md`
- **THEN** the skill is available via the standard skill loading mechanism
<!-- ITO:END -->
