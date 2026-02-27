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

## Cluster web edge

The control-plane deployment includes a Caddy web edge container (`kocao/control-plane-web`) that serves:

- SPA: `/`
- Docs portal: `/docs`
- Scalar API reference: `/api/v1/scalar`
- OpenAPI JSON: `/api/v1/openapi.json`

Legacy `/scalar` and `/openapi.json` are redirected to versioned paths.

## Control-plane configuration

Environment variables:

- `CP_ENV`: `dev|test|prod` (default: `dev`)
- `CP_HTTP_ADDR`: listen address (default: `:8080`)
- `POD_NAMESPACE` (recommended) or `CP_NAMESPACE`: namespace when running in-cluster
- `CP_BOOTSTRAP_TOKEN`: optional bring-up token (wildcard scopes; do not use long-term)
- `CP_AUDIT_PATH`: audit log file path (default: `kocao.audit.jsonl`)

Deprecated:

- `CP_DB_PATH`: deprecated alias for `CP_AUDIT_PATH` (will be removed)

## Layout

- `cmd/`: Go entrypoints
- `internal/`: shared Go packages
- `deploy/`: kustomize manifests
- `hack/`: scripts for local development
- `web/`: React UI (scaffold)
- `harness/`: harness assets (placeholder)
