<!-- ITO:START -->
## Context

The web UI currently uses `react-router-dom` v6 with `HashRouter` and JSX `<Route>` trees. This emits React Router v7 future-flag deprecation warnings and diverges from the project spec which calls for TanStack Router. The routing surface is small (7 files, ~6 routes) making migration straightforward now before complexity grows.

## Goals / Non-Goals

**Goals:**

- Replace react-router-dom with @tanstack/react-router as the sole routing library.
- Preserve hash-based history (no server routing config needed).
- Gain type-safe route params and search params.
- Eliminate React Router v7 deprecation warnings.
- Keep all existing routes and navigation behavior identical.

**Non-Goals:**

- Change route paths or URL structure.
- Add new routes or pages.
- Introduce file-based routing code generation (code-based route tree is sufficient for current scale).
- Migrate to browser-history mode (requires server config).

## Decisions

- **Use code-based route tree, not file-based.**
  - Decision: Define routes in a single `routeTree.ts` file using `createRoute` / `createRootRoute`.
  - Rationale: The app has ~6 routes. File-based routing adds a build plugin and directory conventions that aren't justified at this scale. Code-based keeps it explicit and zero-config.
  - Alternative considered: TanStack file-based routing with `@tanstack/router-plugin`; rejected as over-engineering for <10 routes.

- **Use `createHashHistory` for hash-based routing.**
  - Decision: Configure TanStack Router with `createHashHistory()` to preserve `/#/...` URL patterns.
  - Rationale: Maintains current URL structure and deployment simplicity (static file hosting, no server rewrite rules).

- **Migrate hooks 1:1 where possible.**
  - Decision: Map react-router hooks to TanStack equivalents: `useParams` → `useParams` (from route), `useNavigate` → `useNavigate`, `useSearchParams` → `useSearch`, `useLocation` → `useLocation`, `Link`/`NavLink` → `Link` with `activeProps`.
  - Rationale: Minimizes behavioral changes and keeps the diff focused on the routing library swap.

- **Update test rendering context.**
  - Decision: Replace `<MemoryRouter>` / `<HashRouter>` wrapping in tests with TanStack's `RouterProvider` or `createMemoryHistory` test utilities.
  - Rationale: Tests must render within the new router context to work.

## Risks / Trade-offs

- [TanStack Router API differences may surface edge cases] → Mitigation: small route surface; manual verification of each route after migration.
- [Test utilities differ between libraries] → Mitigation: update test setup once; all tests use the same pattern.
- [Hash history URL format may differ subtly] → Mitigation: verify URL patterns match existing `/#/` prefix behavior.

## Migration Plan

1. Install `@tanstack/react-router`.
2. Create code-based route tree with `createHashHistory`.
3. Migrate `App.tsx` from `HashRouter`/`Routes` to `RouterProvider`.
4. Migrate each page component's hooks one at a time.
5. Migrate `Shell.tsx` navigation components.
6. Update test rendering context.
7. Remove `react-router-dom` from dependencies.
8. Verify all tests pass and no deprecation warnings remain.

## Open Questions

- Should we adopt TanStack Router's loader pattern for data fetching in a follow-up change?
- Should `NavLink` active styling use `activeProps` or `activeOptions` with className callback?
<!-- ITO:END -->
