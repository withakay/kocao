## Why
Operators currently need `kubectl` to inspect pod health, deployment status, and recent logs for kocao components.

Adding an in-product cluster observability page reduces context switching and speeds up troubleshooting, especially for homelab and dev cluster workflows.

## What Changes
- Add API endpoints for namespace-scoped cluster overview and pod log tail retrieval.
- Add a new Cluster page in the web UI with pod/deployment status and on-demand pod logs.
- Add shell navigation and command-palette actions for the Cluster page.
- Add basic configuration visibility (non-secret runtime config indicators) in the dashboard.

## Capabilities

### Modified Capabilities
- `workflow-ui-github`
- `web-ui`
- `control-plane-api`

## Impact
- Backend: new cluster observability handlers and RBAC updates.
- Frontend: new route/page, API client methods, and UX for log inspection.
- Deployment: API service account gains read-only access for pods/deployments/configmaps in namespace.
