# kocao

Kubernetes-native coding agent orchestration platform.

## Local quickstart

```bash
make bootstrap
make test

make kind-up
make images
make kind-load-images
make deploy
```

## Commands

- `make bootstrap`: install local tools and download deps
- `make test`: run Go tests
- `make lint`: gofmt + go vet
- `make kind-up` / `make kind-down`: manage local kind cluster
- `make images`: build local Docker images
- `make kind-load-images`: load images into kind
- `make deploy` / `make undeploy`: apply kustomize overlay to cluster

## Layout

- `cmd/`: Go entrypoints
- `internal/`: shared Go packages
- `deploy/`: kustomize manifests
- `hack/`: scripts for local development
- `web/`: React UI (scaffold)
- `harness/`: harness assets (placeholder)
