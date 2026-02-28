# Tasks for: 003-07_migrate-tanstack-router

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 003-07_migrate-tanstack-router
ito tasks next 003-07_migrate-tanstack-router
ito tasks start 003-07_migrate-tanstack-router 1.1
ito tasks complete 003-07_migrate-tanstack-router 1.1
```

______________________________________________________________________

## Wave 1

- **Depends On**: None

### Task 1.1: Install TanStack Router and create route tree

- **Files**: `web/package.json`, `web/src/ui/routeTree.ts` (new), `web/src/ui/App.tsx`
- **Dependencies**: None
- **Action**: Install `@tanstack/react-router`. Create code-based route tree with `createRootRoute`, `createRoute`, and `createHashHistory`. Replace `HashRouter`/`Routes`/`Route` in `App.tsx` with `createRouter` and `RouterProvider`.
- **Verify**: `cd web && pnpm test`
- **Done When**: App renders via TanStack Router with hash history; existing routes still resolve.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending

### Task 1.2: Migrate page component hooks

- **Files**: `web/src/ui/pages/AttachPage.tsx`, `web/src/ui/pages/SessionDetailPage.tsx`, `web/src/ui/pages/SessionsPage.tsx`, `web/src/ui/pages/RunDetailPage.tsx`, `web/src/ui/pages/RunsPage.tsx`
- **Dependencies**: Task 1.1
- **Action**: Replace `useParams`, `useNavigate`, `useSearchParams`, `useLocation` from react-router-dom with TanStack Router equivalents. Update `Link` imports to TanStack's `Link`.
- **Verify**: `cd web && pnpm test`
- **Done When**: All page components import exclusively from `@tanstack/react-router`; no react-router-dom imports remain in page files.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending

### Task 1.3: Migrate Shell navigation components

- **Files**: `web/src/ui/components/Shell.tsx`
- **Dependencies**: Task 1.1
- **Action**: Replace `NavLink`, `Outlet`, `useLocation` with TanStack equivalents. Use `Link` with `activeProps` or `activeOptions` for active-link styling. Replace `Outlet` with TanStack's `Outlet`.
- **Verify**: `cd web && pnpm test`
- **Done When**: Shell uses only TanStack Router components; active nav styling preserved.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending

______________________________________________________________________

## Wave 2

- **Depends On**: Wave 1

### Task 2.1: Update test rendering context

- **Files**: `web/src/ui/workflow.test.tsx`, `web/src/test/setup.ts`
- **Dependencies**: None
- **Action**: Replace any react-router test wrappers with TanStack Router test utilities (`createMemoryHistory`, `RouterProvider`). Ensure all existing tests pass with new router context.
- **Verify**: `cd web && pnpm test`
- **Done When**: All tests pass using TanStack Router context; no react-router-dom imports in test files.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending

### Task 2.2: Remove react-router-dom dependency

- **Files**: `web/package.json`, `web/pnpm-lock.yaml`
- **Dependencies**: Task 2.1
- **Action**: Uninstall `react-router-dom`. Verify no remaining imports via grep. Run full test suite.
- **Verify**: `cd web && pnpm test && grep -r 'react-router-dom' src/ && echo 'FAIL: still present' || echo 'OK: removed'`
- **Done When**: `react-router-dom` is absent from package.json and no source file imports it.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending

______________________________________________________________________

## Wave 3

- **Depends On**: Wave 2

### Task 3.1: Final verification and cleanup

- **Files**: all web source files
- **Dependencies**: None
- **Action**: Run full test suites (`pnpm test`, `make test`). Verify no React Router v7 deprecation warnings in browser console. Confirm hash-based URLs still work.
- **Verify**: `cd web && pnpm test` and `make test`
- **Done When**: All tests pass, zero deprecation warnings, hash routing works identically to before.
- **Updated At**: 2026-02-24
- **Status**: [ ] pending
