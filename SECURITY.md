# Security

kocao is security-sensitive: it can create pods, control network egress, and provide interactive exec attach into
running workloads. Treat all deployments as operating in a hostile private network.

This repository uses a defense-in-depth approach. The baseline security contract is captured in:

- `docs/security/posture.md`
- `.ito/changes/004-01_baseline-security-review/specs/security-posture/spec.md` (normative requirements)

## Supported Deployment Modes

- Dev (kind): convenience defaults for local testing; not hardened for hostile networks.
- Production: deploy behind TLS termination (ingress/load balancer), restrict network reachability, and enforce
  least-privilege RBAC and deny-by-default egress.

## Reporting a Vulnerability

If you discover a security issue:

- Do not open a public issue with exploit details.
- Prefer reporting privately to the maintainers (out-of-band channel appropriate to your org).

If you must open an issue, keep it high-level and omit secrets, tokens, URLs containing credentials, and
reproduction steps that would enable exploitation.

## Operator Guidance (Short Version)

- Always terminate TLS before the control-plane API. Do not expose it as plain HTTP on a shared network.
- Treat bearer tokens and attach tokens as secrets.
- Prefer restricted egress by default; only allow additional destinations deliberately and audibly.
- Keep attach disabled unless explicitly required; enable it per-session and audit its use.
