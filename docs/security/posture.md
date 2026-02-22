# Security Posture

This document describes the baseline security contract for kocao deployments.

Normative requirements live in `.ito/changes/004-01_baseline-security-review/specs/security-posture/spec.md`.
This doc translates those requirements into operator-facing guidance and deployment expectations.

## Threat Assumptions

- Assume a hostile private network.
- Assume clients may be untrusted or compromised.
- Assume workloads executed by the harness may be attacker-controlled (untrusted repo content).

## Baseline Invariants

1) Authenticated control-plane API

- All `/api/v1/*` endpoints MUST require authentication (bearer token), except explicit health endpoints and
  the OpenAPI document.
- Authorization is scope-based (e.g., `session:read`, `run:write`, `control:write`) and MUST be enforced on every
  mutating operation.

2) Attach safety

- Attach is remote shell access into the harness pod; treat it as privileged.
- Attach endpoints MUST be hardened: strict origin checks, message size limits, and conservative timeouts.
- Attach tokens MUST be treated as secrets and MUST NOT be placed in URLs.

3) Auditability

- Security-relevant actions MUST produce append-only audit events:
  - authorization allow/deny
  - session/run control changes
  - attach token issuance and attach usage
  - egress mode overrides

4) Secrets handling

- Do not place secrets in URLs or log lines.
- Prefer Kubernetes Secrets mounted/injected into runtime components.
- Use bootstrap tokens ONLY for initial bring-up; disable or tightly constrain them in production.

5) Egress control

- Restricted egress is the default posture.
- When full egress is allowed, it MUST be explicit and auditable.

## Deployment Guidance

### TLS and Network Exposure

- Terminate TLS before the control-plane API (ingress controller / service mesh / external LB).
- Do not expose plain HTTP on shared networks.
- Restrict inbound access to known operator/bastion origins.

### Tokens

- Bearer tokens are the primary API auth mechanism.
- Store tokens in Kubernetes Secrets (not ConfigMaps) and rotate regularly.
- `CP_BOOTSTRAP_TOKEN` is intended for bring-up only; treat it as a break-glass secret.

### RBAC (Least Privilege)

- The operator and API run with in-cluster credentials.
- RBAC MUST be least-privilege and limited to the namespace where kocao is installed.
- Any requirement for cluster-scoped permissions MUST be documented with justification.

### Egress Policy Configuration

- In restricted mode, runs are expected to have deny-by-default egress with explicit allow rules.
- GitHub allowlists are configured via `CP_GITHUB_EGRESS_CIDRS` (comma-separated CIDRs) for HTTPS/SSH.

### Attach Controls

- Attach SHOULD be disabled by default and enabled per-session as needed.
- Driver role MUST be tightly controlled (requires `control:write`) and its use SHOULD be audited.

## Current Gaps (Tracked Work)

Baseline gaps and the remediation plan are documented in `docs/security/review-2026-02-22.md` and tracked as
follow-on changes in module 004.
