## Context

kocao runs privileged workflows (orchestration + pod exec attach + git auth). Even when the product is “internal”,
we assume a hostile private network and implement defense-in-depth.

## Goals / Non-Goals

- Goals: document baseline security contract; make remediation work trackable; prioritize the highest-risk issues.
- Non-Goals: implement the remediation in this change (handled by follow-on changes in this module).

## Decisions

- Threat model: internal/single-tenant, but treat private network as hostile.
- Process: changes are split into small, verifiable follow-ons (RBAC/audit correctness, API hardening, attach hardening, operator egress alignment, UI hardening).

## Risks / Trade-offs

- Some requirements in `security-posture` describe desired behavior that may be ahead of current implementation; follow-on changes must close these gaps.

## Open Questions

- Which ingress / TLS termination model will be used for production (in-cluster Service + ingress controller vs external LB)?
