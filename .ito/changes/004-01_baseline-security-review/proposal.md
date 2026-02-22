<!-- ITO:START -->
## Why

kocao is security-sensitive (exec attach into pods, network egress control, Git credential handling) but
currently lacks a clear security contract and has several early-stage gaps. With a hostile private-network
assumption, we need defense-in-depth and a prioritized hardening plan.

## What Changes

- Establish a baseline security posture (threat model + non-goals + invariants) as requirements.
- Capture and triage findings into a prioritized, actionable remediation plan (feeds the follow-on changes in this module).
- Add minimal project guidance artifacts so future changes remain consistent (docs + Ito context).

## Capabilities

### New Capabilities

- `security-posture`: threat model, security invariants, and baseline hardening requirements

### Modified Capabilities

<!-- none (no existing specs yet) -->

## Impact

- Affected code (follow-on changes): `internal/controlplaneapi/**`, `internal/operator/**`, `deploy/**`, `web/**`
- Affected operational posture: authn/authz expectations, audit logging expectations, attach origin/message limits, and least-privilege RBAC
<!-- ITO:END -->
