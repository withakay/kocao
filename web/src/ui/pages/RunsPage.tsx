import { useCallback, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'

export function RunsPage() {
  const { token, invalidateToken } = useAuth()
  const [filter, setFilter] = useState('')

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const q = usePollingQuery(
    `harness-runs:${token}`,
    () => api.listHarnessRuns(token, undefined),
    { intervalMs: 2500, enabled: token.trim() !== '', onUnauthorized }
  )

  const runs = useMemo(() => {
    const all = (q.data?.harnessRuns ?? []).slice().sort((a, b) => b.id.localeCompare(a.id))
    const f = filter.trim().toLowerCase()
    if (f === '') return all
    return all.filter((r) => r.id.toLowerCase().includes(f) || r.repoURL.toLowerCase().includes(f) || (r.workspaceSessionID ?? '').toLowerCase().includes(f))
  }, [q.data, filter])

  return (
    <>
      <Topbar title="Harness Runs" subtitle="All Harness Run resources; filter by workspace session, repo, or id." />

      <div className="grid">
        <section className="card">
          <div className="cardHeader">
            <h2>Filter</h2>
            <button className="btn" onClick={q.reload} type="button">
              Refresh
            </button>
          </div>
          <div className="formRow">
            <div className="label">Search</div>
            <input className="input" value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="harness run id, workspace session id, repo" />
          </div>
          {q.error ? <div className="errorBox">{q.error}</div> : null}
        </section>

        <section className="card">
          <div className="cardHeader">
            <h2>Harness Runs</h2>
            <div className="muted">Live</div>
          </div>
          <table className="table" aria-label="harness runs table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Workspace Session</th>
                <th>Repo</th>
                <th>Harness Run Lifecycle</th>
              </tr>
            </thead>
            <tbody>
              {runs.length === 0 ? (
                <tr>
                  <td colSpan={4} className="muted">
                    {q.loading ? 'Loading…' : 'No harness runs.'}
                  </td>
                </tr>
              ) : (
                runs.map((r) => (
                  <tr key={r.id}>
                    <td className="mono">
                      <Link to={`/harness-runs/${encodeURIComponent(r.id)}`}>{r.id}</Link>
                    </td>
                    <td className="mono">
                      {r.workspaceSessionID ? (
                        <Link to={`/workspace-sessions/${encodeURIComponent(r.workspaceSessionID)}`}>{r.workspaceSessionID}</Link>
                      ) : (
                        '(none)'
                      )}
                    </td>
                    <td className="mono">{r.repoURL}</td>
                    <td>
                      <StatusPill phase={r.phase} />
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
