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
make kind-prepull-harness-profiles
make deploy
```

Registry-backed dev clusters can warm the common harness profiles with:

```bash
HARNESS_IMAGE=ghcr.io/withakay/kocao/harness-runtime IMAGE_TAG=dev-microk8s-amd64fix \
  IMAGE_PULL_SECRETS=ghcr-pull \
  make registry-prepull-harness-profiles
```

That workflow creates a short-lived DaemonSet which pulls the configured `base`, `go`, `web`, and `full` profile tags onto each node, then removes the DaemonSet on exit unless `KEEP_PREPULL_DAEMONSET=1` is set.

Use `make microk8s-prepull-harness-profiles` as a convenience wrapper when the target cluster is reachable through the default `microk8s` kube context.

Delete:

```bash
make undeploy
```
