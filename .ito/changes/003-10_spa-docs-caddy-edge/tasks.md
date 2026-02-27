# Tasks for: 003-10_spa-docs-caddy-edge

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 003-10_spa-docs-caddy-edge
ito tasks next 003-10_spa-docs-caddy-edge
ito tasks start 003-10_spa-docs-caddy-edge 1.1
ito tasks complete 003-10_spa-docs-caddy-edge 1.1
```

______________________________________________________________________

## Wave 1 - Web Edge Image

- **Depends On**: None

### Task 1.1: Add bundled Caddy web image build

- **Files**: `build/Dockerfile.web`, `Makefile`
- **Dependencies**: None
- **Action**: Build SPA assets and docs into a dedicated Caddy image (`kocao/control-plane-web:dev`) and include it in image build/load flows.
- **Verify**: `make images`
- **Done When**: web image builds successfully and Makefile includes web image build/load targets.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 1.2: Add Markdown docs publishing script

- **Files**: `web/scripts/render-docs.mjs`
- **Dependencies**: None
- **Action**: Render markdown files from repository `docs/` into static HTML docs pages included in web image at `/docs`.
- **Verify**: `cd web && node ./scripts/render-docs.mjs --check`
- **Done When**: generated docs include index and page links with API docs links.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

______________________________________________________________________

## Wave 2 - Routing and Deployment

- **Depends On**: Wave 1

### Task 2.1: Update Caddy routes for SPA/docs/versioned API docs

- **Files**: `deploy/base/caddy/Caddyfile`, `deploy/base/caddy/scalar.html`
- **Dependencies**: None
- **Action**: Serve SPA from `/`, docs from `/docs`, Scalar from `/api/v1/scalar`, versioned OpenAPI from `/api/v1/openapi.json`, and add redirects from legacy `/scalar` and `/openapi.json`.
- **Verify**: `docker run --rm -v "$PWD/deploy/base/caddy/Caddyfile:/etc/caddy/Caddyfile" caddy:2 caddy validate --config /etc/caddy/Caddyfile`
- **Done When**: Caddyfile validates and route behavior matches contract.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 2.2: Switch deployment to bundled web image

- **Files**: `deploy/base/api-deployment.yaml`, `deploy/base/kustomization.yaml`
- **Dependencies**: None
- **Action**: Replace ConfigMap-mounted static serving with bundled web image container and remove obsolete static/config ConfigMap mounts.
- **Verify**: `kustomize build deploy/base`
- **Done When**: manifests render with web image and no caddy static/config volumes.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 2.3: Add docs/API links in SPA topbar and configurable Vite proxy target

- **Files**: `web/src/ui/components/Topbar.tsx`, `web/vite.config.ts`
- **Dependencies**: None
- **Action**: Add links to `/docs` and `/api/v1/scalar` in topbar and make Vite API proxy target configurable via env with sensible default.
- **Verify**: `cd web && pnpm tsc --noEmit`
- **Done When**: links are visible and local dev can point API proxy to different clusters.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

______________________________________________________________________

## Wave 3 - Verification and Documentation

- **Depends On**: Wave 1, Wave 2

### Task 3.1: Update deployment docs for GitOps and smoke checks

- **Files**: `README.md`, `deploy/README.md`, `docs/planning/cluster-ui-serving-dev-kind-smoke.md`
- **Dependencies**: None
- **Action**: Document web image, versioned docs endpoints, and smoke verification for SPA/docs/Scalar/OpenAPI.
- **Verify**: Manual review of commands and paths.
- **Done When**: docs describe cluster deployment behavior without implying placeholder UI.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 3.2: Full verification

- **Files**: all modified files
- **Dependencies**: Task 3.1
- **Action**: Run `pnpm -C web tsc --noEmit`, `pnpm -C web test`, `pnpm -C web build`, `kustomize build deploy/base`, and `ito validate 003-10_spa-docs-caddy-edge --strict`.
- **Verify**: listed commands
- **Done When**: all commands succeed.
- **Updated At**: 2026-02-27
- **Status**: [x] complete
