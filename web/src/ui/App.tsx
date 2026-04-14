import { useMemo } from 'react'
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
import { RunsPage } from './pages/RunsPage'
import { RunDetailPage } from './pages/RunDetailPage'
import { AttachPage } from './pages/AttachPage'
import { ClusterPage } from './pages/ClusterPage'
import { SettingsPage } from './pages/SettingsPage'
import { SymphonyPage } from './pages/SymphonyPage'
import { SymphonyDetailPage } from './pages/SymphonyDetailPage'
import { RemoteAgentsPage } from './pages/RemoteAgentsPage'
import { RemoteAgentListPage } from './pages/RemoteAgentListPage'
import { RemoteAgentTaskDetailPage } from './pages/RemoteAgentTaskDetailPage'
import { RemoteAgentTaskTranscriptPage } from './pages/RemoteAgentTaskTranscriptPage'
import { RemoteAgentTaskArtifactsPage } from './pages/RemoteAgentTaskArtifactsPage'
import { RemoteAgentDetailPage } from './pages/RemoteAgentDetailPage'
import { remoteAgentTaskListSearchDefaults } from './lib/remoteAgentDashboard'

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
  component: () => <Navigate to="/workspace-sessions" replace />, 
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
  validateSearch: (search: Record<string, unknown>) => ({
    role: search.role === 'driver' ? ('driver' as const) : ('viewer' as const),
  }),
  component: AttachPage,
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

const clusterRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/cluster',
  component: ClusterPage,
})

const remoteAgentsIndexRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/remote-agents',
  component: () => <Navigate to="/remote-agents/tasks" search={remoteAgentTaskListSearchDefaults} replace />,
})

const remoteAgentTasksRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/remote-agents/tasks',
  validateSearch: (search: Record<string, unknown>) => ({
    q: typeof search.q === 'string' ? search.q : remoteAgentTaskListSearchDefaults.q,
    pool: typeof search.pool === 'string' ? search.pool : remoteAgentTaskListSearchDefaults.pool,
    state: search.state === 'active' || search.state === 'terminal' ? search.state : remoteAgentTaskListSearchDefaults.state,
    artifacts: search.artifacts === 'with-output' ? 'with-output' : remoteAgentTaskListSearchDefaults.artifacts,
  }),
  component: RemoteAgentsPage,
})

const remoteAgentTaskDetailRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/remote-agents/tasks/$taskId',
  component: RemoteAgentTaskDetailPage,
})

const remoteAgentTaskTranscriptRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/remote-agents/tasks/$taskId/transcript',
  component: RemoteAgentTaskTranscriptPage,
})

const remoteAgentTaskArtifactsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/remote-agents/tasks/$taskId/artifacts',
  component: RemoteAgentTaskArtifactsPage,
})

const remoteAgentAgentsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/remote-agents/agents',
  component: RemoteAgentListPage,
})

const remoteAgentDetailRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/remote-agents/agents/$agentId',
  component: RemoteAgentDetailPage,
})

const symphonyRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/symphony',
  component: SymphonyPage,
})

const symphonyDetailRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/symphony/$projectName',
  component: SymphonyDetailPage,
})

const settingsRoute = createRoute({
  getParentRoute: () => shellRoute,
  path: '/settings',
  component: SettingsPage,
})

const routeTree = rootRoute.addChildren([
  shellRoute.addChildren([
    indexRoute,
    sessionsRoute,
    sessionDetailRoute,
    attachRoute,
    runsRoute,
    runDetailRoute,
    clusterRoute,
    remoteAgentsIndexRoute,
    remoteAgentTasksRoute,
    remoteAgentTaskDetailRoute,
    remoteAgentTaskTranscriptRoute,
    remoteAgentTaskArtifactsRoute,
    remoteAgentAgentsRoute,
    remoteAgentDetailRoute,
    symphonyRoute,
    symphonyDetailRoute,
    settingsRoute,
  ]),
])

export function createAppRouter(history?: RouterHistory) {
  return createRouter({
    routeTree,
    history: history ?? createHashHistory(),
    defaultNotFoundComponent: () => <Navigate to="/workspace-sessions" replace />,
  })
}

export type AppRouter = ReturnType<typeof createAppRouter>

declare module '@tanstack/react-router' {
  interface Register {
    router: AppRouter
  }
}

export function App() {
  const router = useMemo(() => createAppRouter(), [])
  return <RouterProvider router={router} />
}
