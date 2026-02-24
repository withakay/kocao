import { HashRouter, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider } from './auth'
import { Shell } from './components/Shell'
import { SessionsPage } from './pages/SessionsPage'
import { SessionDetailPage } from './pages/SessionDetailPage'
import { RunsPage } from './pages/RunsPage'
import { RunDetailPage } from './pages/RunDetailPage'
import { AttachPage } from './pages/AttachPage'

export function App() {
  return (
    <AuthProvider>
      <HashRouter>
        <Routes>
          <Route element={<Shell />}>
            <Route path="/" element={<Navigate to="/workspace-sessions" replace />} />
            <Route path="/workspace-sessions" element={<SessionsPage />} />
            <Route path="/workspace-sessions/:workspaceSessionID" element={<SessionDetailPage />} />
            <Route path="/workspace-sessions/:workspaceSessionID/attach" element={<AttachPage />} />
            <Route path="/harness-runs" element={<RunsPage />} />
            <Route path="/harness-runs/:harnessRunID" element={<RunDetailPage />} />
            <Route path="*" element={<Navigate to="/workspace-sessions" replace />} />
          </Route>
        </Routes>
      </HashRouter>
    </AuthProvider>
  )
}
