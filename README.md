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
- `make build-cli`: build the `kocao` CLI (`bin/kocao`)
- `make kind-up` / `make kind-down`: manage local kind cluster
- `make images`: build local Docker images
- `make kind-load-images`: load images into kind
- `make deploy` / `make undeploy`: apply kustomize overlay to cluster

## CLI

Build:

```bash
make build-cli
```

Configure:

```bash
export KOCAO_API_URL="http://127.0.0.1:8080"
export KOCAO_TOKEN="<bearer-token>"
```

Optional config file (`.json` only):

```json
{
  "api_url": "http://127.0.0.1:8080",
  "token": "<bearer-token>",
  "timeout": "15s",
  "verbose": false
}
```

Default file lookup order (later wins):

1. `~/.config/kocao/settings.json`
2. `settings.json` in the same directory as the `kocao` executable
3. `--config /path/to/settings.json` (if provided)
4. Environment variables (`KOCAO_*`)
5. Command-line flags

Examples:

```bash
./bin/kocao sessions ls
./bin/kocao sessions get <workspace-session-id>
./bin/kocao sessions status <workspace-session-id>
./bin/kocao sessions logs <workspace-session-id> --tail 200
./bin/kocao sessions logs <workspace-session-id> --follow
./bin/kocao sessions attach <workspace-session-id>
./bin/kocao sessions attach <workspace-session-id> --driver
```

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
