/**
 * Terminal engine adapter interface.
 *
 * Decouples attach transport logic from the concrete terminal renderer.
 * ghostty-web is the primary engine; xterm.js is the fallback.
 * A NoopEngine is returned in test/headless environments where neither
 * engine can initialize (no canvas, no WASM).
 */

export interface TerminalEngine {
  /** Mount the terminal into a container element. */
  mount(el: HTMLElement): void
  /** Write UTF-8 bytes or a string to the terminal. */
  write(data: Uint8Array | string): void
  /** Resize the terminal grid. */
  resize(cols: number, rows: number): void
  /** Tear down the terminal and free all resources. */
  dispose(): void
  /** Register a callback for user input (keystrokes, paste). */
  onInput(cb: (data: string) => void): void
  /** Register a callback for terminal resize events. */
  onResize(cb: (dims: { cols: number; rows: number }) => void): void
  /** Get current terminal dimensions. */
  dimensions(): { cols: number; rows: number }
  /** Fit the terminal to its container. */
  fit(): void
  /** Focus the terminal. */
  focus(): void
}

/* ------------------------------------------------------------------ */
/*  ghostty-web engine                                                */
/* ------------------------------------------------------------------ */

class GhosttyEngine implements TerminalEngine {
  private term: import('ghostty-web').Terminal | null = null
  private fitAddon: import('ghostty-web').FitAddon | null = null
  private inputCb: ((data: string) => void) | null = null
  private resizeCb: ((dims: { cols: number; rows: number }) => void) | null = null
  private dataDisposable: import('ghostty-web').IDisposable | null = null
  private resizeDisposable: import('ghostty-web').IDisposable | null = null

  async init(opts: { theme?: import('ghostty-web').ITheme; fontSize?: number; fontFamily?: string }) {
    const ghostty = await import('ghostty-web')
    await ghostty.init()
    this.term = new ghostty.Terminal({
      cols: 80,
      rows: 24,
      cursorBlink: true,
      cursorStyle: 'bar',
      fontSize: opts.fontSize ?? 13,
      fontFamily: opts.fontFamily ?? 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      theme: opts.theme,
      scrollback: 5000,
    })
    this.fitAddon = new ghostty.FitAddon()
    this.term.loadAddon(this.fitAddon)
  }

  mount(el: HTMLElement) {
    if (!this.term) throw new Error('ghostty-web not initialized')
    // Register handlers BEFORE open() so no events are lost during init
    this.dataDisposable = this.term.onData((data: string) => this.inputCb?.(data))
    this.resizeDisposable = this.term.onResize((dims: { cols: number; rows: number }) => this.resizeCb?.(dims))
    this.term.open(el)
    this.fitAddon?.fit()
    this.term.focus()
  }

  write(data: Uint8Array | string) {
    this.term?.write(data)
  }

  resize(cols: number, rows: number) {
    this.term?.resize(cols, rows)
  }

  dispose() {
    // Dispose listeners first, then addons, then terminal
    this.dataDisposable?.dispose()
    this.dataDisposable = null
    this.resizeDisposable?.dispose()
    this.resizeDisposable = null
    this.fitAddon?.dispose()
    this.fitAddon = null
    this.term?.dispose()
    this.term = null
  }

  onInput(cb: (data: string) => void) {
    this.inputCb = cb
  }

  onResize(cb: (dims: { cols: number; rows: number }) => void) {
    this.resizeCb = cb
  }

  dimensions() {
    return { cols: this.term?.cols ?? 80, rows: this.term?.rows ?? 24 }
  }

  fit() {
    this.fitAddon?.fit()
  }

  focus() {
    this.term?.focus()
  }
}

/* ------------------------------------------------------------------ */
/*  xterm.js engine (fallback)                                        */
/* ------------------------------------------------------------------ */

class XtermEngine implements TerminalEngine {
  private term: import('@xterm/xterm').Terminal | null = null
  private fitAddon: import('@xterm/addon-fit').FitAddon | null = null
  private inputCb: ((data: string) => void) | null = null
  private resizeCb: ((dims: { cols: number; rows: number }) => void) | null = null
  private dataDisposable: import('@xterm/xterm').IDisposable | null = null
  private resizeDisposable: import('@xterm/xterm').IDisposable | null = null

  async init(opts: { theme?: Record<string, string>; fontSize?: number; fontFamily?: string }) {
    const { Terminal } = await import('@xterm/xterm')
    const { FitAddon } = await import('@xterm/addon-fit')
    this.term = new Terminal({
      cols: 80,
      rows: 24,
      cursorBlink: true,
      cursorStyle: 'bar',
      fontSize: opts.fontSize ?? 13,
      fontFamily: opts.fontFamily ?? 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      theme: opts.theme as import('@xterm/xterm').ITheme | undefined,
      scrollback: 5000,
    })
    this.fitAddon = new FitAddon()
    this.term.loadAddon(this.fitAddon)
  }

  mount(el: HTMLElement) {
    if (!this.term) throw new Error('xterm.js not initialized')
    // Register handlers BEFORE open() so no events are lost during init
    this.dataDisposable = this.term.onData((data: string) => this.inputCb?.(data))
    this.resizeDisposable = this.term.onResize((dims: { cols: number; rows: number }) => this.resizeCb?.(dims))
    this.term.open(el)
    this.fitAddon?.fit()
    this.term.focus()
  }

  write(data: Uint8Array | string) {
    this.term?.write(data)
  }

  resize(cols: number, rows: number) {
    this.term?.resize(cols, rows)
  }

  dispose() {
    this.dataDisposable?.dispose()
    this.dataDisposable = null
    this.resizeDisposable?.dispose()
    this.resizeDisposable = null
    this.fitAddon?.dispose()
    this.fitAddon = null
    this.term?.dispose()
    this.term = null
  }

  onInput(cb: (data: string) => void) {
    this.inputCb = cb
  }

  onResize(cb: (dims: { cols: number; rows: number }) => void) {
    this.resizeCb = cb
  }

  dimensions() {
    return { cols: this.term?.cols ?? 80, rows: this.term?.rows ?? 24 }
  }

  fit() {
    this.fitAddon?.fit()
  }

  focus() {
    this.term?.focus()
  }
}

/* ------------------------------------------------------------------ */
/*  Factory: try ghostty-web first, fall back to xterm.js             */
/* ------------------------------------------------------------------ */

export type TerminalTheme = {
  foreground?: string
  background?: string
  cursor?: string
  selectionBackground?: string
  black?: string
  red?: string
  green?: string
  yellow?: string
  blue?: string
  magenta?: string
  cyan?: string
  white?: string
  brightBlack?: string
  brightRed?: string
  brightGreen?: string
  brightYellow?: string
  brightBlue?: string
  brightMagenta?: string
  brightCyan?: string
  brightWhite?: string
}

export type CreateEngineOpts = {
  theme?: TerminalTheme
  fontSize?: number
  fontFamily?: string
}

export async function createEngine(opts: CreateEngineOpts = {}): Promise<{ engine: TerminalEngine; name: string }> {
  // Try ghostty-web first
  try {
    const engine = new GhosttyEngine()
    await engine.init(opts)
    return { engine, name: 'ghostty' }
  } catch (e) {
    console.warn('[terminal] ghostty-web init failed, falling back to xterm.js:', e)
  }

  // Fall back to xterm.js
  try {
    const engine = new XtermEngine()
    await engine.init(opts)
    return { engine, name: 'xterm' }
  } catch (e) {
    console.warn('[terminal] xterm.js init also failed:', e)
  }

  // Both engines failed -- return a no-op engine (test/headless environments)
  return { engine: new NoopEngine(), name: 'noop' }
}

/* ------------------------------------------------------------------ */
/*  No-op engine (fallback for environments without canvas/WASM)      */
/* ------------------------------------------------------------------ */

class NoopEngine implements TerminalEngine {
  mount(_el: HTMLElement) { /* no-op */ }
  write(_data: Uint8Array | string) { /* no-op */ }
  resize(_cols: number, _rows: number) { /* no-op */ }
  dispose() { /* no-op */ }
  onInput(_cb: (data: string) => void) { /* no-op */ }
  onResize(_cb: (dims: { cols: number; rows: number }) => void) { /* no-op */ }
  dimensions() { return { cols: 80, rows: 24 } }
  fit() { /* no-op */ }
  focus() { /* no-op */ }
}
