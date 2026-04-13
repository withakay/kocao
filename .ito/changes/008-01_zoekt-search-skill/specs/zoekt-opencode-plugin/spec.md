<!-- ITO:START -->
## ADDED Requirements

### Requirement: Debounced Auto-Reindex on File Changes

The OpenCode plugin SHALL trigger `scripts/zoekt-index.sh` (from the zoekt-search skill) when files are edited, debouncing rapid successive edits to avoid excessive reindexing. The debounce interval SHALL be configurable with a sensible default (e.g., 30 seconds).

- **Requirement ID**: zoekt-opencode-plugin:debounced-auto-reindex

#### Scenario: File edit triggers debounced reindex

- **WHEN** a file is edited via OpenCode tools
- **THEN** the plugin schedules a reindex after the debounce interval, resetting the timer on each subsequent edit within the interval

#### Scenario: Rapid edits are coalesced

- **WHEN** multiple files are edited within the debounce window
- **THEN** only one reindex is triggered after the debounce interval expires

### Requirement: Session Idle Reindex

The plugin SHALL trigger a reindex on `session.idle` events as a conservative fallback, ensuring the index is current even if file-change hooks are missed.

- **Requirement ID**: zoekt-opencode-plugin:session-idle-reindex

#### Scenario: Session idle triggers reindex

- **WHEN** OpenCode emits a `session.idle` event
- **THEN** the plugin triggers `scripts/zoekt-index.sh` if the index is older than the most recently modified tracked file

#### Scenario: Index is already fresh

- **WHEN** OpenCode emits a `session.idle` event but no files have changed since the last index
- **THEN** the plugin skips reindexing

### Requirement: Non-Blocking Reindex

The plugin SHALL run `scripts/zoekt-index.sh` in the background without blocking the agent's current operation. Index failures SHALL be logged but SHALL NOT interrupt agent workflows.

- **Requirement ID**: zoekt-opencode-plugin:non-blocking-reindex

#### Scenario: Reindex runs in background

- **WHEN** a reindex is triggered
- **THEN** the `scripts/zoekt-index.sh` process runs asynchronously and does not block the current agent tool call or user interaction

#### Scenario: Reindex failure is logged

- **WHEN** `scripts/zoekt-index.sh` fails (e.g., binary not found, disk full)
- **THEN** the error is logged as a warning but the agent session continues without interruption

### Requirement: Plugin Location and Structure

The plugin SHALL be located at `.opencode/plugins/zoekt-reindex/` and SHALL follow the OpenCode plugin contract (ESM module with hook exports).

- **Requirement ID**: zoekt-opencode-plugin:plugin-location-and-structure

#### Scenario: Plugin is discovered by OpenCode

- **WHEN** OpenCode starts in a repository containing `.opencode/plugins/zoekt-reindex/`
- **THEN** the plugin is loaded and its hooks are registered for file change and session idle events
<!-- ITO:END -->
