import { useCallback, useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'
import { Btn, Card, CardHeader, ErrorBanner, FormRow, Input, Table, Th, Td, EmptyRow } from '../components/primitives'

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
    return all.filter((r) =>
      r.id.toLowerCase().includes(f) ||
      r.repoURL.toLowerCase().includes(f) ||
      (r.workspaceSessionID ?? '').toLowerCase().includes(f)
    )
  }, [q.data, filter])

  return (
    <>
      <Topbar title="Harness Runs" subtitle="Browse all harness runs. Filter by session, repo, or run id." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <Card>
          <CardHeader
            title="Filter"
            right={<Btn onClick={q.reload} type="button">Refresh</Btn>}
          />
          <FormRow label="Search">
            <Input
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="run id, session id, or repo url"
            />
          </FormRow>
          {q.error ? <ErrorBanner>{q.error}</ErrorBanner> : null}
        </Card>

        <Card>
          <CardHeader
            title="Harness Runs"
            right={<span className="text-[10px] text-muted-foreground/50 font-mono">polling 2.5s</span>}
          />
          <Table label="harness runs table">
            <thead>
              <tr className="border-b border-border/40">
                <Th>ID</Th>
                <Th>Workspace Session</Th>
                <Th>Repo</Th>
                <Th className="w-28">Phase</Th>
              </tr>
            </thead>
            <tbody>
              {runs.length === 0 ? (
                <EmptyRow cols={4} loading={q.loading} message="No harness runs." />
              ) : (
                runs.map((r) => (
                  <tr key={r.id} className="border-b border-border/20 last:border-b-0 hover:bg-muted/30 transition-colors">
                    <Td className="font-mono">
                      <Link to="/harness-runs/$harnessRunID" params={{ harnessRunID: r.id }} className="text-primary hover:underline">{r.id}</Link>
                    </Td>
                    <Td className="font-mono">
                      {r.workspaceSessionID ? (
                        <Link to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: r.workspaceSessionID }} className="text-primary hover:underline">{r.workspaceSessionID}</Link>
                      ) : (
                        <span className="text-muted-foreground">\u2014</span>
                      )}
                    </Td>
                    <Td className="font-mono text-muted-foreground">{r.repoURL}</Td>
                    <Td><StatusPill phase={r.phase} /></Td>
                  </tr>
                ))
              )}
            </tbody>
          </Table>
        </Card>
      </div>
    </>
  )
}
