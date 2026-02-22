# Kubernetes manifests

This directory contains kustomize manifests for the control plane.

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
