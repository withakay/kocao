export type PhaseTone = 'ok' | 'warn' | 'bad' | 'neutral'

export function phaseTone(phase: string | undefined): PhaseTone {
  const p = (phase ?? '').toLowerCase()
  if (p === 'succeeded') return 'ok'
  if (p === 'running' || p === 'active') return 'warn'
  if (p === 'failed' || p === 'terminating') return 'bad'
  return 'neutral'
}
