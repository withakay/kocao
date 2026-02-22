import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../test/server'
import { App } from './App'

type Session = { id: string; repoURL?: string; phase?: string }
type Run = {
  id: string
  sessionID?: string
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
  it('creates a session, starts a run, and shows PR outcome metadata', async () => {
    localStorage.setItem('kocao.apiToken', 't-full')
    window.location.hash = '#/sessions'

    const sessions: Session[] = []
    const runs: Run[] = []

    server.use(
      http.get('/api/v1/sessions', () => HttpResponse.json({ sessions })),
      http.post('/api/v1/sessions', async (ctx: any) => {
        const b = (await ctx.request.json()) as { repoURL?: string }
        const s: Session = { id: 'sess-1', repoURL: b.repoURL ?? '', phase: 'Active' }
        sessions.unshift(s)
        return HttpResponse.json(s, { status: 201 })
      }),
      http.get('/api/v1/sessions/:id', (ctx: any) => {
        const s = sessions.find((x) => x.id === ctx.params.id)
        if (!s) return new HttpResponse('not found', { status: 404 })
        return HttpResponse.json(s)
      }),
      http.get('/api/v1/runs', (ctx: any) => {
        const url = new URL(ctx.request.url)
        const sid = url.searchParams.get('sessionID')
        const out = sid ? runs.filter((r) => r.sessionID === sid) : runs
        return HttpResponse.json({ runs: out })
      }),
      http.get('/api/v1/runs/:id', (ctx: any) => {
        const r = runs.find((x) => x.id === ctx.params.id)
        if (!r) return new HttpResponse('not found', { status: 404 })
        return HttpResponse.json(r)
      }),
      http.post('/api/v1/sessions/:id/runs', async (ctx: any) => {
        const b = (await ctx.request.json()) as { repoURL: string; repoRevision?: string; image: string }
        const r: Run = {
          id: 'run-1',
          sessionID: String(ctx.params.id),
          repoURL: b.repoURL,
          repoRevision: b.repoRevision ?? 'main',
          image: b.image,
          phase: 'Succeeded',
          podName: 'pod-123',
          gitHubBranch: 'feature/mvp-ui',
          pullRequestURL: 'https://github.com/withakay/kocao/pull/123',
          pullRequestStatus: 'merged'
        }
        runs.unshift(r)
        return HttpResponse.json(r, { status: 201 })
      }),
      http.get('/api/v1/audit', () => HttpResponse.json({ events: [] }))
    )

    render(<App />)

    const create = await screen.findByRole('button', { name: 'Create Session' })
    await userEvent.click(create)

    await screen.findByRole('heading', { name: /Session sess-1/ })

    // Ensure the start-run form is actionable even if session data is still loading.
    const repoInput = await screen.findByPlaceholderText('defaults to session repoURL')
    await userEvent.clear(repoInput)
    await userEvent.type(repoInput, 'https://example.com/repo')

    const start = await screen.findByRole('button', { name: 'Start Run' })
    await userEvent.click(start)

    await screen.findByRole('heading', { name: /Run run-1/ })
    await screen.findByText('Succeeded')

    const outcome = await screen.findByRole('heading', { name: 'GitHub Outcome' })
    const card = outcome.closest('section') ?? outcome.parentElement
    expect(card).toBeTruthy()

    const prLink = await within(card as HTMLElement).findByRole('link', { name: 'https://github.com/withakay/kocao/pull/123' })
    expect(prLink).toHaveAttribute('href', 'https://github.com/withakay/kocao/pull/123')
    within(card as HTMLElement).getByText('feature/mvp-ui')
    within(card as HTMLElement).getByText('merged')
  })
})
