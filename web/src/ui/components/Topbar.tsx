import { useEffect, useState } from 'react'
import { useAuth } from '../auth'
import { Btn, Input, ErrorBanner } from './primitives'

export function Topbar({ title, subtitle }: { title: string; subtitle?: string }) {
  const { token, remember, notice, setRemember, setToken } = useAuth()
  const [draft, setDraft] = useState(token)
  const [reveal, setReveal] = useState(false)

  useEffect(() => {
    setDraft(token)
  }, [token])

  return (
    <div className="flex items-center justify-between gap-3 border-b border-border/40 bg-card/50 px-4 py-2 shrink-0">
      <div className="min-w-0">
        <h1 className="text-sm font-semibold tracking-tight truncate">{title}</h1>
        {subtitle ? <p className="text-[11px] text-muted-foreground truncate">{subtitle}</p> : null}
      </div>
      <div className="flex items-center gap-1.5 shrink-0">
        <Input
          className="w-56 !py-1 !text-xs"
          type={reveal ? 'text' : 'password'}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Bearer token"
          aria-label="API token"
        />
        <label className="flex items-center gap-1 text-[10px] text-muted-foreground select-none cursor-pointer">
          <input
            type="checkbox"
            checked={remember}
            onChange={(e) => setRemember(e.target.checked)}
            aria-label="Remember token"
            className="accent-primary"
          />
          Keep
        </label>
        <Btn variant="ghost" onClick={() => setReveal((v) => !v)} type="button">
          {reveal ? 'Hide' : 'Show'}
        </Btn>
        <Btn variant="primary" onClick={() => setToken(draft)} type="button">
          Save
        </Btn>
        {notice ? <ErrorBanner>{notice}</ErrorBanner> : null}
      </div>
    </div>
  )
}
