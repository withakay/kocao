import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'

export function RunDetailPage() {
  const { harnessRunID } = useParams()
  const id = harnessRunID ?? ''
  const { token, invalidateToken } = useAuth()
  const nav = useNavigate()
  const [actionErr, setActionErr] = useState<string | null>(null)
  const [acting, setActing] = useState(false)

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const runQ = usePollingQuery(
    `harness-run:${id}:${token}`,
    () => api.getHarnessRun(token, id),
    { intervalMs: 2000, enabled: token.trim() !== '' && id !== '', onUnauthorized }
  )
  const auditQ = usePollingQuery(
    `audit:${token}`,
    () => api.listAudit(token, 250),
    { intervalMs: 3000, enabled: token.trim() !== '', onUnauthorized }
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
      await api.stopHarnessRun(token, id)
      runQ.reload()
    } catch (e) {
      if (isUnauthorizedError(e)) {
        onUnauthorized()
        return
      }
      setActionErr(e instanceof Error ? e.message : String(e))
    } finally {
      setActing(false)
    }
  }, [token, id, runQ, onUnauthorized])

  const resume = useCallback(async () => {
    setActing(true)
    setActionErr(null)
    try {
      const out = await api.resumeHarnessRun(token, id)
      nav(`/harness-runs/${encodeURIComponent(out.id)}`)
    } catch (e) {
      if (isUnauthorizedError(e)) {
        onUnauthorized()
        return
      }
      setActionErr(e instanceof Error ? e.message : String(e))
    } finally {
      setActing(false)
    }
  }, [token, id, nav, onUnauthorized])

  const attachLinks = run?.workspaceSessionID
    ? {
        viewer: `/workspace-sessions/${encodeURIComponent(run.workspaceSessionID)}/attach?role=viewer`,
        driver: `/workspace-sessions/${encodeURIComponent(run.workspaceSessionID)}/attach?role=driver`
      }
    : null

  return (
    <>
      <Topbar title={`Harness Run ${id}`} subtitle="Harness Run Lifecycle, attach entry points, and GitHub outcome metadata." />

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
                <div className="label">Workspace Session</div>
                <div className="mono">
                  {run.workspaceSessionID ? (
                    <Link to={`/workspace-sessions/${encodeURIComponent(run.workspaceSessionID)}`}>{run.workspaceSessionID}</Link>
                  ) : (
                    '(none)'
                  )}
                </div>
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
                <div className="label">Harness Run Lifecycle</div>
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
            <span className="faint">Needs <span className="mono">harness-run:write</span>.</span>
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
            <div className="muted">This harness run is not associated with a workspace session.</div>
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
            <div className="muted">No harness run loaded.</div>
          )}
        </section>

        <section className="card">
          <div className="cardHeader">
            <h2>Harness Run Audit (Recent)</h2>
            <div className="muted">Derived from /api/v1/audit</div>
          </div>
          {events.length === 0 ? (
            <div className="muted">{auditQ.loading ? 'Loading…' : 'No recent audit events for this harness run.'}</div>
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
