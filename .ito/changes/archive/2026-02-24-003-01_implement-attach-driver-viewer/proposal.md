## Why
Running pods are not usable without an attach path that supports collaboration and safe control handoff. The platform needs explicit driver and viewer semantics with reconnect support.

## What Changes
- Add backend attach streaming with one interactive driver and multiple read-only viewers.
- Implement lease-based driver transfer and short-lived attach credentials.
- Support reconnect behavior without forcing pod restart when clients disconnect.

## Capabilities

### New Capabilities
- `attach-session`: multi-client attach protocol with driver/viewer control and reconnect semantics.

### Modified Capabilities
- None.

## Impact
- Affects websocket APIs, attach auth/token issuance, terminal stream multiplexing, and run UX behavior.
- Introduces coordination logic for control transfer and session continuity.
