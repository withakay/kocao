import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api, isUnauthorizedError, WorkspaceSession } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'
import {
  Badge,
  Btn,
  Card,
  CardHeader,
  ErrorBanner,
  Input,
  ScopeBadge,
  Table,
  Th,
  Td,
  EmptyRow,
} from '../components/primitives'

function createdAtMillis(s: WorkspaceSession): number {
  const raw = (s.createdAt ?? '').trim()
  if (!raw) return 0
  const t = Date.parse(raw)
  return Number.isFinite(t) ? t : 0
}

function formatStarted(raw: string | undefined): { primary: string; secondary?: string } {
  const v = (raw ?? '').trim()
  if (!v) return { primary: '\u2014' }
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return { primary: v }
  return {
    primary: d.toLocaleDateString(),
    secondary: d.toLocaleTimeString(),
  }
}

export function SessionsPage() {
  const { token, invalidateToken } = useAuth()
  const nav = useNavigate()
  const [repoURL, setRepoURL] = useState('https://github.com/withakay/kocao')
  const [creating, setCreating] = useState(false)
  const [createErr, setCreateErr] = useState<string | null>(null)

  const [killingID, setKillingID] = useState<string | null>(null)
  const [killErr, setKillErr] = useState<string | null>(null)

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in Settings.')
  }, [invalidateToken])

  const q = usePollingQuery(
    `workspace-sessions:${token}`,
    () => api.listWorkspaceSessions(token),
    { intervalMs: 2500, enabled: token.trim() !== '', onUnauthorized }
  )

  const sessions = useMemo(() => {
    const all = (q.data?.workspaceSessions ?? []).slice()
    return all.sort((a, b) => {
      const dt = createdAtMillis(b) - createdAtMillis(a)
      if (dt !== 0) return dt
      return (b.displayName ?? b.id).localeCompare(a.displayName ?? a.id)
    })
  }, [q.data])

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

  const kill = useCallback(
    async (s: WorkspaceSession) => {
      if (token.trim() === '') return
      const name = s.displayName ?? s.id
      if (!window.confirm(`Kill workspace session "${name}"? This will terminate active runs and delete the session.`)) {
        return
      }
      setKillingID(s.id)
      setKillErr(null)
      try {
        await api.deleteWorkspaceSession(token, s.id)
        await q.reload()
      } catch (e) {
        if (isUnauthorizedError(e)) {
          onUnauthorized()
          return
        }
        setKillErr(e instanceof Error ? e.message : String(e))
      } finally {
        setKillingID(null)
      }
    },
    [token, q, onUnauthorized]
  )

  const cellRepo = (s: WorkspaceSession) => (s.repoURL && s.repoURL.trim() !== '' ? s.repoURL : '\u2014')

  return (
    <>
      <Topbar title="Workspace Sessions" subtitle="Provision sessions, spawn harness runs, observe lifecycle state." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <Card>
          <CardHeader title="Provision Session" right={<ScopeBadge scope="workspace-session:write" />} />

          <div className="flex items-end gap-2">
            <div className="flex-1 min-w-0">
              <div className="text-xs text-muted-foreground mb-1">Repo URL</div>
              <Input value={repoURL} onChange={(e) => setRepoURL(e.target.value)} placeholder="https://..." />
            </div>
            <Btn variant="primary" disabled={creating || token.trim() === '' || repoURL.trim() === ''} onClick={create} type="button">
              {creating ? 'Provisioning\u2026' : 'Provision'}
            </Btn>
          </div>

          <div className="mt-2 text-[11px] text-muted-foreground/70">
            Creates a durable workspace PVC and session anchor. You can start multiple runs in the session.
          </div>

          {createErr ? <ErrorBanner>{createErr}</ErrorBanner> : null}
          {token.trim() === '' ? (
            <ErrorBanner>
              No bearer token set. Open <Link className="underline" to="/settings">Settings</Link> to configure auth.
            </ErrorBanner>
          ) : null}
        </Card>

        <Card>
          <CardHeader
            title="Workspace Sessions"
            right={(
              <div className="flex items-center gap-2">
                <Btn onClick={q.reload} type="button">Refresh</Btn>
                <span className="text-[10px] text-muted-foreground/50 font-mono">polling 2.5s</span>
                <Badge variant="neutral">newest first</Badge>
              </div>
            )}
          />
          <Table label="workspace sessions table">
            <thead>
              <tr className="border-b border-border/40">
                <Th>Name</Th>
                <Th className="w-36">Started</Th>
                <Th>Repo</Th>
                <Th className="w-28">Phase</Th>
                <Th className="w-28">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {sessions.length === 0 ? (
                <EmptyRow cols={5} loading={q.loading} message="No workspace sessions." />
              ) : (
                sessions.map((s) => {
                  const started = formatStarted(s.createdAt)
                  const canKill = (s.phase ?? '').toLowerCase() === 'active'
                  return (
                    <tr key={s.id} className="border-b border-border/20 last:border-b-0 hover:bg-muted/30 transition-colors">
                      <Td className="font-mono">
                        <Link to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: s.id }} className="text-primary hover:underline">
                          {s.displayName ?? s.id}
                        </Link>
                        {s.displayName ? <div className="text-[10px] text-muted-foreground/70 mt-0.5">{s.id}</div> : null}
                      </Td>
                      <Td className="font-mono text-muted-foreground">
                        <div>{started.primary}</div>
                        {started.secondary ? <div className="text-[10px] text-muted-foreground/70">{started.secondary}</div> : null}
                      </Td>
                      <Td className="font-mono text-muted-foreground truncate max-w-md" title={cellRepo(s)}>
                        {cellRepo(s)}
                      </Td>
                      <Td><StatusPill phase={s.phase} /></Td>
                      <Td>
                        {canKill ? (
                          <Btn
                            variant="danger"
                            disabled={token.trim() === '' || killingID === s.id}
                            onClick={() => kill(s)}
                            type="button"
                          >
                            {killingID === s.id ? 'Killing\u2026' : 'Kill'}
                          </Btn>
                        ) : (
                          <span className="text-[11px] text-muted-foreground/60">\u2014</span>
                        )}
                      </Td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </Table>
          {q.error ? <ErrorBanner>{q.error}</ErrorBanner> : null}
          {killErr ? <ErrorBanner>{killErr}</ErrorBanner> : null}
        </Card>
      </div>
    </>
  )
}
