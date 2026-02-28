<!-- ITO:START -->
## 1. Implementation (retrospective)
- [x] 1.1 Add `createdAt` to workspace session API responses
- [x] 1.2 Add `DELETE /api/v1/workspace-sessions/{id}` endpoint to terminate sessions
- [x] 1.3 Add Settings page for token management; remove token entry from Topbar
- [x] 1.4 Improve sidebar nav (headers + icons) and add Settings navigation
- [x] 1.5 Improve Workspace Sessions page (started column, newest-first ordering, refresh placement)
- [x] 1.6 Add Kill action in list + detail views for Active sessions
- [x] 1.7 Update web tests for new auth UI and messaging

## 2. Verification
- [x] 2.1 `go test ./...`
- [x] 2.2 `pnpm -C web lint`
- [x] 2.3 `pnpm -C web test`
- [x] 2.4 `make kind-refresh` (rebuild + deploy for local dev)
<!-- ITO:END -->
