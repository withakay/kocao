import { Link } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { Badge, btnClass } from './primitives'

export function Topbar({ title, subtitle, right }: { title: string; subtitle?: string; right?: React.ReactNode }) {
  const { token, notice, setNotice } = useAuth()
  const authed = token.trim() !== ''

  return (
    <div className="shrink-0">
      <div className="flex items-center justify-between gap-3 border-b border-border/40 bg-card/50 px-4 py-2">
        <div className="min-w-0">
          <h1 className="text-sm font-semibold tracking-tight truncate">{title}</h1>
          {subtitle ? <p className="text-[11px] text-muted-foreground truncate">{subtitle}</p> : null}
        </div>

        <div className="flex items-center gap-1.5 shrink-0">
          <a className={btnClass('ghost')} href="/docs" target="_blank" rel="noreferrer">
            Docs
          </a>
          <a className={btnClass('ghost')} href="/api/v1/scalar" target="_blank" rel="noreferrer">
            API
          </a>
          <Link className={btnClass('ghost')} to="/settings">
            Settings
          </Link>
          <Badge variant={authed ? 'ok' : 'warn'}>{authed ? 'Authed' : 'No token'}</Badge>
          {right}
        </div>
      </div>

      {notice ? (
        <div className="border-b border-border/40 bg-card/30 px-4 py-2">
          <div className="rounded-md bg-destructive/10 border border-destructive/20 px-3 py-1.5 text-xs text-destructive flex items-center justify-between gap-3">
            <span className="min-w-0 truncate">{notice}</span>
            <button className="underline shrink-0" type="button" onClick={() => setNotice(null)}>
              dismiss
            </button>
          </div>
        </div>
      ) : null}
    </div>
  )
}
