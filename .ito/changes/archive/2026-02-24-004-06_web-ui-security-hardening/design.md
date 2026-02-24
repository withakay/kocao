## Context

The UI is an internal tool today, but we assume private network hostility and apply defense-in-depth.
This change focuses on token exposure reduction and safe defaults.

## Goals / Non-Goals

- Goals: reduce token persistence; remove secrets from URLs; improve auth failure UX.
- Non-Goals: introduce a full login system or external identity provider.

## Decisions

- Default to session-scoped token storage.
- Provide an explicit “remember token” option for persistence.
- Align attach flow with backend improvements so attach tokens are not placed in URLs.
