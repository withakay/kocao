# Security Defense In Depth

## Purpose
Perform a deep code review and harden kocao against a hostile private network.

This module focuses on defense-in-depth: least privilege (RBAC + egress), safer defaults
in the control-plane API, hardened attach/websocket handling, and user-facing guidance.

## Scope
- security-posture
- audit-log
- control-plane-api
- attach-session
- k8s-rbac
- egress-policy
- web-ui
- harness-runtime

## Changes
- [x] 004-01_baseline-security-review
- [ ] 004-02_fix-audit-config-and-rbac
- [x] 004-03_harden-api-surface
- [x] 004-04_harden-attach-websocket
- [ ] 004-05_operator-egress-alignment
- [ ] 004-06_web-ui-security-hardening
- [ ] 004-07_harden-harness-runtime
