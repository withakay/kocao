package controlplaneapi

import "net/http"

// openAPISpec is a minimal OpenAPI document served for client generation.
// Keep this as a stable contract; expand as endpoints mature.
var openAPISpec = []byte(`{
  "openapi": "3.0.3",
  "info": {"title": "kocao control-plane api", "version": "v1"},
  "paths": {
    "/api/v1/workspace-sessions": {"get": {"security": [{"bearerAuth": []}]}, "post": {"security": [{"bearerAuth": []}] }},
    "/api/v1/workspace-sessions/{workspaceSessionID}": {"get": {"security": [{"bearerAuth": []}] }},
    "/api/v1/workspace-sessions/{workspaceSessionID}/harness-runs": {"post": {"security": [{"bearerAuth": []}] }},
    "/api/v1/harness-runs": {"get": {"security": [{"bearerAuth": []}] }},
    "/api/v1/harness-runs/{harnessRunID}": {"get": {"security": [{"bearerAuth": []}] }},
    "/api/v1/harness-runs/{harnessRunID}/stop": {"post": {"security": [{"bearerAuth": []}] }},
    "/api/v1/harness-runs/{harnessRunID}/resume": {"post": {"security": [{"bearerAuth": []}] }},
    "/api/v1/workspace-sessions/{workspaceSessionID}/attach-control": {"patch": {"security": [{"bearerAuth": []}] }},
    "/api/v1/workspace-sessions/{workspaceSessionID}/attach-token": {"post": {"security": [{"bearerAuth": []}] }},
    "/api/v1/workspace-sessions/{workspaceSessionID}/attach": {"get": {"security": [{"bearerAuth": []}] }},
    "/api/v1/workspace-sessions/{workspaceSessionID}/egress-override": {"patch": {"security": [{"bearerAuth": []}] }},
    "/api/v1/audit": {"get": {"security": [{"bearerAuth": []}] }},
    "/api/v1/cluster-overview": {"get": {"security": [{"bearerAuth": []}] }},
    "/api/v1/pods/{podName}/logs": {"get": {"security": [{"bearerAuth": []}] }}
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": {"type": "http", "scheme": "bearer"}
    }
  }
}`)

func openAPIHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPISpec)
}
