import { useEffect, useState } from 'react'
import { useAuth } from '../auth'
import { cn } from '@/lib/utils'

export function Topbar({ title, subtitle }: { title: string; subtitle: string }) {
  const { token, remember, notice, setRemember, setToken } = useAuth()
  const [draft, setDraft] = useState(token)
  const [reveal, setReveal] = useState(false)

  useEffect(() => {
    setDraft(token)
  }, [token])

  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border border-border bg-card p-3">
      <div className="flex flex-col gap-0.5">
        <h1 className="text-base font-semibold tracking-tight">{title}</h1>
        <p className="text-xs text-muted-foreground">{subtitle}</p>
      </div>
      <div className="flex items-center gap-2">
        <input
          className={cn(
            'w-72 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground',
            'placeholder:text-muted-foreground',
            'focus:outline-none focus:ring-2 focus:ring-ring/40 focus:border-ring',
          )}
          type={reveal ? 'text' : 'password'}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Bearer token (required)"
          aria-label="API token"
        />
        <label className="flex items-center gap-1.5 text-xs text-muted-foreground select-none">
          <input
            type="checkbox"
            checked={remember}
            onChange={(e) => setRemember(e.target.checked)}
            aria-label="Remember token"
            className="accent-primary"
          />
          Remember
        </label>
        <button
          className="rounded-md border border-border bg-secondary px-3 py-1.5 text-sm text-secondary-foreground hover:bg-secondary/80 transition-colors cursor-pointer"
          onClick={() => setReveal((v) => !v)}
          type="button"
        >
          {reveal ? 'Hide' : 'Show'}
        </button>
        <button
          className="rounded-md border border-primary/30 bg-primary/10 px-3 py-1.5 text-sm text-foreground hover:bg-primary/20 transition-colors cursor-pointer"
          onClick={() => setToken(draft)}
          type="button"
        >
          Save
        </button>

        {notice ? (
          <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-foreground">
            {notice}
          </div>
        ) : null}
      </div>
    </div>
  )
}
