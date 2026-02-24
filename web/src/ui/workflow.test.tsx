import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RouterProvider } from '@tanstack/react-router'
import { createMemoryHistory } from '@tanstack/react-router'
import { http, HttpResponse } from 'msw'
import { describe, expect, it, vi } from 'vitest'
import { server } from '../test/server'
import { createAppRouter } from './App'

type WorkspaceSession = { id: string; repoURL?: string | undefined; phase?: string | undefined }
type HarnessRun = {
  id: string
  workspaceSessionID?: string | undefined
  repoURL: string
  repoRevision?: string | undefined
  image: string
  phase?: string | undefined
  podName?: string | undefined
  gitHubBranch?: string | undefined
  pullRequestURL?: string | undefined
  pullRequestStatus?: string | undefined
}

function renderApp(path: string) {
  const history = createMemoryHistory({ initialEntries: [path] })
  const router = createAppRouter(history)
  const result = render(<RouterProvider router={router} />)
  return result
}

describe('workflow-ui-github', () => {
  it('stores token in session storage by default and local storage when remembered', async () => {
    localStorage.removeItem('kocao.apiToken')
    sessionStorage.removeItem('kocao.apiToken')

    server.use(
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] })),
      http.get('/api/v1/workspace-sessions', () => HttpResponse.json({ workspaceSessions: [] }))
    )

    const { unmount } = renderApp('/workspace-sessions')

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

    const workspaceSessions: WorkspaceSession[] = []
    const harnessRuns: HarnessRun[] = []
    let lastStartBody: any = null

    server.use(
      http.get('/api/v1/workspace-sessions', () => HttpResponse.json({ workspaceSessions })),
      http.post('/api/v1/workspace-sessions', async (ctx: any) => {
        const b = (await ctx.request.json()) as { repoURL?: string }
        const s: WorkspaceSession = { id: 'sess-1', repoURL: b.repoURL ?? '', phase: 'Active' }
        workspaceSessions.unshift(s)
        return HttpResponse.json(s, { status: 201 })
      }),
      http.get('/api/v1/workspace-sessions/:id', (ctx: any) => {
        const s = workspaceSessions.find((x) => x.id === ctx.params['id'])
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
        const r = harnessRuns.find((x) => x.id === ctx.params['id'])
        if (!r) return new HttpResponse('not found', { status: 404 })
        return HttpResponse.json(r)
      }),
      http.post('/api/v1/workspace-sessions/:id/harness-runs', async (ctx: any) => {
        const b = (await ctx.request.json()) as { repoURL: string; repoRevision?: string; image: string; args?: string[] }
        lastStartBody = b
        const r: HarnessRun = {
          id: 'run-1',
          workspaceSessionID: String(ctx.params['id']),
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

    const { unmount } = renderApp('/workspace-sessions')

    const create = await screen.findByRole('button', { name: 'Provision' })
    await userEvent.click(create)

    await screen.findByRole('heading', { name: /Workspace Session sess-1/ })

    // Ensure the start-run form is actionable even if session data is still loading.
    const repoInput = await screen.findByPlaceholderText('defaults to workspace session repoURL')
    await userEvent.clear(repoInput)
    await userEvent.type(repoInput, 'https://example.com/repo')

    const start = await screen.findByRole('button', { name: 'Start Harness Run' })
    await userEvent.click(start)

    await screen.findByRole('heading', { name: /Harness Run run-1/ })
    expect(lastStartBody?.args).toBeUndefined()
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

  it('maps Task to args when starting a run', async () => {
    sessionStorage.setItem('kocao.apiToken', 't-full')

    const harnessRuns: HarnessRun[] = []
    let lastStartBody: any = null

    server.use(
      http.get('/api/v1/workspace-sessions/sess-1', () =>
        HttpResponse.json({ id: 'sess-1', repoURL: 'https://example.com/repo', phase: 'Active' })
      ),
      http.get('/api/v1/harness-runs', () => HttpResponse.json({ harnessRuns })),
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] })),
      http.post('/api/v1/workspace-sessions/:id/harness-runs', async (ctx: any) => {
        const b = (await ctx.request.json()) as { repoURL: string; repoRevision?: string; image: string; args?: string[] }
        lastStartBody = b
        const r: HarnessRun = {
          id: 'run-task',
          workspaceSessionID: String(ctx.params['id']),
          repoURL: b.repoURL,
          repoRevision: b.repoRevision ?? 'main',
          image: b.image,
          phase: 'Running',
          podName: 'pod-task'
        }
        harnessRuns.unshift(r)
        return HttpResponse.json(r, { status: 201 })
      }),
      http.get('/api/v1/harness-runs/:id', () =>
        HttpResponse.json({ id: 'run-task', workspaceSessionID: 'sess-1', repoURL: 'https://example.com/repo', image: 'kocao/harness-runtime:dev', phase: 'Running' })
      )
    )

    const { unmount } = renderApp('/workspace-sessions/sess-1')

    const taskInput = await screen.findByPlaceholderText('make ci')
    await userEvent.type(taskInput, 'make ci')

    const start = await screen.findByRole('button', { name: 'Start Harness Run' })
    await userEvent.click(start)

    await screen.findByRole('heading', { name: /Harness Run run-task/ })
    expect(lastStartBody?.args).toEqual(['bash', '-lc', 'make ci'])

    unmount()
  })

  it('uses Advanced args when provided, even when Task is set', async () => {
    sessionStorage.setItem('kocao.apiToken', 't-full')

    const harnessRuns: HarnessRun[] = []
    let lastStartBody: any = null

    server.use(
      http.get('/api/v1/workspace-sessions/sess-1', () =>
        HttpResponse.json({ id: 'sess-1', repoURL: 'https://example.com/repo', phase: 'Active' })
      ),
      http.get('/api/v1/harness-runs', () => HttpResponse.json({ harnessRuns })),
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] })),
      http.post('/api/v1/workspace-sessions/:id/harness-runs', async (ctx: any) => {
        const b = (await ctx.request.json()) as { repoURL: string; repoRevision?: string; image: string; args?: string[] }
        lastStartBody = b
        const r: HarnessRun = {
          id: 'run-advanced',
          workspaceSessionID: String(ctx.params['id']),
          repoURL: b.repoURL,
          repoRevision: b.repoRevision ?? 'main',
          image: b.image,
          phase: 'Running',
          podName: 'pod-advanced'
        }
        harnessRuns.unshift(r)
        return HttpResponse.json(r, { status: 201 })
      }),
      http.get('/api/v1/harness-runs/:id', () =>
        HttpResponse.json({ id: 'run-advanced', workspaceSessionID: 'sess-1', repoURL: 'https://example.com/repo', image: 'kocao/harness-runtime:dev', phase: 'Running' })
      )
    )

    const { unmount } = renderApp('/workspace-sessions/sess-1')

    const taskInput = await screen.findByPlaceholderText('make ci')
    await userEvent.type(taskInput, 'make ci')

    const advancedInput = await screen.findByPlaceholderText('["go", "test", "./..."]')
    await userEvent.click(advancedInput)
    await userEvent.paste('["go", "test", "./..."]')

    const start = await screen.findByRole('button', { name: 'Start Harness Run' })
    await userEvent.click(start)

    await screen.findByRole('heading', { name: /Harness Run run-advanced/ })
    expect(lastStartBody?.args).toEqual(['go', 'test', './...'])

    unmount()
  })
})

describe('auth-failures', () => {
  it('clears token on 401 and prompts to re-enter credentials', async () => {
    sessionStorage.setItem('kocao.apiToken', 't-bad')
    localStorage.removeItem('kocao.apiToken')

    server.use(
      http.get('/api/v1/workspace-sessions', () => new HttpResponse('unauthorized', { status: 401 })),
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] }))
    )

    const { unmount } = renderApp('/workspace-sessions')

    await screen.findByText(/Bearer token rejected \(401\)/)
    expect(sessionStorage.getItem('kocao.apiToken')).toBeNull()
    expect(localStorage.getItem('kocao.apiToken')).toBeNull()

    await screen.findByText('No bearer token set. Auth required for API calls.')

    unmount()
  })
})

describe('attach-ui', () => {
  it('establishes websocket without URL token', async () => {
    sessionStorage.setItem('kocao.apiToken', 't-full')

    let cookieCalls = 0
    server.use(
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] })),
      http.get('/api/v1/harness-runs/:id', () => new HttpResponse('not found', { status: 404 })),
      http.post('/api/v1/workspace-sessions/:id/attach-cookie', (ctx: any) => {
        cookieCalls += 1
        return HttpResponse.json(
          {
            expiresAt: new Date().toISOString(),
            workspaceSessionID: String(ctx.params['id']),
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
      const { unmount } = renderApp('/workspace-sessions/sess-1/attach?role=viewer')

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
