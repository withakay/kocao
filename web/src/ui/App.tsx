import {
  createHashHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Navigate,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import type { RouterHistory } from '@tanstack/react-router'
import { AuthProvider } from './auth'
import { Shell } from './components/Shell'
import { SessionsPage } from './pages/SessionsPage'
import { SessionDetailPage } from './pages/SessionDetailPage'
import { AttachPage } from './pages/AttachPage'
import { RunsPage } from './pages/RunsPage'
import { RunDetailPage } from './pages/RunDetailPage'

// ── Route tree ──────────────────────────────────────────────

const rootRoute = createRootRoute({
  component: () => (
    <AuthProvider>
      <Outlet />
    </AuthProvider>
  ),
})

const shellRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'shell',
  component: Shell,
})

const indexRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/',
  component: () => <Navigate to="/workspace-sessions" />,
})

const sessionsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/workspace-sessions',
  component: SessionsPage,
})

const sessionDetailRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/workspace-sessions/$workspaceSessionID',
  component: SessionDetailPage,
})

const attachRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/workspace-sessions/$workspaceSessionID/attach',
  component: AttachPage,
  validateSearch: (search: Record<string, unknown>) => ({
    role: search['role'] === 'driver' ? 'driver' as const : 'viewer' as const,
  }),
})

const runsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/harness-runs',
  component: RunsPage,
})

const runDetailRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/harness-runs/$harnessRunID',
  component: RunDetailPage,
})

const routeTree = rootRoute.addChildren([
  shellRoute.addChildren([
    indexRoute,
    sessionsRoute,
    sessionDetailRoute,
    attachRoute,
    runsRoute,
    runDetailRoute,
  ]),
])

// ── Router factory ──────────────────────────────────────────

export function createAppRouter(history?: RouterHistory) {
  return createRouter({
    routeTree,
    history: history ?? createHashHistory(),
    defaultNotFoundComponent: () => <Navigate to="/workspace-sessions" />,
  })
}

export type AppRouter = ReturnType<typeof createAppRouter>

// ── Type registration ───────────────────────────────────────

declare module '@tanstack/react-router' {
  interface Register {
    router: AppRouter
  }
}

// ── App component ───────────────────────────────────────────

const defaultRouter = createAppRouter()

export function App() {
  return <RouterProvider router={defaultRouter} />
}
