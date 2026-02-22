## Context

Current wiring uses `CP_DB_PATH` as the audit path (API calls `newAuditStore(auditPath)` with `cfg.DBPath`).
This breaks the principle of least surprise and makes it hard to evolve storage.

Attach functionality requires `pods/exec`, which is a sensitive permission; it should be granted only to the API,
and only in the namespace where harness pods run.

## Goals / Non-Goals

- Goals: correct configuration naming/wiring; make attach work in-cluster; narrow RBAC via service account split.
- Non-Goals: introduce a real database or multi-namespace isolation (can be a follow-on if needed).

## Decisions

- Introduce `CP_AUDIT_PATH` and rename internal fields to reflect the real usage.
- Split service accounts: `control-plane-api` and `control-plane-operator` with distinct Roles/RoleBindings.

## Risks / Trade-offs

- Granting `pods/exec` is high-privilege within a namespace. Splitting service accounts reduces blast radius, but namespace isolation may still be warranted later.

## Migration Plan

- Add `CP_AUDIT_PATH`.
- Keep `CP_DB_PATH` as a deprecated alias (optional), with clear precedence rules documented.

## Open Questions

- Do we want a dedicated namespace for harness pods to further reduce `pods/exec` blast radius?
