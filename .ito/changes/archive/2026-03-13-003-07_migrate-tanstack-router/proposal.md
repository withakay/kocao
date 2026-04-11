<!-- ITO:START -->
## Why

The web UI was specified to use TanStack Router but is currently built on react-router-dom v6, which emits deprecation warnings for React Router v7 and diverges from the project spec. Migrating now eliminates the warnings, aligns with the spec, and gains TanStack Router's type-safe route definitions and built-in search param validation before the routing surface grows further.

## What Changes

- Replace `react-router-dom` with `@tanstack/react-router` as the sole client-side routing library.
- Rewrite route definitions from JSX `<Route>` trees to TanStack's file-based or code-based route tree with typed params.
- Replace `HashRouter` with TanStack's hash-based history adapter.
- Migrate all page components from react-router hooks (`useParams`, `useNavigate`, `useSearchParams`, `useLocation`) to TanStack equivalents.
- Migrate navigation components (`Link`, `NavLink`) to TanStack's `Link` component with typed `to` props.
- Remove `react-router-dom` dependency entirely.
- Update all existing tests that depend on router rendering context.

## Capabilities

### New Capabilities

- `web-routing`: defines client-side routing technology, route structure, and navigation patterns for the web UI

### Modified Capabilities

<!-- none — web-ui spec covers auth UX only, not routing -->

## Impact

- Affected files (7 imports today): `App.tsx`, `Shell.tsx`, `AttachPage.tsx`, `SessionDetailPage.tsx`, `SessionsPage.tsx`, `RunDetailPage.tsx`, `RunsPage.tsx`
- Affected tests: `workflow.test.tsx` (renders `<App />` which uses the router)
- Affected dependencies: remove `react-router-dom`, add `@tanstack/react-router`
- No backend/API changes
<!-- ITO:END -->
