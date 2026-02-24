<!-- ITO:START -->
## Context

Today the UI can create runs, but it does not send execution parameters (`args`, etc). As a result, a “run” created from the UI typically starts a harness pod that clones the repo and then idles (sleep infinity) until a user attaches.
The control-plane API already supports execution fields in the run create request, and the operator/harness runtime already supports executing args after checkout.

## Goals / Non-Goals

**Goals:**

- Add a Task input to the UI to run a command non-interactively.
- Preserve harness entrypoint behavior (checkout + safety checks) by using container args and not overriding container command.
- Keep interactive attach workflows working (empty Task creates an idle, attachable pod).

**Non-Goals:**

- Implement log streaming in the UI.
- Add a full run template/preset system.
- Allow overriding the harness ENTRYPOINT via Kubernetes `command` from the UI.

## Decisions

- **Task mapping**: represent Task as container args `bash -lc <task>`.
  - Rationale: harness image includes bash; `-lc` gives consistent shell behavior; args preserve entrypoint checkout logic.
- **Advanced args**: expose an optional raw args editor for parity with API.
  - Rationale: enables power users without forcing everyone into low-level fields.
- **Empty Task**: omit args to keep the existing interactive/attach-first behavior.
  - Rationale: preserves current workflows and keeps “attach to a prepared repo checkout” as a first-class path.
- **Secrets warning**: show UI warning discouraging secrets in Task/args.
  - Rationale: Task/args are stored in Kubernetes objects and may be visible to cluster readers.

## Risks / Trade-offs

- [Users put secrets in Task] -> Mitigation: UI warning; document preferred secret handling (GitAuth/env from Secret) in separate work.
- [Task semantics differ by shell] -> Mitigation: default to `bash -lc`; advanced args allow explicit control.
- [Confusion around “command” vs “args”] -> Mitigation: UI does not expose container `command`; call the field Task and document behavior.

## Migration Plan

- Backward compatible UI change.
- Existing API callers remain supported.
- Rollback: redeploy previous web build; API/operator remain compatible.

## Open Questions

- Whether to expose additional execution inputs in Advanced (working dir, env, ttl) in the first iteration or follow-up.
<!-- ITO:END -->
