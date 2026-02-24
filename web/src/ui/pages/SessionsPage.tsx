import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth'
import { api, isUnauthorizedError, WorkspaceSession } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'

export function SessionsPage() {
  const { token, invalidateToken } = useAuth()
  const nav = useNavigate()
  const [repoURL, setRepoURL] = useState('https://github.com/withakay/kocao')
  const [creating, setCreating] = useState(false)
  const [createErr, setCreateErr] = useState<string | null>(null)

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const q = usePollingQuery(
    `workspace-sessions:${token}`,
    () => api.listWorkspaceSessions(token),
    {
      intervalMs: 2500,
      enabled: token.trim() !== '',
      onUnauthorized
    }
  )

  const sessions = useMemo(() => (q.data?.workspaceSessions ?? []).slice().sort((a, b) => b.id.localeCompare(a.id)), [q.data])

  const create = useCallback(async () => {
    setCreating(true)
    setCreateErr(null)
    try {
      const sess = await api.createWorkspaceSession(token, repoURL)
      nav(`/workspace-sessions/${encodeURIComponent(sess.id)}`)
    } catch (e) {
      if (isUnauthorizedError(e)) {
        onUnauthorized()
        return
      }
      setCreateErr(e instanceof Error ? e.message : String(e))
    } finally {
      setCreating(false)
    }
  }, [token, repoURL, nav, onUnauthorized])

  const cellRepo = (s: WorkspaceSession) => (s.repoURL && s.repoURL.trim() !== '' ? s.repoURL : '(none)')

  return (
    <>
      <Topbar title="Workspace Sessions" subtitle="Create workspace sessions, start harness runs, and monitor Workspace Session Lifecycle." />

      <div className="grid">
        <section className="card">
          <div className="cardHeader">
            <h2>Create Workspace Session</h2>
            <button className="btn" onClick={q.reload} type="button">
              Refresh
            </button>
          </div>

          <div className="formRow">
            <div className="label">Repo URL</div>
            <input className="input" value={repoURL} onChange={(e) => setRepoURL(e.target.value)} placeholder="https://..." />
          </div>

          <div className="rowActions">
            <button className="btn btnPrimary" disabled={creating || token.trim() === ''} onClick={create} type="button">
              {creating ? 'Creating…' : 'Create Workspace Session'}
            </button>
            <span className="faint">Requires token with <span className="mono">workspace-session:write</span>.</span>
          </div>

          {createErr ? <div className="errorBox">{createErr}</div> : null}
          {q.error ? <div className="errorBox">{q.error}</div> : null}
          {token.trim() === '' ? (
            <div className="errorBox">Set a bearer token in the top bar to use the API.</div>
          ) : null}
        </section>

        <section className="card">
          <div className="cardHeader">
            <h2>Workspace Sessions</h2>
            <div className="muted">Live</div>
          </div>

          <table className="table" aria-label="workspace sessions table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Repo</th>
                <th>Workspace Session Lifecycle</th>
              </tr>
            </thead>
            <tbody>
              {sessions.length === 0 ? (
                <tr>
                  <td colSpan={3} className="muted">
                    {q.loading ? 'Loading…' : 'No workspace sessions.'}
                  </td>
                </tr>
              ) : (
                sessions.map((s) => (
                  <tr key={s.id}>
                    <td className="mono">
                      <Link to={`/workspace-sessions/${encodeURIComponent(s.id)}`}>{s.id}</Link>
                    </td>
                    <td className="mono" title={cellRepo(s)}>
                      {cellRepo(s)}
                    </td>
                    <td>
                      <StatusPill phase={s.phase} />
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </section>
      </div>
    </>
  )
}
