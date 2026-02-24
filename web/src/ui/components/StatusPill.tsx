import { phaseTone } from '../lib/phases'
import { cn } from '@/lib/utils'

export function StatusPill({ phase }: { phase?: string }) {
  const t = phaseTone(phase)
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-full border border-border bg-muted/50 px-2.5 py-1 text-xs text-foreground"
      title={phase ?? 'Unknown'}
    >
      <span
        className={cn(
          'size-2 rounded-full',
          t === 'ok' && 'bg-status-ok',
          t === 'warn' && 'bg-status-warn',
          t === 'bad' && 'bg-status-bad',
          t === 'neutral' && 'bg-muted-foreground',
        )}
      />
      <span>{phase && phase.trim() !== '' ? phase : 'Unknown'}</span>
    </span>
  )
}
