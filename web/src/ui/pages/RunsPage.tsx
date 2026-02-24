import { useCallback, useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
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
      <Topbar title="Harness Runs" subtitle="Browse all harness runs. Filter by session, repo, or run id." />

      <div className="mt-4 flex flex-col gap-4">
        <section className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold tracking-tight">Filter</h2>
            <button
              className="rounded-md border border-border bg-secondary px-3 py-1.5 text-sm text-secondary-foreground hover:bg-secondary/80 transition-colors cursor-pointer"
              onClick={q.reload}
              type="button"
            >
              Refresh
            </button>
          </div>
          <div className="flex items-center gap-3">
            <div className="text-xs text-muted-foreground w-20 shrink-0">Search</div>
            <input
              className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/40 focus:border-ring"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="run id, session id, or repo url"
            />
          </div>
          {q.error ? <div className="mt-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground">{q.error}</div> : null}
        </section>

        <section className="rounded-lg border border-border bg-card p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold tracking-tight">Harness Runs</h2>
            <div className="text-xs text-muted-foreground font-mono">polling 2.5s</div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-sm" aria-label="harness runs table">
              <thead>
                <tr className="border-b border-border text-left">
                  <th className="py-2 pr-4 text-xs font-medium text-muted-foreground">ID</th>
                  <th className="py-2 pr-4 text-xs font-medium text-muted-foreground">Workspace Session</th>
                  <th className="py-2 pr-4 text-xs font-medium text-muted-foreground">Repo</th>
                   <th className="py-2 text-xs font-medium text-muted-foreground">Phase</th>
                </tr>
              </thead>
              <tbody>
                {runs.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="py-4 text-center text-muted-foreground">
                      {q.loading ? 'Loading…' : 'No harness runs.'}
                    </td>
                  </tr>
                ) : (
                  runs.map((r) => (
                    <tr key={r.id} className="border-b border-border/50 last:border-b-0">
                      <td className="py-2.5 pr-4 font-mono text-sm">
                        <Link to="/harness-runs/$harnessRunID" params={{ harnessRunID: r.id }} className="text-primary hover:underline">{r.id}</Link>
                      </td>
                      <td className="py-2.5 pr-4 font-mono text-sm">
                        {r.workspaceSessionID ? (
                          <Link to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: r.workspaceSessionID }} className="text-primary hover:underline">{r.workspaceSessionID}</Link>
                        ) : (
                          <span className="text-muted-foreground">(none)</span>
                        )}
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
        </section>
      </div>
    </>
  )
}
