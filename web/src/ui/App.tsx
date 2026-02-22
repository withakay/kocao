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
            <Route path="/" element={<Navigate to="/sessions" replace />} />
            <Route path="/sessions" element={<SessionsPage />} />
            <Route path="/sessions/:sessionID" element={<SessionDetailPage />} />
            <Route path="/sessions/:sessionID/attach" element={<AttachPage />} />
            <Route path="/runs" element={<RunsPage />} />
            <Route path="/runs/:runID" element={<RunDetailPage />} />
            <Route path="*" element={<Navigate to="/sessions" replace />} />
          </Route>
        </Routes>
      </HashRouter>
    </AuthProvider>
  )
}
