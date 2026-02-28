/**
 * Terminal engine adapter boundary.
 *
 * Defines a renderer-agnostic interface for terminal engines and provides
 * concrete adapters for xterm.js and ghostty-web. The attach transport layer
 * programs against TerminalAdapter only so engines are hot-swappable.
 */

// ---------------------------------------------------------------------------
// Adapter interface
// ---------------------------------------------------------------------------

export type TerminalEngineType = 'xterm' | 'ghostty'

export interface TerminalAdapter {
  /** Mount the terminal into a container element. */
  open(container: HTMLElement): void

  /** Write output data (raw bytes decoded to string) to the terminal. */
  write(data: string): void

  /** Register a callback for user input. Returns an unsubscribe function. */
  onData(cb: (data: string) => void): () => void

  /** Fit terminal to container dimensions (best-effort). */
  fit(): void

  /** Tear down the terminal and release resources. */
  dispose(): void
}

// ---------------------------------------------------------------------------
// xterm.js adapter
// ---------------------------------------------------------------------------

export async function createXtermAdapter(): Promise<TerminalAdapter> {
  const { Terminal } = await import('@xterm/xterm')
  const { FitAddon } = await import('@xterm/addon-fit')

  const term = new Terminal({
    fontSize: 14,
    fontFamily: 'ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, monospace',
    theme: {
      background: '#09090b',
      foreground: '#fafafa',
      cursor: '#fafafa',
    },
    cursorBlink: true,
    convertEol: true,
  })
  const fitAddon = new FitAddon()
  term.loadAddon(fitAddon)

  return {
    open(container: HTMLElement) {
      term.open(container)
      fitAddon.fit()
    },
    write(data: string) {
      term.write(data)
    },
    onData(cb: (data: string) => void) {
      const disposable = term.onData(cb)
      return () => disposable.dispose()
    },
    fit() {
      fitAddon.fit()
    },
    dispose() {
      term.dispose()
    },
  }
}

// ---------------------------------------------------------------------------
// ghostty-web adapter
// ---------------------------------------------------------------------------

export async function createGhosttyAdapter(): Promise<TerminalAdapter> {
  const ghostty = await import('ghostty-web')
  await ghostty.init()

  const term = new ghostty.Terminal({
    fontSize: 14,
    fontFamily: 'ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, Consolas, monospace',
    theme: {
      background: '#09090b',
      foreground: '#fafafa',
      cursor: '#fafafa',
    },
  })

  return {
    open(container: HTMLElement) {
      term.open(container)
    },
    write(data: string) {
      term.write(data)
    },
    onData(cb: (data: string) => void) {
      const disposable = term.onData(cb)
      return () => disposable.dispose()
    },
    fit() {
      // ghostty-web auto-sizes; fit is best-effort / no-op if unsupported
      if (typeof (term as any).fit === 'function') {
        ;(term as any).fit()
      }
    },
    dispose() {
      term.dispose()
    },
  }
}

// ---------------------------------------------------------------------------
// Factory
// ---------------------------------------------------------------------------

export async function createTerminalAdapter(engine: TerminalEngineType): Promise<TerminalAdapter> {
  if (engine === 'ghostty') {
    return createGhosttyAdapter()
  }
  return createXtermAdapter()
}
