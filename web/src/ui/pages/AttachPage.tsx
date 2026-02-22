import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { base64DecodeToBytes, base64EncodeBytes } from '../lib/base64'
import { Topbar } from '../components/Topbar'

type AttachMsg = {
  type: string
  data?: string
  cols?: number
  rows?: number
  message?: string
  sessionID?: string
  clientID?: string
  role?: string
  driverID?: string
  leaseMS?: number
}

export function AttachPage() {
  const { sessionID } = useParams()
  const id = sessionID ?? ''
  const { token } = useAuth()
  const [sp] = useSearchParams()
  const role = (sp.get('role') === 'driver' ? 'driver' : 'viewer') as 'viewer' | 'driver'

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
      setStatus('fetching token')
      try {
        const t = await api.createAttachToken(token, id, role)
        if (!alive) return
        setStatus('connecting')

        const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
        const url = `${proto}://${window.location.host}/api/v1/sessions/${encodeURIComponent(id)}/attach?token=${encodeURIComponent(
          t.token
        )}`

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
  }, [token, id, role, decoder])

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

  return (
    <>
      <Topbar title={`Attach ${id}`} subtitle="Interactive attach session (viewer/driver)" />

      <div className="grid">
        <section className="card">
          <div className="cardHeader">
            <h2>Connection</h2>
            <div className="muted">{status}</div>
          </div>

          <div className="formRow">
            <div className="label">Role</div>
            <div className="mono">{role}</div>
          </div>

          <div className="formRow">
            <div className="label">Client</div>
            <div className="mono">{hello?.clientID ?? '(pending)'}</div>
          </div>

          <div className="formRow">
            <div className="label">Driver</div>
            <div className="mono">{driverState?.driverID ?? hello?.driverID ?? '(none)'}</div>
          </div>

          <div className="formRow">
            <div className="label">Lease</div>
            <div className="mono">{String(driverState?.leaseMS ?? hello?.leaseMS ?? 0)}ms</div>
          </div>

          <div className="rowActions">
            <button className="btn" onClick={takeControl} type="button">
              Take Control
            </button>
            <Link className="btn" to={`/sessions/${encodeURIComponent(id)}`}>
              Back to Session
            </Link>
          </div>

          {err ? <div className="errorBox">{err}</div> : null}
          {token.trim() === '' ? <div className="errorBox">Set a bearer token in the top bar.</div> : null}
        </section>

        <section className="card">
          <div className="cardHeader">
            <h2>Terminal</h2>
            <div className="muted">stdout via websocket</div>
          </div>

          <div className="terminal mono" ref={termRef} aria-label="terminal" />

          <div className="rowActions">
            <input
              className="input"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={role === 'driver' ? 'Type a command and press Enter' : 'Read-only'}
              disabled={role !== 'driver'}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  sendLine()
                }
              }}
            />
            <button className="btn btnPrimary" onClick={sendLine} type="button" disabled={role !== 'driver'}>
              Send
            </button>
          </div>
        </section>
      </div>
    </>
  )
}
