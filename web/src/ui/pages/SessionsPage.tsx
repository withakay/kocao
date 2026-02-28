import { useCallback, useEffect, useMemo, useState } from 'react'
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

function formatAge(nowMs: number, startedMs: number): string {
  if (!Number.isFinite(startedMs) || startedMs <= 0) return '\u2014'
  const deltaSec = Math.max(0, Math.floor((nowMs - startedMs) / 1000))
  if (deltaSec < 10) return 'just now'

  const mins = Math.floor(deltaSec / 60)
  if (mins < 60) return `${mins}m ago`

  const hrs = Math.floor(mins / 60)
  if (hrs < 48) return `${hrs}h ago`

  const days = Math.floor(hrs / 24)
  return `${days}d ago`
}

function formatStarted(raw: string | undefined): { age: string; exact: string; title: string } {
  const v = (raw ?? '').trim()
  if (!v) return { age: '\u2014', exact: '\u2014', title: '' }
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return { age: '\u2014', exact: v, title: v }

  const now = Date.now()
  const age = formatAge(now, d.getTime())
  const exact = d.toLocaleString()
  const title = d.toISOString()

  return { age, exact, title }
}

export function SessionsPage() {
  const { token, invalidateToken } = useAuth()
  const nav = useNavigate()
  const [repoURL, setRepoURL] = useState('https://github.com/withakay/kocao')
  const [creating, setCreating] = useState(false)
  const [createErr, setCreateErr] = useState<string | null>(null)

  const [terminatingID, setTerminatingID] = useState<string | null>(null)
  const [terminateErr, setTerminateErr] = useState<string | null>(null)

  const [menuID, setMenuID] = useState<string | null>(null)

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

  const terminate = useCallback(
    async (s: WorkspaceSession) => {
      if (token.trim() === '') return
      const name = s.displayName ?? s.id
      if (!window.confirm(`Terminate workspace session "${name}"? This will terminate active runs and delete the session.`)) {
        return
      }
      setTerminatingID(s.id)
      setTerminateErr(null)
      try {
        await api.deleteWorkspaceSession(token, s.id)
        await q.reload()
      } catch (e) {
        if (isUnauthorizedError(e)) {
          onUnauthorized()
          return
        }
        setTerminateErr(e instanceof Error ? e.message : String(e))
      } finally {
        setTerminatingID(null)
      }
    },
    [token, q, onUnauthorized]
  )

  useEffect(() => {
    if (!menuID) return

    const onPointerDown = (e: PointerEvent) => {
      const el = e.target as HTMLElement | null
      if (!el) return
      if (el.closest('[data-session-menu]')) return
      setMenuID(null)
    }

    document.addEventListener('pointerdown', onPointerDown)
    return () => document.removeEventListener('pointerdown', onPointerDown)
  }, [menuID])

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
                <Th className="w-44">Started</Th>
                <Th>Repo</Th>
                <Th className="w-28">Phase</Th>
                <Th className="w-16">Actions</Th>
              </tr>
            </thead>
            <tbody>
              {sessions.length === 0 ? (
                <EmptyRow cols={5} loading={q.loading} message="No workspace sessions." />
              ) : (
                sessions.map((s) => {
                  const started = formatStarted(s.createdAt)
                  const canTerminate = (s.phase ?? '').toLowerCase() === 'active'
                  const open = menuID === s.id

                  return (
                    <tr key={s.id} className="border-b border-border/20 last:border-b-0 hover:bg-muted/30 transition-colors">
                      <Td className="font-mono">
                        <Link to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: s.id }} className="text-primary hover:underline">
                          {s.displayName ?? s.id}
                        </Link>
                        {s.displayName ? <div className="text-[10px] text-muted-foreground/70 mt-0.5">{s.id}</div> : null}
                      </Td>
                      <Td className="font-mono text-muted-foreground" title={started.title}>
                        <div>{started.age}</div>
                        <div className="text-[10px] text-muted-foreground/70">{started.exact}</div>
                      </Td>
                      <Td className="font-mono text-muted-foreground truncate max-w-md" title={cellRepo(s)}>
                        {cellRepo(s)}
                      </Td>
                      <Td><StatusPill phase={s.phase} /></Td>
                      <Td>
                        <div className="relative inline-flex" data-session-menu>
                          <Btn
                            variant="ghost"
                            className="px-2"
                            type="button"
                            aria-label="Session actions"
                            onClick={() => setMenuID((cur) => (cur === s.id ? null : s.id))}
                          >
                            <DotsIcon />
                          </Btn>

                          {open ? (
                            <div className="absolute right-0 top-[calc(100%+6px)] z-20 min-w-40 rounded-md border border-border/70 bg-card shadow-lg shadow-black/30 overflow-hidden">
                              <button
                                type="button"
                                className="w-full text-left px-3 py-2 text-xs text-foreground hover:bg-secondary/60 disabled:opacity-40 disabled:cursor-not-allowed"
                                disabled={!canTerminate || terminatingID === s.id || token.trim() === ''}
                                onClick={async () => {
                                  setMenuID(null)
                                  if (!canTerminate) return
                                  await terminate(s)
                                }}
                              >
                                <span className="text-destructive">Terminate session</span>
                                {terminatingID === s.id ? <span className="ml-2 text-[10px] text-muted-foreground">working…</span> : null}
                              </button>
                              <div className="px-3 py-2 text-[10px] text-muted-foreground/70 border-t border-border/40">
                                {canTerminate ? 'Deletes the session and terminates runs.' : 'Only Active sessions can be terminated.'}
                              </div>
                            </div>
                          ) : null}
                        </div>
                      </Td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </Table>
          {q.error ? <ErrorBanner>{q.error}</ErrorBanner> : null}
          {terminateErr ? <ErrorBanner>{terminateErr}</ErrorBanner> : null}
        </Card>
      </div>
    </>
  )
}

function DotsIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M12 12.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z" fill="currentColor" />
      <path d="M19 12.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z" fill="currentColor" />
      <path d="M5 12.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z" fill="currentColor" />
    </svg>
  )
}
