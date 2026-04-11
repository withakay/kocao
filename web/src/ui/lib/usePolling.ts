import { useCallback, useEffect, useRef, useState } from 'react'
import { isUnauthorizedError } from './api'

type QueryState<T> = {
  data: T | null
  loading: boolean
  error: string | null
  reload: () => void
}

export function usePollingQuery<T>(
  key: string,
  fn: () => Promise<T>,
  opts: { intervalMs: number; enabled: boolean; onUnauthorized?: () => void }
): QueryState<T> {
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [nonce, setNonce] = useState(0)
  const fnRef = useRef(fn)
  const onUnauthorizedRef = useRef(opts.onUnauthorized)

  useEffect(() => {
    fnRef.current = fn
  }, [fn])

  useEffect(() => {
    onUnauthorizedRef.current = opts.onUnauthorized
  }, [opts.onUnauthorized])

  const reload = useCallback(() => {
    setNonce((n: number) => n + 1)
  }, [])

  useEffect(() => {
    void key
    void nonce

    if (!opts.enabled) {
      setData(null)
      setLoading(false)
      setError(null)
      return
    }
    let mounted = true
    let running = false

    const run = async () => {
      if (running) return
      running = true
      setLoading(true)
      setError(null)
      try {
        const out = await fnRef.current()
        if (!mounted) return
        setData(out)
      } catch (e) {
        if (!mounted) return
        if (isUnauthorizedError(e) && onUnauthorizedRef.current) {
          onUnauthorizedRef.current()
          setData(null)
          setError(null)
          return
        }
        setError(e instanceof Error ? e.message : String(e))
      } finally {
        if (mounted) setLoading(false)
        running = false
      }
    }

    run()
    const enableInterval = import.meta.env.MODE !== 'test'
    const t = enableInterval ? window.setInterval(run, Math.max(500, opts.intervalMs)) : null
    return () => {
      mounted = false
      if (t !== null) window.clearInterval(t)
    }
  }, [key, opts.enabled, opts.intervalMs, nonce])

  return { data, loading, error, reload }
}
