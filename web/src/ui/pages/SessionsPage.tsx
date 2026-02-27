import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api, isUnauthorizedError, WorkspaceSession } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'
import { Btn, Card, CardHeader, ErrorBanner, FormRow, Input, ScopeBadge, Table, Th, Td, EmptyRow } from '../components/primitives'

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
    { intervalMs: 2500, enabled: token.trim() !== '', onUnauthorized }
  )

  const sessions = useMemo(
    () => (q.data?.workspaceSessions ?? []).slice().sort((a, b) => b.id.localeCompare(a.id)),
    [q.data]
  )

  const create = useCallback(async () => {
    setCreating(true)
    setCreateErr(null)
    try {
      const sess = await api.createWorkspaceSession(token, repoURL)
      nav({ to: '/workspace-sessions/$workspaceSessionID', params: { workspaceSessionID: sess.id } })
    } catch (e) {
      if (isUnauthorizedError(e)) { onUnauthorized(); return }
      setCreateErr(e instanceof Error ? e.message : String(e))
    } finally {
      setCreating(false)
    }
  }, [token, repoURL, nav, onUnauthorized])

  const cellRepo = (s: WorkspaceSession) => (s.repoURL && s.repoURL.trim() !== '' ? s.repoURL : '\u2014')

  return (
    <>
      <Topbar title="Workspace Sessions" subtitle="Provision sessions, spawn harness runs, observe lifecycle state." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <Card>
          <CardHeader
            title="Provision Session"
            right={<Btn onClick={q.reload} type="button">Refresh</Btn>}
          />
          <FormRow label="Repo URL">
            <Input
              value={repoURL}
              onChange={(e) => setRepoURL(e.target.value)}
              placeholder="https://..."
            />
          </FormRow>
          <div className="flex items-center gap-2 mt-1 pl-27">
            <Btn variant="primary" disabled={creating || token.trim() === ''} onClick={create} type="button">
              {creating ? 'Provisioning\u2026' : 'Provision'}
            </Btn>
            <ScopeBadge scope="workspace-session:write" />
          </div>
          {createErr ? <ErrorBanner>{createErr}</ErrorBanner> : null}
          {q.error ? <ErrorBanner>{q.error}</ErrorBanner> : null}
          {token.trim() === '' ? <ErrorBanner>No bearer token set. Auth required for API calls.</ErrorBanner> : null}
        </Card>

        <Card>
          <CardHeader
            title="Workspace Sessions"
            right={<span className="text-[10px] text-muted-foreground/50 font-mono">polling 2.5s</span>}
          />
          <Table label="workspace sessions table">
            <thead>
              <tr className="border-b border-border/40">
                <Th>ID</Th>
                <Th>Repo</Th>
                <Th className="w-28">Phase</Th>
              </tr>
            </thead>
            <tbody>
              {sessions.length === 0 ? (
                <EmptyRow cols={3} loading={q.loading} message="No workspace sessions." />
              ) : (
                sessions.map((s) => (
                  <tr key={s.id} className="border-b border-border/20 last:border-b-0 hover:bg-muted/30 transition-colors">
                    <Td className="font-mono">
                      <Link to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: s.id }} className="text-primary hover:underline">{s.id}</Link>
                    </Td>
                    <Td className="font-mono text-muted-foreground truncate max-w-md" title={cellRepo(s)}>
                      {cellRepo(s)}
                    </Td>
                    <Td><StatusPill phase={s.phase} /></Td>
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
