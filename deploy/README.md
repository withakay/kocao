# Kubernetes manifests

This directory contains kustomize manifests for the control plane.

The control-plane API pod runs two containers:

- `api` (`kocao/control-plane-api`)
- `caddy` web edge (`kocao/control-plane-web`)

The web edge serves:

- `/` (SPA)
- `/docs` (rendered markdown docs)
- `/api/v1/scalar` (Scalar API reference)
- `/api/v1/openapi.json` (live OpenAPI schema via API proxy)

Local dev (kind):

```bash
make kind-up
make images
make kind-load-images
make deploy
```

Delete:

```bash
make undeploy
```
