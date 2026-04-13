<!-- ITO:START -->
## ADDED Requirements

### Requirement: Index Subcommand

The `agent-zoekt index` command SHALL invoke `zoekt-index` against the current working directory, storing the index at `.git/zoekt` by default. The command SHALL use filesystem-based indexing (not git-based) so that uncommitted edits are searchable.

- **Requirement ID**: zoekt-wrapper-cli:index-subcommand

#### Scenario: Index the current repository

- **WHEN** `agent-zoekt index` is run in a git repository
- **THEN** `zoekt-index -index .git/zoekt .` is executed and the index is written to `.git/zoekt/`

#### Scenario: Index with custom path

- **WHEN** `agent-zoekt index --index-dir /tmp/myindex` is run
- **THEN** `zoekt-index -index /tmp/myindex .` is executed and the index is written to the specified directory

#### Scenario: Index target directory

- **WHEN** `agent-zoekt index --dir /some/path` is run
- **THEN** `zoekt-index -index .git/zoekt /some/path` is executed, indexing the specified directory

#### Scenario: zoekt-index binary not found

- **WHEN** `agent-zoekt index` is run and `zoekt-index` is not on PATH
- **THEN** the command exits with a non-zero status and prints an error message indicating `zoekt-index` is not installed, with installation instructions

### Requirement: Search Subcommand

The `agent-zoekt search` command SHALL invoke `zoekt` against the index at `.git/zoekt` by default, passing the query and outputting results in JSONL format. The command SHALL hide the zoekt flag mismatch (`-index` for indexing vs `-index_dir` for searching).

- **Requirement ID**: zoekt-wrapper-cli:search-subcommand

#### Scenario: Search with default index

- **WHEN** `agent-zoekt search "func main"` is run in a git repository
- **THEN** `zoekt -index_dir .git/zoekt -jsonl "func main"` is executed and JSONL results are printed to stdout

#### Scenario: Search with custom index directory

- **WHEN** `agent-zoekt search --index-dir /tmp/myindex "func main"` is run
- **THEN** `zoekt -index_dir /tmp/myindex -jsonl "func main"` is executed

#### Scenario: Search with result limit

- **WHEN** `agent-zoekt search -n 20 "pattern"` is run
- **THEN** the search returns at most 20 results

#### Scenario: No index exists

- **WHEN** `agent-zoekt search "query"` is run but `.git/zoekt` does not exist
- **THEN** the command exits with a non-zero status and prints an error message suggesting the user run `agent-zoekt index` first

#### Scenario: zoekt binary not found

- **WHEN** `agent-zoekt search "query"` is run and `zoekt` is not on PATH
- **THEN** the command exits with a non-zero status and prints an error message indicating `zoekt` is not installed, with installation instructions

### Requirement: Stable Agent Contract

The `agent-zoekt` binary SHALL provide a stable CLI contract that abstracts over zoekt flag naming inconsistencies. Agents SHALL only need to know `agent-zoekt index` and `agent-zoekt search <query>` without understanding zoekt's internal flag conventions.

- **Requirement ID**: zoekt-wrapper-cli:stable-agent-contract

#### Scenario: Flag names are consistent

- **WHEN** an agent uses `--index-dir` with either `index` or `search` subcommands
- **THEN** the flag is accepted by both subcommands and maps to the correct underlying zoekt flag (`-index` for `zoekt-index`, `-index_dir` for `zoekt`)

### Requirement: Default Index Location

The default index location SHALL be `.git/zoekt` relative to the repository root. This location is inside `.git/` so `zoekt-index` automatically ignores it (avoiding recursive indexing) and it is not tracked by git.

- **Requirement ID**: zoekt-wrapper-cli:default-index-location

#### Scenario: Default index resolves to .git/zoekt

- **WHEN** `agent-zoekt index` or `agent-zoekt search` is run without `--index-dir`
- **THEN** the command uses `.git/zoekt` relative to the nearest git root as the index directory

#### Scenario: Not in a git repository

- **WHEN** `agent-zoekt` is run outside a git repository without `--index-dir`
- **THEN** the command exits with a non-zero status and prints an error indicating it must be run inside a git repository or with an explicit `--index-dir`
<!-- ITO:END -->
