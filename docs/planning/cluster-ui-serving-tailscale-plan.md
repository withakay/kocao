# Optional Tailscale plan (disabled by default)

This plan keeps default `dev-kind` unchanged and introduces an opt-in overlay:

- Base overlay: `deploy/overlays/dev-kind`
- Optional overlay: `deploy/overlays/dev-kind-tailscale`
- Sidecar patch: `deploy/overlays/dev-kind/tailscale-patch.yaml`

## Enable flow

1. Create secret with auth key:

```bash
kubectl -n kocao-system create secret generic tailscale-auth \
  --from-literal=TS_AUTHKEY="<tskey-...>"
```

2. Render/apply optional overlay:

```bash
kustomize build deploy/overlays/dev-kind-tailscale | kubectl apply -f -
```

3. Verify sidecar joins tailnet and exposes control-plane edge according to your policy.

## Security posture

- Keep Tailscale overlay opt-in and environment-scoped.
- Rotate `tailscale-auth` credentials regularly.
- Restrict who can enable the optional overlay in CI/CD.
- Keep API bearer auth and RBAC checks unchanged behind the Tailscale transport.
