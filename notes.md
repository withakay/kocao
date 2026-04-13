---
## Cluster Usage Policy
## 2026-04-11 20:03:27 UTC

**`hz-arm` (context: `hz-arm`) is a PRODUCTION cluster — never use it for smoke testing, dev deployments, or acceptance testing.** It's a 4-node Hetzner ARM64 cluster (hz-mk8s-arm1 through hz-nbg-mk8s-arm4) connected via Tailscale. Use local kind clusters (`make kind-up`) or a dedicated dev cluster for testing instead.
