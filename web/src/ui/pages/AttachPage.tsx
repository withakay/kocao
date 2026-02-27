import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useParams, useSearch } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { base64DecodeToBytes, base64EncodeBytes } from '../lib/base64'
import { useAttachLayout } from '../lib/useLayoutState'
import { Topbar } from '../components/Topbar'
import { GhosttyTerminal, type TerminalHandle } from '../components/GhosttyTerminal'
import { ResizablePanel } from '../components/ResizablePanel'
import { InspectorPanel } from '../components/InspectorPanel'
import { Btn, btnClass, Badge, Card, ErrorBanner } from '../components/primitives'

type AttachMsg = {
  type: string
  data?: string
  cols?: number
  rows?: number
  message?: string
  workspaceSessionID?: string
  clientID?: string
  role?: string
  driverID?: string
  leaseMS?: number
}

type ActivityEvent = {
  id: number
  timestamp: number
  type: string
  message: string
}

export function AttachPage() {
  const { workspaceSessionID } = useParams({ strict: false })
  const id = workspaceSessionID ?? ''
  const { token, invalidateToken } = useAuth()
  const search = useSearch({ strict: false }) as { role?: 'viewer' | 'driver' }
  const role = search.role === 'driver' ? 'driver' : 'viewer'

  const { fullscreen, inspectorOpen, activityOpen, setFullscreen, setInspectorOpen, setActivityOpen, toggleFullscreen, toggleInspector, toggleActivity } = useAttachLayout()

  const [status, setStatus] = useState('initializing')
  const [err, setErr] = useState<string | null>(null)
  const [hello, setHello] = useState<{ clientID: string; role: string; driverID: string; leaseMS: number } | null>(null)
  const [driverState, setDriverState] = useState<{ driverID: string; leaseMS: number } | null>(null)
  const [engineName, setEngineName] = useState<string | null>(null)
  const [activityEvents, setActivityEvents] = useState<ActivityEvent[]>([])

  const wsRef = useRef<WebSocket | null>(null)
  const termRef = useRef<TerminalHandle>(null)
  const activityIdRef = useRef(1)

  const addActivityEvent = useCallback((type: string, message: string) => {
    const nextID = activityIdRef.current
    activityIdRef.current += 1
    setActivityEvents((prev) => [{ id: nextID, timestamp: Date.now(), type, message }, ...prev].slice(0, 120))
  }, [])

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const handleTerminalResize = useCallback((dims: { cols: number; rows: number }) => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }))
  }, [])

  const handleTerminalInput = useCallback((data: string) => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    const bytes = new TextEncoder().encode(data)
    ws.send(JSON.stringify({ type: 'stdin', data: base64EncodeBytes(bytes) }))
  }, [])

  useEffect(() => {
    if (token.trim() === '' || id === '') return
    let alive = true
    let keepaliveTimer: number | null = null
    let ws: WebSocket | null = null

    const run = async () => {
      setStatus('setting cookie')
      try {
        await api.createAttachCookie(token, id, role)
        if (!alive) return
        setStatus('connecting')

        const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
        const url = `${proto}://${window.location.host}/api/v1/workspace-sessions/${encodeURIComponent(id)}/attach`

        ws = new WebSocket(url)
        wsRef.current = ws

        ws.onopen = () => {
          setStatus('connected')
          addActivityEvent('connected', 'WebSocket connected')

          const dims = termRef.current?.dimensions()
          if (dims && dims.cols > 0 && dims.rows > 0) {
            ws?.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }))
          }

          keepaliveTimer = window.setInterval(() => {
            try {
              ws?.send(JSON.stringify({ type: 'keepalive' }))
            } catch {
              // ignore
            }
          }, 10_000)
        }

        ws.onmessage = (ev) => {
          let m: AttachMsg
          try {
            m = JSON.parse(String(ev.data)) as AttachMsg
          } catch {
            return
          }

          if (m.type === 'stdout' && m.data) {
            termRef.current?.write(base64DecodeToBytes(m.data))
            return
          }
          if (m.type === 'error') {
            const msg = m.message ?? 'unknown'
            termRef.current?.write(new TextEncoder().encode(`\r\n[error] ${msg}\r\n`))
            addActivityEvent('error', msg)
            return
          }
          if (m.type === 'hello') {
            setHello({ clientID: m.clientID ?? '', role: m.role ?? '', driverID: m.driverID ?? '', leaseMS: m.leaseMS ?? 0 })
            addActivityEvent('hello', `Client ${m.clientID ?? '-'} (${m.role ?? '-'})`)
            return
          }
          if (m.type === 'state') {
            setDriverState({ driverID: m.driverID ?? '', leaseMS: m.leaseMS ?? 0 })
            addActivityEvent('state', `Driver ${m.driverID ?? '-'} lease ${m.leaseMS ?? 0}ms`)
            return
          }
          if (m.type === 'backend_closed') {
            termRef.current?.write(new TextEncoder().encode('\r\n[backend closed]\r\n'))
            addActivityEvent('backend_closed', 'Backend closed session')
            return
          }
        }

        ws.onerror = () => {
          setErr('websocket error')
          addActivityEvent('error', 'WebSocket error')
        }
        ws.onclose = () => {
          termRef.current?.write(new TextEncoder().encode('\r\n[disconnected]\r\n'))
          setStatus('disconnected')
          addActivityEvent('disconnected', 'WebSocket disconnected')
          if (keepaliveTimer !== null) window.clearInterval(keepaliveTimer)
          keepaliveTimer = null
        }
      } catch (e) {
        if (isUnauthorizedError(e)) {
          onUnauthorized()
          setErr('unauthorized (401)')
          setStatus('error')
          return
        }
        setErr(e instanceof Error ? e.message : String(e))
        setStatus('error')
      }
    }

    run()
    return () => {
      alive = false
      if (keepaliveTimer !== null) window.clearInterval(keepaliveTimer)
      keepaliveTimer = null
      ws?.close()
      wsRef.current = null
    }
  }, [token, id, role, onUnauthorized, addActivityEvent])

  const takeControl = useCallback(() => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({ type: 'take_control' }))
    addActivityEvent('take_control', 'Requested control')
  }, [addActivityEvent])

  const clearActivity = useCallback(() => {
    setActivityEvents([])
  }, [])

  const handlePanelResize = useCallback(() => {
    termRef.current?.fit()
  }, [])

  const statusVariant = status === 'connected' ? 'ok' : status === 'error' || status === 'disconnected' ? 'bad' : 'neutral'

  const activityPanel = (
    <>
      {!activityOpen && (
        <div className="flex items-center justify-between px-3 py-1.5 border-t border-border/40 bg-card/50 shrink-0">
          <span className="text-[10px] text-muted-foreground">{activityEvents.length} events</span>
          <Btn variant="ghost" onClick={() => setActivityOpen(true)} type="button" className="text-xs px-2 py-1">
            Show Activity
          </Btn>
        </div>
      )}
      {activityOpen && (
        <div className="border-t border-border/40 bg-card/50 shrink-0 max-h-48 overflow-hidden flex flex-col">
          <div className="flex items-center justify-between px-3 py-1.5 border-b border-border/40">
            <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              Activity ({activityEvents.length})
            </span>
            <div className="flex gap-1">
              <Btn variant="ghost" onClick={clearActivity} type="button" className="text-xs px-2 py-1">Clear</Btn>
              <Btn variant="ghost" onClick={() => setActivityOpen(false)} type="button" className="text-xs px-2 py-1">Hide</Btn>
            </div>
          </div>
          <div className="overflow-y-auto flex-1 p-2">
            {activityEvents.length === 0 ? (
              <div className="text-[10px] text-muted-foreground/50 text-center py-2">No activity yet</div>
            ) : (
              <div className="space-y-1">
                {activityEvents.map((evt) => (
                  <div key={evt.id} className="text-[10px] font-mono">
                    <span className="text-muted-foreground/50">{new Date(evt.timestamp).toLocaleTimeString()}</span>{' '}
                    <span className="text-primary/80">[{evt.type}]</span>{' '}
                    <span className="text-foreground/80">{evt.message}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </>
  )

  return (
    <>
      {!fullscreen && <Topbar title={`Attach ${id}`} subtitle="Live terminal — viewer or driver mode via websocket." />}

      <div className={cn('flex overflow-hidden', fullscreen ? 'h-screen' : 'flex-1 p-4')}>
        <div className="flex-1 min-w-0 flex flex-col overflow-hidden">
          {fullscreen ? (
            <div className="flex flex-col min-h-0 flex-1">
              <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border/40 bg-card/50 shrink-0">
                <Badge variant={statusVariant}>{status}</Badge>
                <Badge variant={role === 'driver' ? 'info' : 'neutral'}>{role}</Badge>
                {engineName && <Badge variant="neutral">{engineName}</Badge>}
                <span className="text-[10px] font-mono text-muted-foreground flex-1">{hello?.clientID ?? ''} | {id}</span>
                <Btn variant="ghost" onClick={toggleInspector} type="button">Inspector</Btn>
                <Btn variant="ghost" onClick={toggleActivity} type="button">Activity</Btn>
                <Btn variant="ghost" onClick={toggleFullscreen} type="button">Exit Fullscreen</Btn>
              </div>
              <div className="flex flex-col min-h-0 flex-1 rounded-none border-0 bg-card overflow-hidden">
                <GhosttyTerminal
                  ref={termRef}
                  className="flex-1 min-h-0"
                  onInput={handleTerminalInput}
                  onResize={handleTerminalResize}
                  onReady={setEngineName}
                  disableInput={role !== 'driver'}
                />
                {activityPanel}
              </div>
            </div>
          ) : (
            <ResizablePanel
              id={`attach-${id}`}
              direction="vertical"
              defaultSize={120}
              minSize={80}
              maxSize={300}
              onResize={handlePanelResize}
              className="flex-1"
            >
              <Card>
                <div className="flex items-center gap-3 flex-wrap">
                  <Badge variant={statusVariant}>{status}</Badge>
                  <Badge variant={role === 'driver' ? 'info' : 'neutral'}>{role}</Badge>
                  {engineName && <Badge variant="neutral">{engineName}</Badge>}
                  <span className="text-[10px] font-mono text-muted-foreground">
                    client: {hello?.clientID ?? '…'} | driver: {driverState?.driverID ?? hello?.driverID ?? '—'} | lease: {String(driverState?.leaseMS ?? hello?.leaseMS ?? 0)}ms
                  </span>
                  <div className="flex items-center gap-1.5 ml-auto">
                    <Btn onClick={takeControl} type="button">Seize Control</Btn>
                    <Link className={btnClass('ghost')} to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: id }}>← Session</Link>
                    <Btn variant="ghost" onClick={toggleInspector} type="button">Inspector</Btn>
                    <Btn variant="ghost" onClick={toggleActivity} type="button">Activity</Btn>
                    <Btn variant="ghost" onClick={toggleFullscreen} type="button">Fullscreen</Btn>
                  </div>
                </div>
                {err ? <ErrorBanner>{err}</ErrorBanner> : null}
                {token.trim() === '' ? <ErrorBanner>No bearer token set. Auth required.</ErrorBanner> : null}
              </Card>

              <div className="flex flex-col min-h-0 rounded-lg border border-border/60 bg-card overflow-hidden">
                <GhosttyTerminal
                  ref={termRef}
                  className="flex-1 min-h-0"
                  onInput={handleTerminalInput}
                  onResize={handleTerminalResize}
                  onReady={setEngineName}
                  disableInput={role !== 'driver'}
                />
                {activityPanel}
              </div>
            </ResizablePanel>
          )}
        </div>

        <InspectorPanel open={inspectorOpen} title="Inspector" onClose={() => setInspectorOpen(false)}>
          <div className="space-y-3">
            <InspectorRow label="Session" value={id} />
            <InspectorRow label="Client ID" value={hello?.clientID ?? '—'} />
            <InspectorRow label="Role" value={role} badge={role === 'driver' ? 'info' : 'neutral'} />
            <InspectorRow label="Driver ID" value={driverState?.driverID ?? hello?.driverID ?? '—'} />
            <InspectorRow label="Lease" value={`${String(driverState?.leaseMS ?? hello?.leaseMS ?? 0)}ms`} />
            <InspectorRow label="Engine" value={engineName ?? '—'} />
            <div>
              <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">WebSocket Status</div>
              <Badge variant={statusVariant}>{status}</Badge>
            </div>
            <div>
              <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">Recent Activity (8)</div>
              <div className="space-y-1">
                {activityEvents.slice(0, 8).map((evt) => (
                  <div key={evt.id} className="text-[10px] font-mono">
                    <span className="text-muted-foreground/50">{new Date(evt.timestamp).toLocaleTimeString()}</span>{' '}
                    <span className="text-primary/80">[{evt.type}]</span>{' '}
                    <span className="text-foreground/80">{evt.message}</span>
                  </div>
                ))}
                {activityEvents.length === 0 && <div className="text-[10px] text-muted-foreground/50">No activity yet</div>}
              </div>
            </div>
          </div>
        </InspectorPanel>
      </div>
    </>
  )
}

function InspectorRow({
  label,
  value,
  badge,
}: {
  label: string
  value: string
  badge?: 'ok' | 'warn' | 'bad' | 'neutral' | 'info'
}) {
  return (
    <div>
      <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">{label}</div>
      {badge ? (
        <Badge variant={badge}>{value}</Badge>
      ) : (
        <div className="text-[11px] font-mono text-foreground/80 break-all">{value}</div>
      )}
    </div>
  )
}
