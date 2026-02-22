import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'

export function RunsPage() {
  const { token } = useAuth()
  const [filter, setFilter] = useState('')

  const q = usePollingQuery(
    `runs:${token}`,
    () => api.listRuns(token, undefined),
    { intervalMs: 2500, enabled: token.trim() !== '' }
  )

  const runs = useMemo(() => {
    const all = (q.data?.runs ?? []).slice().sort((a, b) => b.id.localeCompare(a.id))
    const f = filter.trim().toLowerCase()
    if (f === '') return all
    return all.filter((r) => r.id.toLowerCase().includes(f) || r.repoURL.toLowerCase().includes(f) || (r.sessionID ?? '').toLowerCase().includes(f))
  }, [q.data, filter])

  return (
    <>
      <Topbar title="Runs" subtitle="All HarnessRun resources; filter by session, repo, or id." />

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
            <input className="input" value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="run id, session id, repo" />
          </div>
          {q.error ? <div className="errorBox">{q.error}</div> : null}
        </section>

        <section className="card">
          <div className="cardHeader">
            <h2>Runs</h2>
            <div className="muted">Live</div>
          </div>
          <table className="table" aria-label="runs table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Session</th>
                <th>Repo</th>
                <th>Phase</th>
              </tr>
            </thead>
            <tbody>
              {runs.length === 0 ? (
                <tr>
                  <td colSpan={4} className="muted">
                    {q.loading ? 'Loading…' : 'No runs.'}
                  </td>
                </tr>
              ) : (
                runs.map((r) => (
                  <tr key={r.id}>
                    <td className="mono">
                      <Link to={`/runs/${encodeURIComponent(r.id)}`}>{r.id}</Link>
                    </td>
                    <td className="mono">
                      {r.sessionID ? <Link to={`/sessions/${encodeURIComponent(r.sessionID)}`}>{r.sessionID}</Link> : '(none)'}
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
