## Why
With CRD lifecycle in place, teams still need a secure API contract to create and manage sessions and runs. API, RBAC, and audit controls are required to make orchestrated execution safe and operable.

## What Changes
- Add a REST API surface for session and run lifecycle operations.
- Implement authentication and scope-based RBAC checks for all mutating operations.
- Record append-only audit events for run lifecycle, credential use, attach control changes, and egress overrides.

## Capabilities

### New Capabilities
- `control-plane-api`: authenticated and authorized API contract for session and run orchestration.

### Modified Capabilities
- None.

## Impact
- Affects API handlers, auth middleware, database schema/tables for permissions and audit records, and client contracts.
- Establishes security and traceability guarantees for all user-facing control-plane actions.
