<!-- ITO:START -->
## Why

Egress controls are security-critical, but the current API/operator behavior is inconsistent.
The API supports an `allowedHosts` override annotation, while the operator only enforces mode + GitHub CIDRs.
This creates a functionality gap and risks a false sense of security.

## What Changes

- Align control-plane API egress override semantics with what the operator can actually enforce.
- Harden configuration validation for GitHub CIDR allowlisting.
- Improve documentation and observability so misconfiguration is obvious.

## Capabilities

### New Capabilities

- `egress-policy`: egress modes, configuration, and enforcement contract

### Modified Capabilities

<!-- none (no existing specs yet) -->

## Impact

- Affected code: `internal/controlplaneapi/api.go`, `internal/operator/controllers/egress_policy.go`
- Affected manifests/config: `deploy/overlays/dev-kind/config.env` (and future env configuration)
- User-facing impact: clearer errors when requesting unsupported egress overrides
<!-- ITO:END -->
