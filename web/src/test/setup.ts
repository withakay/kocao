import '@testing-library/jest-dom/vitest'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { server } from './server'

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
