import { useCallback, useEffect, useImperativeHandle, useRef, forwardRef, useState } from 'react'
import { createEngine, type CreateEngineOpts, type TerminalEngine } from './TerminalAdapter'

export type TerminalHandle = {
  /** Write raw bytes (decoded stdout) into the terminal. */
  write(data: Uint8Array | string): void
  /** Get current terminal dimensions. */
  dimensions(): { cols: number; rows: number }
  /** Fit terminal to its container. */
  fit(): void
  /** Focus the terminal. */
  focus(): void
}

type GhosttyTerminalProps = {
  /** Called when user types in the terminal. */
  onInput?: (data: string) => void
  /** Called when terminal dimensions change. */
  onResize?: (dims: { cols: number; rows: number }) => void
  /** Called when engine is ready, with engine name ('ghostty' | 'xterm' | 'noop'). */
  onReady?: (engineName: string) => void
  /** Terminal theme colors. Treat as immutable — changing requires remount. */
  theme?: CreateEngineOpts['theme']
  /** Font size in pixels. */
  fontSize?: number
  /** CSS class for the container. */
  className?: string
  /** Disable input (viewer mode). */
  disableInput?: boolean
}

export const GhosttyTerminal = forwardRef<TerminalHandle, GhosttyTerminalProps>(function GhosttyTerminal(
  { onInput, onResize, onReady, theme, fontSize, className, disableInput },
  ref,
) {
  const containerRef = useRef<HTMLDivElement>(null)
  const engineRef = useRef<TerminalEngine | null>(null)
  const [ready, setReady] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Keep latest callbacks in refs to avoid re-creating the engine on callback changes
  const onInputRef = useRef(onInput)
  onInputRef.current = onInput
  const onResizeRef = useRef(onResize)
  onResizeRef.current = onResize
  const onReadyRef = useRef(onReady)
  onReadyRef.current = onReady
  const disableInputRef = useRef(disableInput)
  disableInputRef.current = disableInput

  // Serialize theme to avoid re-creating engine on referentially-new-but-equal objects
  const themeKey = useRef('')
  const themeRef = useRef(theme)
  const serialized = theme ? JSON.stringify(theme) : ''
  if (serialized !== themeKey.current) {
    themeKey.current = serialized
    themeRef.current = theme
  }

  useImperativeHandle(ref, () => ({
    write(data: Uint8Array | string) {
      engineRef.current?.write(data)
    },
    dimensions() {
      return engineRef.current?.dimensions() ?? { cols: 80, rows: 24 }
    },
    fit() {
      engineRef.current?.fit()
    },
    focus() {
      engineRef.current?.focus()
    },
  }), [])

  // Create engine and mount on container.
  // Theme is read from a ref (stable) so only fontSize triggers re-init.
  useEffect(() => {
    const el = containerRef.current
    if (!el) return

    let disposed = false

    const setup = async () => {
      try {
        const { engine, name } = await createEngine({
          theme: themeRef.current ?? defaultTheme,
          fontSize: fontSize ?? 13,
        })
        if (disposed) {
          engine.dispose()
          return
        }

        engine.onInput((data: string) => {
          if (!disableInputRef.current) onInputRef.current?.(data)
        })
        engine.onResize((dims: { cols: number; rows: number }) => {
          onResizeRef.current?.(dims)
        })
        engine.mount(el)
        engineRef.current = engine
        setReady(true)
        setError(null)
        onReadyRef.current?.(name)
      } catch (e) {
        console.warn('[GhosttyTerminal] engine setup failed:', e)
        setError(e instanceof Error ? e.message : 'Terminal engine failed to initialize')
      }
    }

    setup()

    return () => {
      disposed = true
      engineRef.current?.dispose()
      engineRef.current = null
      setReady(false)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- theme is tracked via themeKey ref
  }, [fontSize])

  // ResizeObserver to auto-fit when container changes size.
  // Uses RAF with cancellation to avoid queuing duplicate fit() calls.
  const fitOnResize = useCallback(() => {
    engineRef.current?.fit()
  }, [])

  useEffect(() => {
    const el = containerRef.current
    if (!el || !ready) return
    let rafId: number | null = null
    const ro = new ResizeObserver(() => {
      if (rafId !== null) cancelAnimationFrame(rafId)
      rafId = requestAnimationFrame(() => {
        rafId = null
        fitOnResize()
      })
    })
    ro.observe(el)
    return () => {
      ro.disconnect()
      if (rafId !== null) cancelAnimationFrame(rafId)
    }
  }, [ready, fitOnResize])

  if (error) {
    return (
      <div className={className} style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div className="text-xs text-destructive text-center px-4">
          <div className="font-medium mb-1">Terminal failed to load</div>
          <div className="text-muted-foreground">{error}</div>
        </div>
      </div>
    )
  }

  return (
    <div className="relative" style={{ width: '100%', height: '100%' }}>
      {!ready && (
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="text-xs text-muted-foreground animate-pulse">Loading terminal...</span>
        </div>
      )}
      <div
        ref={containerRef}
        className={className}
        style={{ width: '100%', height: '100%', overflow: 'hidden' }}
      />
    </div>
  )
})

/* ------------------------------------------------------------------ */
/*  Default dark terminal theme (matches the app's monochrome dark)   */
/* ------------------------------------------------------------------ */

const defaultTheme: CreateEngineOpts['theme'] = {
  foreground: '#ebebeb',
  background: '#1c1c1c',
  cursor: '#a3d9c8',
  selectionBackground: '#3a3a3a',
  black: '#1c1c1c',
  red: '#d4644a',
  green: '#5ebc8a',
  yellow: '#c9a651',
  blue: '#5c8abf',
  magenta: '#b07ab8',
  cyan: '#5eb8b8',
  white: '#cccccc',
  brightBlack: '#555555',
  brightRed: '#e8735a',
  brightGreen: '#72d4a0',
  brightYellow: '#dfc06b',
  brightBlue: '#74a5d4',
  brightMagenta: '#c994d1',
  brightCyan: '#78d4d4',
  brightWhite: '#eeeeee',
}
