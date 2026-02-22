import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

type QueryState<T> = {
  data: T | null
  loading: boolean
  error: string | null
  reload: () => void
}

export function usePollingQuery<T>(
  key: string,
  fn: () => Promise<T>,
  opts: { intervalMs: number; enabled: boolean }
): QueryState<T> {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [nonce, setNonce] = useState(0)

  const reload = useCallback(() => {
    setNonce((n: number) => n + 1)
  }, [])

  const stableKey = useMemo(() => key, [key])

  useEffect(() => {
    if (!opts.enabled) return
    let mounted = true

    const run = async () => {
      setLoading(true)
      setError(null)
      try {
        const out = await fn()
        if (!mounted) return
        setData(out)
      } catch (e) {
        if (!mounted) return
        setError(e instanceof Error ? e.message : String(e))
      } finally {
        if (mounted) setLoading(false)
      }
    }

    run()
    const enableInterval = import.meta.env.MODE !== 'test'
    const t = enableInterval ? window.setInterval(run, Math.max(500, opts.intervalMs)) : null
    return () => {
      mounted = false
      if (t !== null) window.clearInterval(t)
    }
  }, [stableKey, fn, opts.enabled, opts.intervalMs, nonce])

  return { data, loading, error, reload }
}
