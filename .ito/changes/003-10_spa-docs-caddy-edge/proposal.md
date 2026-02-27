## Why
The cluster deployment currently serves a placeholder HTML page from Caddy rather than the real React SPA, and API documentation endpoints are split between unversioned paths with no first-class docs portal.

To reduce operator overhead and keep deployment simple, we want one web edge image that serves the built SPA, rendered Markdown docs, and Scalar, while proxying API/WebSocket traffic to the local control-plane API container.

## What Changes
- Build a dedicated Caddy web image that bundles SPA assets, docs site assets, and Scalar entrypoint.
- Update control-plane deployment to use the bundled web image instead of ConfigMap-hosted static files.
- Serve docs at `/docs` and expose API docs at `/api/v1/scalar` and `/api/v1/openapi.json`.
- Keep compatibility redirects from legacy `/scalar` and `/openapi.json` endpoints.
- Add docs links in the SPA topbar and make local Vite API proxy target configurable for cluster development.

## Capabilities

### Modified Capabilities
- `workflow-ui-github`
- `web-ui`

## Impact
- Affects Docker image build workflow (`Makefile`, `build/` Dockerfiles)
- Affects control-plane deployment manifests and Caddy edge routing
- Adds Markdown docs publishing pipeline into web image build
- Updates operator-facing deployment and smoke documentation
