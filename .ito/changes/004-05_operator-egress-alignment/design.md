## Context

NetworkPolicy is IP/CIDR-based. The API currently exposes `allowedHosts`, but the operator cannot enforce
hostnames via NetworkPolicy. This mismatch is both a UX bug and a security risk.

## Goals / Non-Goals

- Goals: align API/operator semantics; improve validation; ensure behavior is deterministic and auditable.
- Non-Goals: implement hostname-based egress enforcement (would require an egress proxy / DNS policy / sidecar).

## Decisions

- Reject `allowedHosts` in the API until there is a real enforcement mechanism.
- Parse and validate `CP_GITHUB_EGRESS_CIDRS` using proper CIDR parsing.

## Risks / Trade-offs

- Some desired egress use cases (allow by hostname) are deferred.

## Open Questions

- Do we want an optional egress-proxy architecture for hostname allowlisting in the harness runtime?
