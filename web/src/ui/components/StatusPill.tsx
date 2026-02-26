import { phaseTone } from '../lib/phases'
import { Badge } from './primitives'

const toneMap = { ok: 'ok', warn: 'warn', bad: 'bad', neutral: 'neutral' } as const

export function StatusPill({ phase }: { phase?: string }) {
  const t = phaseTone(phase)
  return (
    <Badge variant={toneMap[t]}>
      {phase && phase.trim() !== '' ? phase : 'Unknown'}
    </Badge>
  )
}
