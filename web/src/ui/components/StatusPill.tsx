import { phaseTone } from '../lib/phases'

export function StatusPill({ phase }: { phase?: string }) {
  const t = phaseTone(phase)
  const dotClass = t === 'ok' ? 'dot dotOk' : t === 'warn' ? 'dot dotWarn' : t === 'bad' ? 'dot dotBad' : 'dot'
  return (
    <span className="pill" title={phase ?? 'Unknown'}>
      <span className={dotClass} />
      <span>{phase && phase.trim() !== '' ? phase : 'Unknown'}</span>
    </span>
  )
}
