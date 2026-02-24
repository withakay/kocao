import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams, useSearch } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { base64DecodeToBytes, base64EncodeBytes } from '../lib/base64'
import { Topbar } from '../components/Topbar'
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
  const { workspaceSessionID } = useParams({ strict: false })
  const id = workspaceSessionID ?? ''
  const { token, invalidateToken } = useAuth()
  const { role } = useSearch({ strict: false }) as { role: 'viewer' | 'driver' }

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const [status, setStatus] = useState('initializing')
  const [err, setErr] = useState<string | null>(null)
  const [hello, setHello] = useState<{ clientID: string; role: string; driverID: string; leaseMS: number } | null>(null)
  const [driverState, setDriverState] = useState<{ driverID: string; leaseMS: number } | null>(null)
  const [input, setInput] = useState('')

  const wsRef = useRef<WebSocket | null>(null)
  const termRef = useRef<HTMLDivElement | null>(null)
  const decoder = useMemo(() => new TextDecoder('utf-8'), [])

  useEffect(() => {
    if (token.trim() === '' || id === '') return
    let alive = true
    let keepaliveTimer: number | null = null
    let ws: WebSocket | null = null

    const append = (s: string) => {
      if (!termRef.current) return
      termRef.current.textContent = (termRef.current.textContent ?? '') + s
      termRef.current.scrollTop = termRef.current.scrollHeight
    }

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
          append(`[connected]\n`)
          setStatus('connected')

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
            const bytes = base64DecodeToBytes(m.data)
            append(decoder.decode(bytes))
            return
          }
          if (m.type === 'error') {
            append(`\n[error] ${m.message ?? 'unknown'}\n`)
            return
          }
          if (m.type === 'hello') {
            setHello({
              clientID: m.clientID ?? '',
              role: m.role ?? '',
              driverID: m.driverID ?? '',
              leaseMS: m.leaseMS ?? 0
            })
            return
          }
          if (m.type === 'state') {
            setDriverState({ driverID: m.driverID ?? '', leaseMS: m.leaseMS ?? 0 })
            return
          }
          if (m.type === 'backend_closed') {
            append(`\n[backend closed]\n`)
            return
          }
        }

        ws.onerror = () => {
          setErr('websocket error')
        }

        ws.onclose = () => {
          append(`\n[disconnected]\n`)
          setStatus('disconnected')
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
  }, [token, id, role, decoder, onUnauthorized])

  const sendLine = () => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    const bytes = new TextEncoder().encode(input + '\n')
    ws.send(JSON.stringify({ type: 'stdin', data: base64EncodeBytes(bytes) }))
    setInput('')
  }

  const takeControl = () => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({ type: 'take_control' }))
  }

  const cardClass = 'rounded-lg border border-border bg-card p-4'
  const headerClass = 'flex items-center justify-between mb-3'
  const rowClass = 'flex items-start gap-3 mb-3'
  const labelClass = 'text-xs text-muted-foreground w-24 shrink-0 pt-0.5'
  const btnClass =
    'rounded-md border border-border bg-secondary px-3 py-1.5 text-sm text-secondary-foreground hover:bg-secondary/80 transition-colors cursor-pointer'
  const errorClass = 'mt-3 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground'

  return (
    <>
      <Topbar title={`Attach ${id}`} subtitle="Live terminal — viewer or driver mode via websocket." />

      <div className="mt-4 flex flex-col gap-4">
        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">Connection</h2>
            <div
              className={cn(
                'text-xs font-mono px-2 py-0.5 rounded-full border',
                status === 'connected'
                  ? 'border-status-ok/30 text-status-ok bg-status-ok/10'
                  : status === 'error' || status === 'disconnected'
                    ? 'border-destructive/30 text-destructive bg-destructive/10'
                    : 'border-border text-muted-foreground bg-muted/50',
              )}
            >
              {status}
            </div>
          </div>

          <div className={rowClass}>
            <div className={labelClass}>Role</div>
            <div className={cn(
              'font-mono text-sm px-2 py-0.5 rounded border',
              role === 'driver'
                ? 'border-primary/30 bg-primary/10 text-foreground'
                : 'border-border bg-muted/50 text-muted-foreground',
            )}>
              {role}
            </div>
          </div>

          <div className={rowClass}>
            <div className={labelClass}>Client</div>
            <div className="font-mono text-sm">{hello?.clientID ?? '(pending)'}</div>
          </div>

          <div className={rowClass}>
            <div className={labelClass}>Driver</div>
            <div className="font-mono text-sm">{driverState?.driverID ?? hello?.driverID ?? '(none)'}</div>
          </div>

          <div className={rowClass}>
            <div className={labelClass}>Lease</div>
            <div className="font-mono text-sm">{String(driverState?.leaseMS ?? hello?.leaseMS ?? 0)}ms</div>
          </div>

          <div className="flex items-center gap-3 mt-1">
            <button className={btnClass} onClick={takeControl} type="button">
              Seize Control
            </button>
            <Link className={btnClass} to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: id }}>
              ← Session
            </Link>
          </div>

          {err ? <div className={errorClass}>{err}</div> : null}
          {token.trim() === '' ? <div className={errorClass}>No bearer token set. Auth required.</div> : null}
        </section>

        <section className={cardClass}>
          <div className={headerClass}>
            <h2 className="text-sm font-semibold tracking-tight">Terminal</h2>
            <div className="text-xs text-muted-foreground">stdout stream via websocket</div>
          </div>

          <div
            className="min-h-[320px] max-h-[600px] overflow-auto rounded-md border border-border bg-background p-3 font-mono text-sm text-foreground whitespace-pre-wrap"
            ref={termRef}
          />

          <div className="flex items-center gap-3 mt-3">
            <input
              className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground font-mono placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/40 focus:border-ring disabled:opacity-40 disabled:cursor-not-allowed"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={role === 'driver' ? 'stdin → enter to send' : 'read-only (viewer)'}
              disabled={role !== 'driver'}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  sendLine()
                }
              }}
            />
            <button
              className="rounded-md border border-primary/30 bg-primary/10 px-4 py-2 text-sm text-foreground hover:bg-primary/20 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
              onClick={sendLine}
              type="button"
              disabled={role !== 'driver'}
            >
              Send
            </button>
          </div>
        </section>
      </div>
    </>
  )
}
