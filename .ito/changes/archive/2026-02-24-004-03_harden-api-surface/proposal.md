<!-- ITO:START -->
## Why

The control-plane API currently lacks several common hardening defaults (request size limits, timeouts beyond headers,
and safer “fail closed” patterns). Under a hostile private network assumption, these gaps increase the risk of abuse
and make accidental exposure easier as the API grows.

## What Changes

- Add request body limits and more defensive JSON decoding behavior.
- Add server-level timeouts and basic hardening headers.
- Reduce footguns by making authz patterns harder to accidentally bypass as new endpoints are added.

## Capabilities

### New Capabilities

- `control-plane-api`: security and robustness requirements for the HTTP API surface

### Modified Capabilities

<!-- none (no existing specs yet) -->

## Impact

- Affected code: `internal/controlplaneapi/*`, `cmd/control-plane-api/main.go`
- Affected behavior: request rejection for oversized bodies; more consistent HTTP behavior under load
<!-- ITO:END -->
