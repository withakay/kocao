import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'

export function RunDetailPage() {
  const { runID } = useParams()
  const id = runID ?? ''
  const { token } = useAuth()
  const nav = useNavigate()
  const [actionErr, setActionErr] = useState<string | null>(null)
  const [acting, setActing] = useState(false)

  const runQ = usePollingQuery(
    `run:${id}:${token}`,
    () => api.getRun(token, id),
    { intervalMs: 2000, enabled: token.trim() !== '' && id !== '' }
  )
  const auditQ = usePollingQuery(
    `audit:${token}`,
    () => api.listAudit(token, 250),
    { intervalMs: 3000, enabled: token.trim() !== '' }
  )

  const run = runQ.data
  const events = useMemo(() => {
    const evs = auditQ.data?.events ?? []
    return evs.filter((e) => e.resourceID === id).slice(-40)
  }, [auditQ.data, id])

  const stop = useCallback(async () => {
    setActing(true)
    setActionErr(null)
    try {
      await api.stopRun(token, id)
      runQ.reload()
    } catch (e) {
      setActionErr(e instanceof Error ? e.message : String(e))
    } finally {
      setActing(false)
    }
  }, [token, id, runQ])

  const resume = useCallback(async () => {
    setActing(true)
    setActionErr(null)
    try {
      const out = await api.resumeRun(token, id)
      nav(`/runs/${encodeURIComponent(out.id)}`)
    } catch (e) {
      setActionErr(e instanceof Error ? e.message : String(e))
    } finally {
      setActing(false)
    }
  }, [token, id, nav])

  const attachLinks = run?.sessionID
    ? {
        viewer: `/sessions/${encodeURIComponent(run.sessionID)}/attach?role=viewer`,
        driver: `/sessions/${encodeURIComponent(run.sessionID)}/attach?role=driver`
      }
    : null

  return (
    <>
      <Topbar title={`Run ${id}`} subtitle="Lifecycle, attach entry points, and GitHub outcome metadata." />

      <div className="grid">
        <section className="card">
          <div className="cardHeader">
            <h2>Details</h2>
            <button className="btn" onClick={runQ.reload} type="button">
              Refresh
            </button>
          </div>

          {run ? (
            <>
              <div className="formRow">
                <div className="label">Session</div>
                <div className="mono">{run.sessionID ? <Link to={`/sessions/${encodeURIComponent(run.sessionID)}`}>{run.sessionID}</Link> : '(none)'}</div>
              </div>
              <div className="formRow">
                <div className="label">Repo</div>
                <div className="mono">{run.repoURL}</div>
              </div>
              <div className="formRow">
                <div className="label">Revision</div>
                <div className="mono">{run.repoRevision && run.repoRevision.trim() !== '' ? run.repoRevision : '(none)'}</div>
              </div>
              <div className="formRow">
                <div className="label">Image</div>
                <div className="mono">{run.image}</div>
              </div>
              <div className="formRow">
                <div className="label">Phase</div>
                <div>
                  <StatusPill phase={run.phase} />
                </div>
              </div>
              <div className="formRow">
                <div className="label">Pod</div>
                <div className="mono">{run.podName && run.podName.trim() !== '' ? run.podName : '(none yet)'}</div>
              </div>
            </>
          ) : (
            <div className="muted">{runQ.loading ? 'Loading…' : runQ.error ?? 'No data.'}</div>
          )}

          <div className="rowActions">
            <button className="btn btnDanger" disabled={acting || token.trim() === ''} onClick={stop} type="button">
              Stop
            </button>
            <button className="btn" disabled={acting || token.trim() === ''} onClick={resume} type="button">
              Resume
            </button>
            <span className="faint">Needs <span className="mono">run:write</span>.</span>
          </div>

          {actionErr ? <div className="errorBox">{actionErr}</div> : null}
          {runQ.error ? <div className="errorBox">{runQ.error}</div> : null}
        </section>

        <section className="card">
          <div className="cardHeader">
            <h2>Attach</h2>
            <div className="muted">Websocket console</div>
          </div>

          {attachLinks ? (
            <>
              <div className="muted">Attach tokens are short-lived; this page fetches a token and opens the websocket.</div>
              <div className="rowActions">
                <Link className="btn" to={attachLinks.viewer}>
                  Open Viewer
                </Link>
                <Link className="btn btnPrimary" to={attachLinks.driver}>
                  Open Driver
                </Link>
              </div>
            </>
          ) : (
            <div className="muted">This run is not associated with a session.</div>
          )}
        </section>
      </div>

      <div className="grid">
        <section className="card">
          <div className="cardHeader">
            <h2>GitHub Outcome</h2>
            <div className="muted">From run metadata</div>
          </div>
          {run ? (
            <>
              <div className="formRow">
                <div className="label">Branch</div>
                <div className="mono">{run.gitHubBranch && run.gitHubBranch.trim() !== '' ? run.gitHubBranch : '(none reported)'}</div>
              </div>
              <div className="formRow">
                <div className="label">PR URL</div>
                <div className="mono">
                  {run.pullRequestURL && run.pullRequestURL.trim() !== '' ? (
                    <a href={run.pullRequestURL} target="_blank" rel="noreferrer">
                      {run.pullRequestURL}
                    </a>
                  ) : (
                    '(none reported)'
                  )}
                </div>
              </div>
              <div className="formRow">
                <div className="label">PR Status</div>
                <div className="mono">{run.pullRequestStatus && run.pullRequestStatus.trim() !== '' ? run.pullRequestStatus : '(none reported)'}</div>
              </div>
            </>
          ) : (
            <div className="muted">No run loaded.</div>
          )}
        </section>

        <section className="card">
          <div className="cardHeader">
            <h2>Run Audit (Recent)</h2>
            <div className="muted">Derived from /api/v1/audit</div>
          </div>
          {events.length === 0 ? (
            <div className="muted">{auditQ.loading ? 'Loading…' : 'No recent audit events for this run.'}</div>
          ) : (
            <table className="table" aria-label="audit table">
              <thead>
                <tr>
                  <th>At</th>
                  <th>Action</th>
                  <th>Outcome</th>
                </tr>
              </thead>
              <tbody>
                {events.map((e) => (
                  <tr key={e.id}>
                    <td className="mono">{new Date(e.at).toLocaleTimeString()}</td>
                    <td className="mono">{e.action}</td>
                    <td className="mono">{e.outcome}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </div>
    </>
  )
}
