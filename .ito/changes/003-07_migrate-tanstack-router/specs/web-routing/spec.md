<!-- ITO:START -->
## ADDED Requirements

### Requirement: Web UI uses TanStack Router for client-side routing

The web UI SHALL use `@tanstack/react-router` as the sole client-side routing library.

Route definitions SHALL be type-safe, providing compile-time validation of route params and search params.

The router SHALL use hash-based history so the SPA works without server-side route configuration.

#### Scenario: Application routes via TanStack Router

- **WHEN** the web UI is loaded in a browser
- **THEN** all navigation is handled by `@tanstack/react-router` with hash-based history

#### Scenario: Route params are type-safe

- **WHEN** a page component accesses route params (e.g., `workspaceSessionID`)
- **THEN** the params are typed at compile time through the TanStack route definition

### Requirement: No react-router-dom dependency

The web UI SHALL NOT depend on `react-router-dom` or any `@remix-run/router` package.

#### Scenario: Dependency removed

- **WHEN** the web package dependencies are inspected
- **THEN** `react-router-dom` is not present in `dependencies` or `devDependencies`

### Requirement: Navigation components use TanStack Link

All internal navigation SHALL use TanStack Router's `Link` component with typed `to` props.

#### Scenario: Links use typed routes

- **WHEN** a component renders an internal navigation link
- **THEN** it uses `@tanstack/react-router`'s `Link` component with a route path validated at compile time
<!-- ITO:END -->
