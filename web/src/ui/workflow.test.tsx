import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it, vi } from 'vitest'
import { server } from '../test/server'
import { App } from './App'

type WorkspaceSession = { id: string; repoURL?: string; phase?: string }
type HarnessRun = {
  id: string
  workspaceSessionID?: string
  repoURL: string
  repoRevision?: string
  image: string
  phase?: string
  podName?: string
  gitHubBranch?: string
  pullRequestURL?: string
  pullRequestStatus?: string
}

describe('workflow-ui-github', () => {
  it('stores token in session storage by default and local storage when remembered', async () => {
    localStorage.removeItem('kocao.apiToken')
    sessionStorage.removeItem('kocao.apiToken')
    window.location.hash = '#/workspace-sessions'

    server.use(
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] })),
      http.get('/api/v1/workspace-sessions', () => HttpResponse.json({ workspaceSessions: [] }))
    )

    const { unmount } = render(<App />)

    const input = await screen.findByLabelText('API token')
    await userEvent.clear(input)
    await userEvent.type(input, 't-123')

    const save = await screen.findByRole('button', { name: 'Save' })
    await userEvent.click(save)

    expect(sessionStorage.getItem('kocao.apiToken')).toBe('t-123')
    expect(localStorage.getItem('kocao.apiToken')).toBeNull()

    const remember = await screen.findByLabelText('Remember token')
    await userEvent.click(remember)
    expect(localStorage.getItem('kocao.apiToken')).toBe('t-123')
    expect(sessionStorage.getItem('kocao.apiToken')).toBeNull()

    await userEvent.click(remember)
    expect(sessionStorage.getItem('kocao.apiToken')).toBe('t-123')
    expect(localStorage.getItem('kocao.apiToken')).toBeNull()

    unmount()
  })

  it('creates a session, starts a run, and shows PR outcome metadata', async () => {
    sessionStorage.setItem('kocao.apiToken', 't-full')
    window.location.hash = '#/workspace-sessions'

    const workspaceSessions: WorkspaceSession[] = []
    const harnessRuns: HarnessRun[] = []

    server.use(
      http.get('/api/v1/workspace-sessions', () => HttpResponse.json({ workspaceSessions })),
      http.post('/api/v1/workspace-sessions', async (ctx: any) => {
        const b = (await ctx.request.json()) as { repoURL?: string }
        const s: WorkspaceSession = { id: 'sess-1', repoURL: b.repoURL ?? '', phase: 'Active' }
        workspaceSessions.unshift(s)
        return HttpResponse.json(s, { status: 201 })
      }),
      http.get('/api/v1/workspace-sessions/:id', (ctx: any) => {
        const s = workspaceSessions.find((x) => x.id === ctx.params.id)
        if (!s) return new HttpResponse('not found', { status: 404 })
        return HttpResponse.json(s)
      }),
      http.get('/api/v1/harness-runs', (ctx: any) => {
        const url = new URL(ctx.request.url)
        const sid = url.searchParams.get('workspaceSessionID')
        const out = sid ? harnessRuns.filter((r) => r.workspaceSessionID === sid) : harnessRuns
        return HttpResponse.json({ harnessRuns: out })
      }),
      http.get('/api/v1/harness-runs/:id', (ctx: any) => {
        const r = harnessRuns.find((x) => x.id === ctx.params.id)
        if (!r) return new HttpResponse('not found', { status: 404 })
        return HttpResponse.json(r)
      }),
      http.post('/api/v1/workspace-sessions/:id/harness-runs', async (ctx: any) => {
        const b = (await ctx.request.json()) as { repoURL: string; repoRevision?: string; image: string }
        const r: HarnessRun = {
          id: 'run-1',
          workspaceSessionID: String(ctx.params.id),
          repoURL: b.repoURL,
          repoRevision: b.repoRevision ?? 'main',
          image: b.image,
          phase: 'Succeeded',
          podName: 'pod-123',
          gitHubBranch: 'feature/mvp-ui',
          pullRequestURL: 'https://github.com/withakay/kocao/pull/123',
          pullRequestStatus: 'merged'
        }
        harnessRuns.unshift(r)
        return HttpResponse.json(r, { status: 201 })
      }),
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] }))
    )

    const { unmount } = render(<App />)

    const create = await screen.findByRole('button', { name: 'Create Workspace Session' })
    await userEvent.click(create)

    await screen.findByRole('heading', { name: /Workspace Session sess-1/ })

    // Ensure the start-run form is actionable even if session data is still loading.
    const repoInput = await screen.findByPlaceholderText('defaults to workspace session repoURL')
    await userEvent.clear(repoInput)
    await userEvent.type(repoInput, 'https://example.com/repo')

    const start = await screen.findByRole('button', { name: 'Start Harness Run' })
    await userEvent.click(start)

    await screen.findByRole('heading', { name: /Harness Run run-1/ })
    await screen.findByText('Succeeded')

    const outcome = await screen.findByRole('heading', { name: 'GitHub Outcome' })
    const card = outcome.closest('section') ?? outcome.parentElement
    expect(card).toBeTruthy()

    const prLink = await within(card as HTMLElement).findByRole('link', { name: 'https://github.com/withakay/kocao/pull/123' })
    expect(prLink).toHaveAttribute('href', 'https://github.com/withakay/kocao/pull/123')
    within(card as HTMLElement).getByText('feature/mvp-ui')
    within(card as HTMLElement).getByText('merged')

    unmount()
  })
})

describe('auth-failures', () => {
  it('clears token on 401 and prompts to re-enter credentials', async () => {
    sessionStorage.setItem('kocao.apiToken', 't-bad')
    localStorage.removeItem('kocao.apiToken')
    window.location.hash = '#/workspace-sessions'

    server.use(
      http.get('/api/v1/workspace-sessions', () => new HttpResponse('unauthorized', { status: 401 })),
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] }))
    )

    const { unmount } = render(<App />)

    await screen.findByText(/Bearer token rejected \(401\)/)
    expect(sessionStorage.getItem('kocao.apiToken')).toBeNull()
    expect(localStorage.getItem('kocao.apiToken')).toBeNull()

    await screen.findByText('Set a bearer token in the top bar to use the API.')

    unmount()
  })
})

describe('attach-ui', () => {
  it('establishes websocket without URL token', async () => {
    sessionStorage.setItem('kocao.apiToken', 't-full')
    window.location.hash = '#/workspace-sessions/sess-1/attach?role=viewer'

    let cookieCalls = 0
    server.use(
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] })),
      http.get('/api/v1/harness-runs/:id', () => new HttpResponse('not found', { status: 404 })),
      http.post('/api/v1/workspace-sessions/:id/attach-cookie', (ctx: any) => {
        cookieCalls += 1
        return HttpResponse.json(
          {
            expiresAt: new Date().toISOString(),
            workspaceSessionID: String(ctx.params.id),
            clientID: 'cli-1',
            role: 'viewer'
          },
          { status: 201 }
        )
      })
    )

    const urls: string[] = []

    class MockWebSocket {
      readonly url: string
      readyState = 1
      onopen: ((ev: any) => void) | null = null
      onmessage: ((ev: any) => void) | null = null
      onerror: ((ev: any) => void) | null = null
      onclose: ((ev: any) => void) | null = null

      constructor(url: string) {
        this.url = url
        urls.push(url)
        setTimeout(() => {
          this.onopen?.({})
        }, 0)
      }

      send(_data: any) {
        // no-op
      }

      close() {
        this.readyState = 3
        this.onclose?.({})
      }
    }

    vi.stubGlobal('WebSocket', MockWebSocket)
    try {
      const { unmount } = render(<App />)

      await screen.findByText('connected')
      expect(cookieCalls).toBeGreaterThan(0)
      expect(urls.length).toBeGreaterThan(0)
      for (const u of urls) {
        expect(u).not.toContain('token=')
      }
      expect(urls[urls.length - 1]).toContain('/api/v1/workspace-sessions/sess-1/attach')

      unmount()
    } finally {
      vi.unstubAllGlobals()
    }
  })
})
