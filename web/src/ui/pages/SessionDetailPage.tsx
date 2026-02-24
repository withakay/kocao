import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'

export function SessionDetailPage() {
  const { workspaceSessionID } = useParams()
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
      nav(`/harness-runs/${encodeURIComponent(out.id)}`)
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

  return (
    <>
      <Topbar title={`Workspace Session ${id}`} subtitle="Durable workspace context with Harness Run creation." />

      <div className="grid">
        <section className="card">
          <div className="cardHeader">
            <h2>Details</h2>
            <button className="btn" onClick={() => (sessQ.reload(), runsQ.reload())} type="button">
              Refresh
            </button>
          </div>
          {sess ? (
            <>
              <div className="formRow">
                <div className="label">Repo</div>
                <div className="mono">{sess.repoURL && sess.repoURL.trim() !== '' ? sess.repoURL : '(none)'}</div>
              </div>
              <div className="formRow">
                <div className="label">Workspace Session Lifecycle</div>
                <div>
                  <StatusPill phase={sess.phase} />
                </div>
              </div>
            </>
          ) : (
            <div className="muted">{sessQ.loading ? 'Loading…' : sessQ.error ?? 'No data.'}</div>
          )}

          {sessQ.error ? <div className="errorBox">{sessQ.error}</div> : null}
        </section>

        <section className="card">
          <div className="cardHeader">
            <h2>Start Harness Run</h2>
            <div className="muted">Creates a Harness Run</div>
          </div>

          <div className="formRow">
            <div className="label">Repo URL</div>
            <input
              className="input"
              value={effectiveRepo}
              onChange={(e) => setRepoURL(e.target.value)}
              placeholder="defaults to workspace session repoURL"
            />
          </div>
          <div className="formRow">
            <div className="label">Revision</div>
            <input className="input" value={repoRevision} onChange={(e) => setRepoRevision(e.target.value)} placeholder="main (optional)" />
          </div>
          <div className="formRow">
            <div className="label">Task</div>
            <textarea
              className="input"
              rows={3}
              value={task}
              onChange={(e) => setTask(e.target.value)}
              placeholder="make ci"
            />
            <div className="faint">Runs as <span className="mono">bash -lc "&lt;task&gt;"</span>. Do not put secrets in Task text.</div>
          </div>
          <div className="formRow">
            <div className="label">Advanced args (JSON)</div>
            <textarea
              className="input"
              rows={3}
              value={advancedArgs}
              onChange={(e) => setAdvancedArgs(e.target.value)}
              placeholder='["go", "test", "./..."]'
            />
            <div className="faint">Overrides Task when provided.</div>
          </div>
          <div className="formRow">
            <div className="label">Image</div>
            <input className="input" value={image} onChange={(e) => setImage(e.target.value)} placeholder="kocao/harness-runtime:dev" />
          </div>
          <div className="formRow">
            <div className="label">Egress</div>
            <select className="select" value={egressMode} onChange={(e) => setEgressMode(e.target.value as any)}>
              <option value="restricted">restricted (GitHub-only)</option>
              <option value="full">full (internet)</option>
            </select>
          </div>

          <div className="rowActions">
            <button className="btn btnPrimary" disabled={starting || token.trim() === '' || effectiveRepo.trim() === ''} onClick={start} type="button">
              {starting ? 'Starting…' : 'Start Harness Run'}
            </button>
            <span className="faint">Needs <span className="mono">harness-run:write</span>.</span>
          </div>

          {startErr ? <div className="errorBox">{startErr}</div> : null}
        </section>
      </div>

      <div className="grid">
        <section className="card">
          <div className="cardHeader">
            <h2>Harness Runs</h2>
            <div className="muted">Live</div>
          </div>
          <table className="table" aria-label="harness runs table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Repo</th>
                <th>Harness Run Lifecycle</th>
              </tr>
            </thead>
            <tbody>
              {runs.length === 0 ? (
                <tr>
                  <td colSpan={3} className="muted">
                    {runsQ.loading ? 'Loading…' : 'No harness runs for this workspace session.'}
                  </td>
                </tr>
              ) : (
                runs.map((r) => (
                  <tr key={r.id}>
                    <td className="mono">
                      <Link to={`/harness-runs/${encodeURIComponent(r.id)}`}>{r.id}</Link>
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
          {runsQ.error ? <div className="errorBox">{runsQ.error}</div> : null}
        </section>

        <section className="card">
          <div className="cardHeader">
            <h2>Workspace Session Audit (Recent)</h2>
            <div className="muted">Derived from /api/v1/audit</div>
          </div>

          {events.length === 0 ? (
            <div className="muted">{auditQ.loading ? 'Loading…' : 'No recent audit events for this workspace session.'}</div>
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
