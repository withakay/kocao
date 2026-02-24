import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
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
      nav({ to: '/workspace-sessions/$workspaceSessionID', params: { workspaceSessionID: sess.id } })
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
      <Topbar title="Workspace Sessions" subtitle="Provision sessions, spawn harness runs, observe lifecycle state." />

      <div className="mt-4 flex flex-col gap-4">
        <section className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold tracking-tight">Provision Session</h2>
            <button
              className="rounded-md border border-border bg-secondary px-3 py-1.5 text-sm text-secondary-foreground hover:bg-secondary/80 transition-colors cursor-pointer"
              onClick={q.reload}
              type="button"
            >
              Refresh
            </button>
          </div>

          <div className="flex items-center gap-3 mb-3">
            <div className="text-xs text-muted-foreground w-20 shrink-0">Repo URL</div>
            <input
              className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/40 focus:border-ring"
              value={repoURL}
              onChange={(e) => setRepoURL(e.target.value)}
              placeholder="https://..."
            />
          </div>

          <div className="flex items-center gap-3">
            <button
              className="rounded-md border border-primary/30 bg-primary/10 px-4 py-2 text-sm text-foreground hover:bg-primary/20 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
              disabled={creating || token.trim() === ''}
              onClick={create}
              type="button"
            >
              {creating ? 'Provisioning…' : 'Provision'}
            </button>
            <span className="text-xs text-muted-foreground">
              Scope: <code className="font-mono text-foreground/80">workspace-session:write</code>
            </span>
          </div>

          {createErr ? <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground">{createErr}</div> : null}
          {q.error ? <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground">{q.error}</div> : null}
          {token.trim() === '' ? (
            <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground">No bearer token set. Auth required for API calls.</div>
          ) : null}
        </section>

        <section className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold tracking-tight">Workspace Sessions</h2>
            <div className="text-xs text-muted-foreground font-mono">polling 2.5s</div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-sm" aria-label="workspace sessions table">
              <thead>
                <tr className="border-b border-border text-left">
                  <th className="py-2 pr-4 text-xs font-medium text-muted-foreground">ID</th>
                  <th className="py-2 pr-4 text-xs font-medium text-muted-foreground">Repo</th>
                  <th className="py-2 text-xs font-medium text-muted-foreground">Phase</th>
                </tr>
              </thead>
              <tbody>
                {sessions.length === 0 ? (
                  <tr>
                    <td colSpan={3} className="py-4 text-center text-muted-foreground">
                      {q.loading ? 'Loading…' : 'No workspace sessions.'}
                    </td>
                  </tr>
                ) : (
                  sessions.map((s) => (
                    <tr key={s.id} className="border-b border-border/50 last:border-b-0">
                      <td className="py-2.5 pr-4 font-mono text-sm">
                        <Link to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: s.id }} className="text-primary hover:underline">{s.id}</Link>
                      </td>
                      <td className="py-2.5 pr-4 font-mono text-sm text-muted-foreground" title={cellRepo(s)}>
                        {cellRepo(s)}
                      </td>
                      <td className="py-2.5">
                        <StatusPill phase={s.phase} />
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </>
  )
}
