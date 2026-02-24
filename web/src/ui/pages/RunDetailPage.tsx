import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'

export function RunDetailPage() {
  const { harnessRunID } = useParams({ strict: false })
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
      nav({ to: '/harness-runs/$harnessRunID', params: { harnessRunID: out.id } })
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

  const attachSessionID = run?.workspaceSessionID ?? null

  const cardClass = 'rounded-lg border border-border bg-card p-4'
  const headerClass = 'flex items-center justify-between mb-3'
  const rowClass = 'flex items-start gap-3 mb-3'
  const labelClass = 'text-xs text-muted-foreground w-28 shrink-0 pt-0.5'
  const refreshBtnClass =
    'rounded-md border border-border bg-secondary px-3 py-1.5 text-sm text-secondary-foreground hover:bg-secondary/80 transition-colors cursor-pointer'
  const errorClass = 'mt-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground'

  return (
    <>
      <Topbar title={`Harness Run ${id}`} subtitle="Run lifecycle, attach entry points, and GitHub outcome." />

      <div className="mt-4 flex flex-col gap-4">
        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">Details</h2>
            <button className={refreshBtnClass} onClick={runQ.reload} type="button">
              Refresh
            </button>
          </div>

          {run ? (
            <>
              <div className={rowClass}>
                <div className={labelClass}>Workspace Session</div>
                <div className="font-mono text-sm">
                  {run.workspaceSessionID ? (
                    <Link to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: run.workspaceSessionID }} className="text-primary hover:underline">{run.workspaceSessionID}</Link>
                  ) : (
                    <span className="text-muted-foreground">(none)</span>
                  )}
                </div>
              </div>
              <div className={rowClass}>
                <div className={labelClass}>Repo</div>
                <div className="font-mono text-sm">{run.repoURL}</div>
              </div>
              <div className={rowClass}>
                <div className={labelClass}>Revision</div>
                <div className="font-mono text-sm">{run.repoRevision && run.repoRevision.trim() !== '' ? run.repoRevision : '(none)'}</div>
              </div>
              <div className={rowClass}>
                <div className={labelClass}>Image</div>
                <div className="font-mono text-sm">{run.image}</div>
              </div>
              <div className={rowClass}>
                <div className={labelClass}>Phase</div>
                <div>
                  <StatusPill phase={run.phase} />
                </div>
              </div>
              <div className={rowClass}>
                <div className={labelClass}>Pod</div>
                <div className="font-mono text-sm">{run.podName && run.podName.trim() !== '' ? run.podName : '(none yet)'}</div>
              </div>
            </>
          ) : (
            <div className="text-sm text-muted-foreground">{runQ.loading ? 'Loading…' : runQ.error ?? 'No data.'}</div>
          )}

          <div className="flex items-center gap-3 mt-1">
            <button
              className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-1.5 text-sm text-foreground hover:bg-destructive/20 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
              disabled={acting || token.trim() === ''}
              onClick={stop}
              type="button"
            >
              Stop
            </button>
            <button
              className={refreshBtnClass + ' disabled:opacity-40 disabled:cursor-not-allowed'}
              disabled={acting || token.trim() === ''}
              onClick={resume}
              type="button"
            >
              Resume
            </button>
            <span className="text-xs text-muted-foreground">
              Scope: <code className="font-mono text-foreground/80">harness-run:write</code>
            </span>
          </div>

          {actionErr ? <div className={errorClass}>{actionErr}</div> : null}
          {runQ.error ? <div className={errorClass}>{runQ.error}</div> : null}
        </section>

        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">Attach</h2>
            <div className="text-xs text-muted-foreground font-mono">websocket attach</div>
          </div>

          {attachSessionID ? (
            <>
              <div className="text-sm text-muted-foreground mb-3">Attach tokens are ephemeral. This page acquires a token and opens the websocket.</div>
              <div className="flex items-center gap-3">
                <Link
                  className={refreshBtnClass}
                  to="/workspace-sessions/$workspaceSessionID/attach"
                  params={{ workspaceSessionID: attachSessionID }}
                  search={{ role: 'viewer' }}
                >
                  Open Viewer
                </Link>
                <Link
                  className="rounded-md border border-primary/30 bg-primary/10 px-3 py-1.5 text-sm text-foreground hover:bg-primary/20 transition-colors"
                  to="/workspace-sessions/$workspaceSessionID/attach"
                  params={{ workspaceSessionID: attachSessionID }}
                  search={{ role: 'driver' }}
                >
                  Open Driver
                </Link>
              </div>
            </>
          ) : (
            <div className="text-sm text-muted-foreground">This harness run is not associated with a workspace session.</div>
          )}
        </section>
      </div>

      <div className="mt-4 flex flex-col gap-4">
        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">GitHub Outcome</h2>
            <div className="text-xs text-muted-foreground font-mono">source: run metadata</div>
          </div>
          {run ? (
            <>
              <div className={rowClass}>
                <div className={labelClass}>Branch</div>
                <div className="font-mono text-sm">{run.gitHubBranch && run.gitHubBranch.trim() !== '' ? run.gitHubBranch : '(none reported)'}</div>
              </div>
              <div className={rowClass}>
                <div className={labelClass}>PR URL</div>
                <div className="font-mono text-sm">
                  {run.pullRequestURL && run.pullRequestURL.trim() !== '' ? (
                    <a href={run.pullRequestURL} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                      {run.pullRequestURL}
                    </a>
                  ) : (
                    '(none reported)'
                  )}
                </div>
              </div>
              <div className={rowClass}>
                <div className={labelClass}>PR Status</div>
                <div className="font-mono text-sm">{run.pullRequestStatus && run.pullRequestStatus.trim() !== '' ? run.pullRequestStatus : '(none reported)'}</div>
              </div>
            </>
          ) : (
            <div className="text-sm text-muted-foreground">No harness run loaded.</div>
          )}
        </section>

        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">Audit Trail</h2>
            <div className="text-xs text-muted-foreground font-mono">source: /api/v1/audit</div>
          </div>
          {events.length === 0 ? (
            <div className="text-sm text-muted-foreground">{auditQ.loading ? 'Loading…' : 'No audit events for this run.'}</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm" aria-label="audit table">
                <thead>
                  <tr className="border-b border-border text-left">
                    <th className="py-2 pr-4 text-xs font-medium text-muted-foreground">At</th>
                    <th className="py-2 pr-4 text-xs font-medium text-muted-foreground">Action</th>
                    <th className="py-2 text-xs font-medium text-muted-foreground">Outcome</th>
                  </tr>
                </thead>
                <tbody>
                  {events.map((e) => (
                    <tr key={e.id} className="border-b border-border/50 last:border-b-0">
                      <td className="py-2.5 pr-4 font-mono text-sm">{new Date(e.at).toLocaleTimeString()}</td>
                      <td className="py-2.5 pr-4 font-mono text-sm">{e.action}</td>
                      <td className="py-2.5 font-mono text-sm">{e.outcome}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </>
  )
}
