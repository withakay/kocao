<!-- ITO:START -->
## ADDED Requirements

### Requirement: Session API Contract Consistency

The control-plane session endpoints SHALL return a consistent public contract across create, status, stop, logs, and prompt responses.

- **Requirement ID**: control-plane-api:session-api-contract-consistency

#### Scenario: Session endpoints emit the same public identifiers

- **WHEN** a client calls create, status, or stop for the same harness run
- **THEN** each response uses the same public identifiers and field names (`runId`, `sessionId`, lifecycle phase, and related metadata)

### Requirement: Session API Safety On Failure

The control-plane session endpoints SHALL distinguish between recoverable soft failures and hard lifecycle failures.

- **Requirement ID**: control-plane-api:session-api-safety-on-failure

#### Scenario: Stop hits a stale proxy timeout

- **WHEN** the stop path encounters the known stale proxy timeout after a long-lived stream
- **THEN** the API may classify that specific case as a soft failure and complete the lifecycle with explicit persisted state

#### Scenario: Stop hits a hard remote failure

- **WHEN** the stop path encounters an unrelated transport or upstream failure
- **THEN** the API returns an error and persists failed lifecycle state instead of reporting false success
<!-- ITO:END -->
