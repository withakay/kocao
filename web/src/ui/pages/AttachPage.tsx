import { useCallback, useEffect, useRef, useState, useMemo } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { base64DecodeToBytes, base64EncodeBytes } from '../lib/base64'
import { Topbar } from '../components/Topbar'
import { Btn, btnClass, Badge, Card, ErrorBanner } from '../components/primitives'
import { GhosttyTerminal, type TerminalHandle } from '../components/GhosttyTerminal'
import { ResizablePanel } from '../components/ResizablePanel'
import { FullscreenContext } from '../lib/useLayoutState'
import { cn } from '@/lib/utils'

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

export function AttachPage() {
  const { workspaceSessionID } = useParams()
  const id = workspaceSessionID ?? ''
  const { token, invalidateToken } = useAuth()
  const [sp] = useSearchParams()
  const role = (sp.get('role') === 'driver' ? 'driver' : 'viewer') as 'viewer' | 'driver'

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const [status, setStatus] = useState('initializing')
  const [err, setErr] = useState<string | null>(null)
  const [hello, setHello] = useState<{ clientID: string; role: string; driverID: string; leaseMS: number } | null>(null)
  const [driverState, setDriverState] = useState<{ driverID: string; leaseMS: number } | null>(null)
  const [fullscreen, setFullscreen] = useState(false)
  const [engineName, setEngineName] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const termRef = useRef<TerminalHandle>(null)

  // Send resize message to backend when terminal dimensions change
  const handleTerminalResize = useCallback((dims: { cols: number; rows: number }) => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }))
  }, [])

  // Send input from terminal to backend as stdin
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

          // Send initial terminal dimensions
          const dims = termRef.current?.dimensions()
          if (dims && dims.cols > 0 && dims.rows > 0) {
            ws?.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }))
          }

          keepaliveTimer = window.setInterval(() => {
            try { ws?.send(JSON.stringify({ type: 'keepalive' })) } catch { /* ignore */ }
          }, 10_000)
        }

        ws.onmessage = (ev) => {
          let m: AttachMsg
          try { m = JSON.parse(String(ev.data)) as AttachMsg } catch { return }

          if (m.type === 'stdout' && m.data) {
            const bytes = base64DecodeToBytes(m.data)
            termRef.current?.write(bytes)
            return
          }
          if (m.type === 'error') {
            const msg = m.message ?? 'unknown'
            termRef.current?.write(new TextEncoder().encode(`\r\n[error] ${msg}\r\n`))
            return
          }
          if (m.type === 'hello') {
            setHello({ clientID: m.clientID ?? '', role: m.role ?? '', driverID: m.driverID ?? '', leaseMS: m.leaseMS ?? 0 })
            return
          }
          if (m.type === 'state') {
            setDriverState({ driverID: m.driverID ?? '', leaseMS: m.leaseMS ?? 0 })
            return
          }
          if (m.type === 'backend_closed') {
            termRef.current?.write(new TextEncoder().encode('\r\n[backend closed]\r\n'))
            return
          }
        }

        ws.onerror = () => { setErr('websocket error') }
        ws.onclose = () => {
          termRef.current?.write(new TextEncoder().encode('\r\n[disconnected]\r\n'))
          setStatus('disconnected')
          if (keepaliveTimer !== null) window.clearInterval(keepaliveTimer)
          keepaliveTimer = null
        }
      } catch (e) {
        if (isUnauthorizedError(e)) { onUnauthorized(); setErr('unauthorized (401)'); setStatus('error'); return }
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
  }, [token, id, role, onUnauthorized])

  const takeControl = () => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({ type: 'take_control' }))
  }

  const statusVariant = status === 'connected' ? 'ok' : status === 'error' || status === 'disconnected' ? 'bad' : 'neutral'

  const toggleFullscreen = useCallback(() => {
    setFullscreen((v) => !v)
  }, [])

  const fullscreenCtx = useMemo(() => ({
    fullscreen,
    toggleFullscreen,
  }), [fullscreen, toggleFullscreen])

  const handlePanelResize = useCallback(() => {
    termRef.current?.fit()
  }, [])

  return (
    <FullscreenContext.Provider value={fullscreenCtx}>
      {!fullscreen && <Topbar title={`Attach ${id}`} subtitle="Live terminal \u2014 viewer or driver mode via websocket." />}

      <div className={cn('flex flex-col overflow-hidden', fullscreen ? 'h-screen' : 'flex-1 p-4')}>
        {fullscreen ? (
          // Fullscreen mode: just terminal with minimal header
          <div className="flex flex-col min-h-0 flex-1">
            <div className="flex items-center gap-2 px-3 py-1.5 border-b border-border/40 bg-card/50 shrink-0">
              <Badge variant={statusVariant}>{status}</Badge>
              <Badge variant={role === 'driver' ? 'info' : 'neutral'}>{role}</Badge>
              {engineName && <Badge variant="neutral">{engineName}</Badge>}
              <span className="text-[10px] font-mono text-muted-foreground flex-1">
                {hello?.clientID ?? ''} | {id}
              </span>
              <Btn variant="ghost" onClick={() => setFullscreen(false)} type="button">Exit Fullscreen</Btn>
            </div>
            <GhosttyTerminal
              ref={termRef}
              className="flex-1 min-h-0"
              onInput={handleTerminalInput}
              onResize={handleTerminalResize}
              onReady={setEngineName}
              disableInput={role !== 'driver'}
            />
          </div>
        ) : (
          // Normal mode: resizable panel with connection info and terminal
          <ResizablePanel
            id={`attach-${id}`}
            direction="vertical"
            defaultSize={120}
            minSize={80}
            maxSize={300}
            onResize={handlePanelResize}
            className="flex-1"
          >
            {/* Connection info panel */}
            <Card>
              <div className="flex items-center gap-3 flex-wrap">
                <Badge variant={statusVariant}>{status}</Badge>
                <Badge variant={role === 'driver' ? 'info' : 'neutral'}>{role}</Badge>
                {engineName && <Badge variant="neutral">{engineName}</Badge>}
                <span className="text-[10px] font-mono text-muted-foreground">
                  client: {hello?.clientID ?? '\u2026'} | driver: {driverState?.driverID ?? hello?.driverID ?? '\u2014'} | lease: {String(driverState?.leaseMS ?? hello?.leaseMS ?? 0)}ms
                </span>
                <div className="flex items-center gap-1.5 ml-auto">
                  <Btn onClick={takeControl} type="button">Seize Control</Btn>
                  <Link className={btnClass('ghost')} to={`/workspace-sessions/${encodeURIComponent(id)}`}>\u2190 Session</Link>
                  <Btn variant="ghost" onClick={() => setFullscreen(true)} type="button">Fullscreen</Btn>
                </div>
              </div>
              {err ? <ErrorBanner>{err}</ErrorBanner> : null}
              {token.trim() === '' ? <ErrorBanner>No bearer token set. Auth required.</ErrorBanner> : null}
            </Card>

            {/* Terminal panel */}
            <div className="flex flex-col min-h-0 rounded-lg border border-border/60 bg-card overflow-hidden">
              <GhosttyTerminal
                ref={termRef}
                className="flex-1 min-h-0"
                onInput={handleTerminalInput}
                onResize={handleTerminalResize}
                onReady={setEngineName}
                disableInput={role !== 'driver'}
              />
            </div>
          </ResizablePanel>
        )}
      </div>
    </FullscreenContext.Provider>
  )
}
