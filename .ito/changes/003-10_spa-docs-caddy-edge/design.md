## Context
The current in-cluster Caddy edge serves static files from ConfigMaps (`index.html`, `scalar.html`) and proxies API traffic. This works for basic smoke tests but does not ship the actual React SPA or project docs.

We need a low-maintenance deployment model where the same edge image serves:
- React SPA (`/`)
- Markdown docs portal (`/docs`)
- Scalar API docs (`/api/v1/scalar`)
- versioned OpenAPI endpoint (`/api/v1/openapi.json`)

while preserving API/auth behavior and attach websocket proxying.

## Goals / Non-Goals

**Goals**
- Deploy real SPA in cluster without introducing extra services.
- Keep one public web edge endpoint for UI + docs + API docs.
- Preserve API and websocket proxy behavior to the local API container.
- Keep developer workflow simple for local Vite-to-cluster development.

**Non-Goals**
- Replace HashRouter with BrowserRouter.
- Introduce a new docs backend service.
- Replace OpenAPI generation strategy in this change.

## Decisions

- Decision: Build a dedicated Caddy web image (`kocao/control-plane-web`) that bundles static assets.
  - Rationale: keeps runtime simple (single edge container) and avoids ConfigMap size/maintenance pain for SPA assets.
  - Alternatives considered:
    - Keep ConfigMap static assets: not viable for full SPA asset set and docs growth.
    - Separate web deployment/service: more moving parts than needed.

- Decision: Publish docs as pre-rendered HTML in image build from repository Markdown.
  - Rationale: deterministic output, no runtime dependency on external CDNs.
  - Alternatives considered:
    - Client-side markdown rendering via CDN: fragile in restricted networks.

- Decision: Version API docs paths under `/api/v1/*` with redirects from legacy paths.
  - Rationale: clearer contract and forward compatibility.

## Risks / Trade-offs

- [Web image build complexity increases] -> Keep build pipeline in one Dockerfile and minimal script.
- [Caddy route regressions affect API/attach] -> Preserve explicit route ordering and keep websocket matcher tests/smoke checks.
- [Docs drift from repo markdown] -> Generate docs directly from `docs/` at build time.

## Migration Plan
1. Add web image Dockerfile and docs render script.
2. Update Caddyfile routes for SPA/docs/Scalar/versioned OpenAPI.
3. Switch deployment from ConfigMap static serving to bundled web image.
4. Update docs and smoke checks.
5. Validate build, manifests, and web tests.

## Open Questions
- None.
