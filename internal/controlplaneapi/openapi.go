package controlplaneapi

import "net/http"

// openAPISpec is a minimal OpenAPI document served for client generation.
// Keep this as a stable contract; expand as endpoints mature.
var openAPISpec = []byte(`{
  "openapi": "3.0.3",
  "info": {"title": "kocao control-plane api", "version": "v1"},
  "paths": {
    "/api/v1/sessions": {"get": {"security": [{"bearerAuth": []}]}, "post": {"security": [{"bearerAuth": []}] }},
    "/api/v1/sessions/{sessionID}": {"get": {"security": [{"bearerAuth": []}] }},
    "/api/v1/sessions/{sessionID}/runs": {"post": {"security": [{"bearerAuth": []}] }},
    "/api/v1/runs": {"get": {"security": [{"bearerAuth": []}] }},
    "/api/v1/runs/{runID}": {"get": {"security": [{"bearerAuth": []}] }},
    "/api/v1/runs/{runID}/stop": {"post": {"security": [{"bearerAuth": []}] }},
    "/api/v1/runs/{runID}/resume": {"post": {"security": [{"bearerAuth": []}] }},
    "/api/v1/sessions/{sessionID}/attach-control": {"patch": {"security": [{"bearerAuth": []}] }},
    "/api/v1/sessions/{sessionID}/egress-override": {"patch": {"security": [{"bearerAuth": []}] }},
    "/api/v1/audit": {"get": {"security": [{"bearerAuth": []}] }}
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": {"type": "http", "scheme": "bearer"}
    }
  }
}`)

func openAPIHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPISpec)
}
