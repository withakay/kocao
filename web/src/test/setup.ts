import '@testing-library/jest-dom/vitest'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from './server'

// xterm.js requires matchMedia and getContext in the browser; stub for jsdom
if (typeof window !== 'undefined') {
  window.matchMedia ??= (query: string) =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList

  HTMLCanvasElement.prototype.getContext = (() => null) as any
}

beforeAll(() =>
  server.listen({
    onUnhandledRequest(req, print) {
      // Allow non-API requests through (WASM, module imports, etc.)
      const url = new URL(req.url)
      if (url.pathname.startsWith('/api/')) {
        print.error()
      }
    },
  }),
)
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
