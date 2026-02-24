import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'

export function SessionDetailPage() {
  const { workspaceSessionID } = useParams({ strict: false })
  const id = workspaceSessionID ?? ''
  const { token, invalidateToken } = useAuth()
  const nav = useNavigate()

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const sessQ = usePollingQuery(
    `workspace-session:${id}:${token}`,
    () => api.getWorkspaceSession(token, id),
    { intervalMs: 2500, enabled: token.trim() !== '' && id !== '', onUnauthorized }
  )
  const runsQ = usePollingQuery(
    `harness-runs:${id}:${token}`,
    () => api.listHarnessRuns(token, id),
    { intervalMs: 2500, enabled: token.trim() !== '' && id !== '', onUnauthorized }
  )
  const auditQ = usePollingQuery(
    `audit:${token}`,
    () => api.listAudit(token, 200),
    { intervalMs: 3000, enabled: token.trim() !== '', onUnauthorized }
  )

  const runs = useMemo(() => (runsQ.data?.harnessRuns ?? []).slice().sort((a, b) => b.id.localeCompare(a.id)), [runsQ.data])
  const events = useMemo(() => {
    const evs = auditQ.data?.events ?? []
    return evs.filter((e) => e.resourceID === id).slice(-30)
  }, [auditQ.data, id])

  const [repoURL, setRepoURL] = useState('')
  const [repoRevision, setRepoRevision] = useState('')
  const [task, setTask] = useState('')
  const [advancedArgs, setAdvancedArgs] = useState('')
  const [image, setImage] = useState('kocao/harness-runtime:dev')
  const [egressMode, setEgressMode] = useState<'restricted' | 'full'>('restricted')
  const [starting, setStarting] = useState(false)
  const [startErr, setStartErr] = useState<string | null>(null)

  const start = useCallback(async () => {
    setStarting(true)
    setStartErr(null)
    try {
      const trimmedTask = task.trim()
      const trimmedAdvancedArgs = advancedArgs.trim()
      let args: string[] | undefined
      if (trimmedAdvancedArgs !== '') {
        let parsed: unknown
        try {
          parsed = JSON.parse(trimmedAdvancedArgs)
        } catch {
          throw new Error('Advanced args must be valid JSON (array of strings).')
        }
        if (!Array.isArray(parsed) || parsed.some((v) => typeof v !== 'string')) {
          throw new Error('Advanced args must be a JSON array of strings.')
        }
        args = parsed as string[]
      } else if (trimmedTask !== '') {
        args = ['bash', '-lc', trimmedTask]
      }

      const out = await api.startHarnessRun(token, id, {
        repoURL: repoURL.trim() !== '' ? repoURL.trim() : sessQ.data?.repoURL ?? '',
        repoRevision: repoRevision.trim() !== '' ? repoRevision.trim() : undefined,
        image: image.trim(),
        egressMode,
        args
      })
      nav({ to: '/harness-runs/$harnessRunID', params: { harnessRunID: out.id } })
    } catch (e) {
      if (isUnauthorizedError(e)) {
        onUnauthorized()
        return
      }
      setStartErr(e instanceof Error ? e.message : String(e))
    } finally {
      setStarting(false)
    }
  }, [token, id, repoURL, repoRevision, task, advancedArgs, image, egressMode, nav, sessQ.data, onUnauthorized])

  const sess = sessQ.data
  const effectiveRepo = repoURL.trim() !== '' ? repoURL.trim() : sess?.repoURL ?? ''

  const inputClass =
    'w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/40 focus:border-ring'
  const cardClass = 'rounded-lg border border-border bg-card p-4'
  const headerClass = 'flex items-center justify-between mb-3'
  const refreshBtnClass =
    'rounded-md border border-border bg-secondary px-3 py-1.5 text-sm text-secondary-foreground hover:bg-secondary/80 transition-colors cursor-pointer'
  const rowClass = 'flex items-start gap-3 mb-3'
  const labelClass = 'text-xs text-muted-foreground w-28 shrink-0 pt-2'

  return (
    <>
      <Topbar title={`Workspace Session ${id}`} subtitle="Session context, harness run dispatch, and audit trail." />

      <div className="mt-4 flex flex-col gap-4">
        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">Details</h2>
            <button className={refreshBtnClass} onClick={() => (sessQ.reload(), runsQ.reload())} type="button">
              Refresh
            </button>
          </div>
          {sess ? (
            <>
              <div className={rowClass}>
                <div className={labelClass}>Repo</div>
                <div className="font-mono text-sm">{sess.repoURL && sess.repoURL.trim() !== '' ? sess.repoURL : '(none)'}</div>
              </div>
              <div className={rowClass}>
                <div className={labelClass}>Phase</div>
                <div>
                  <StatusPill phase={sess.phase} />
                </div>
              </div>
            </>
          ) : (
            <div className="text-sm text-muted-foreground">{sessQ.loading ? 'Loading…' : sessQ.error ?? 'No data.'}</div>
          )}

          {sessQ.error ? <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground">{sessQ.error}</div> : null}
        </section>

        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">Dispatch Harness Run</h2>
            <div className="text-xs text-muted-foreground font-mono">scope: harness-run:write</div>
          </div>

          <div className={rowClass}>
            <div className={labelClass}>Repo URL</div>
            <input
              className={inputClass}
              value={effectiveRepo}
              onChange={(e) => setRepoURL(e.target.value)}
              placeholder="defaults to workspace session repoURL"
            />
          </div>
          <div className={rowClass}>
            <div className={labelClass}>Revision</div>
            <input className={inputClass} value={repoRevision} onChange={(e) => setRepoRevision(e.target.value)} placeholder="main (optional)" />
          </div>
          <div className={rowClass}>
            <div className={labelClass}>Task</div>
            <div className="flex-1">
              <textarea
                className={inputClass}
                rows={3}
                value={task}
                onChange={(e) => setTask(e.target.value)}
                placeholder="make ci"
              />
              <div className="mt-1 text-xs text-muted-foreground">
                Executed as <code className="font-mono text-foreground/80">bash -lc "&lt;task&gt;"</code>. Never embed secrets here.
              </div>
            </div>
          </div>
          <div className={rowClass}>
            <div className={labelClass}>Advanced args (JSON)</div>
            <div className="flex-1">
              <textarea
                className={inputClass}
                rows={3}
                value={advancedArgs}
                onChange={(e) => setAdvancedArgs(e.target.value)}
                placeholder='["go", "test", "./..."]'
              />
              <div className="mt-1 text-xs text-muted-foreground">Overrides task field when set.</div>
            </div>
          </div>
          <div className={rowClass}>
            <div className={labelClass}>Image</div>
            <input className={inputClass} value={image} onChange={(e) => setImage(e.target.value)} placeholder="kocao/harness-runtime:dev" />
          </div>
          <div className={rowClass}>
            <div className={labelClass}>Egress</div>
            <select
              className="rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring/40 focus:border-ring"
              value={egressMode}
              onChange={(e) => setEgressMode(e.target.value as 'restricted' | 'full')}
            >
              <option value="restricted">restricted (GitHub-only)</option>
              <option value="full">full (internet)</option>
            </select>
          </div>

          <div className="flex items-center gap-3 mt-1">
            <button
              className="rounded-md border border-primary/30 bg-primary/10 px-4 py-2 text-sm text-foreground hover:bg-primary/20 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
              disabled={starting || token.trim() === '' || effectiveRepo.trim() === ''}
              onClick={start}
              type="button"
            >
              {starting ? 'Starting…' : 'Start Harness Run'}
            </button>
            <span className="text-xs text-muted-foreground font-mono">
              scope: harness-run:write
            </span>
          </div>

          {startErr ? <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground">{startErr}</div> : null}
        </section>
      </div>

      <div className="mt-4 flex flex-col gap-4">
        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">Harness Runs</h2>
            <div className="text-xs text-muted-foreground font-mono">polling 2.5s</div>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm" aria-label="harness runs table">
              <thead>
                <tr className="border-b border-border text-left">
                  <th className="py-2 pr-4 text-xs font-medium text-muted-foreground">ID</th>
                  <th className="py-2 pr-4 text-xs font-medium text-muted-foreground">Repo</th>
                  <th className="py-2 text-xs font-medium text-muted-foreground">Phase</th>
                </tr>
              </thead>
              <tbody>
                {runs.length === 0 ? (
                  <tr>
                    <td colSpan={3} className="py-4 text-center text-muted-foreground">
                      {runsQ.loading ? 'Loading…' : 'No harness runs for this workspace session.'}
                    </td>
                  </tr>
                ) : (
                  runs.map((r) => (
                    <tr key={r.id} className="border-b border-border/50 last:border-b-0">
                      <td className="py-2.5 pr-4 font-mono text-sm">
                        <Link to="/harness-runs/$harnessRunID" params={{ harnessRunID: r.id }} className="text-primary hover:underline">{r.id}</Link>
                      </td>
                      <td className="py-2.5 pr-4 font-mono text-sm text-muted-foreground">{r.repoURL}</td>
                      <td className="py-2.5">
                        <StatusPill phase={r.phase} />
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
          {runsQ.error ? <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground">{runsQ.error}</div> : null}
        </section>

        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">Audit Trail</h2>
            <div className="text-xs text-muted-foreground font-mono">source: /api/v1/audit</div>
          </div>

          {events.length === 0 ? (
            <div className="text-sm text-muted-foreground">{auditQ.loading ? 'Loading…' : 'No audit events for this session.'}</div>
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
