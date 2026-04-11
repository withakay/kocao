## ADDED Requirements

### Requirement: GitHub Projects v2 Polling
The system SHALL poll a configured GitHub Projects v2 board and evaluate only issue-backed items against configured status mappings.

#### Scenario: Eligible issue-backed item is discovered
- **WHEN** the configured GitHub project contains an open issue-backed item whose status maps to an active Symphony state
- **THEN** the item is returned as a candidate for orchestration

### Requirement: Cross-Repository Issue Normalization
The GitHub source SHALL normalize linked GitHub issues into a stable issue model that includes repository identity, issue number, title, body, labels, URL, and created/updated timestamps.

#### Scenario: Linked issue metadata is normalized
- **WHEN** the source loads a project item linked to a GitHub issue
- **THEN** the orchestrator receives normalized issue data suitable for prompt rendering, sorting, and observability output

### Requirement: Unsupported Project Items Are Surfaced
The system SHALL skip and surface reasons for non-issue items, pull-request-backed items, archived items, and items from repositories outside the allowlist.

#### Scenario: Pull request item is skipped
- **WHEN** the GitHub project contains an item backed by a pull request instead of an issue
- **THEN** the item is not dispatched and the skip reason is visible to operators

### Requirement: Active Item State Refresh
The system SHALL refresh the current state of running items from GitHub during reconciliation.

#### Scenario: Running item becomes terminal in GitHub
- **WHEN** a previously claimed GitHub project item transitions to a configured terminal state
- **THEN** the orchestrator marks the item ineligible on the next reconciliation cycle and stops or releases active execution
