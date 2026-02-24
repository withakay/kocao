# Cluster UI + Scalar smoke (dev-kind)

1. Build and deploy:

```bash
make images
make kind-load-images
make deploy
```

2. Open service endpoint (default kind nodeport): `http://localhost:30080`.

3. Verify edge routes:

```bash
curl -i http://localhost:30080/healthz
curl -i http://localhost:30080/readyz
curl -i http://localhost:30080/openapi.json
curl -i http://localhost:30080/scalar
```

4. Verify API proxy behavior with an auth token:

```bash
curl -i -H "Authorization: Bearer <token>" http://localhost:30080/api/v1/workspace-sessions
```

5. Verify websocket attach path forwards through Caddy by exercising attach from the UI and confirming an active websocket upgrade on:

`/api/v1/workspace-sessions/{workspaceSessionID}/attach`
