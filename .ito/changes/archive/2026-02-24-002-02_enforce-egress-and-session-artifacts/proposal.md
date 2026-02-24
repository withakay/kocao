## Why
MVP security and durability require strict default networking and session persistence outside ephemeral pods. Without these guarantees, the platform cannot meet baseline trust and resume expectations.

## What Changes
- Enforce default-deny egress with an allowlist path for GitHub and optional full-internet run override.
- Persist session artifacts to external storage on terminal run states.
- Restore persisted session state before resuming runs in new pods.

## Capabilities

### New Capabilities
- `session-durability`: egress controls plus artifact persist/restore behavior for resilient sessions.

### Modified Capabilities
- None.

## Impact
- Affects Kubernetes NetworkPolicies, run policy flags, storage abstraction and persistence flows.
- Impacts run startup/shutdown semantics and security posture controls.
