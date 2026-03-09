import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api, isUnauthorizedError, type SymphonyProjectRequest } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { Badge, Card, CardHeader, EmptyRow, ErrorBanner, ScopeBadge, Table, Td, Th } from '../components/primitives'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'
import { SymphonyProjectForm } from '../components/SymphonyProjectForm'

function formatTimestamp(value?: string): string {
  const raw = (value ?? '').trim()
  if (!raw) return '—'
  const parsed = Date.parse(raw)
  if (!Number.isFinite(parsed)) return raw
  return new Date(parsed).toLocaleString()
}

export function SymphonyPage() {
  const { token, invalidateToken } = useAuth()
  const navigate = useNavigate()
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in Settings.')
  }, [invalidateToken])

  const q = usePollingQuery(`symphony-projects:${token}`, () => api.listSymphonyProjects(token), {
    intervalMs: 5000,
    enabled: token.trim() !== '',
    onUnauthorized,
  })

  const projects = useMemo(() => (q.data?.symphonyProjects ?? []).slice().sort((left, right) => left.name.localeCompare(right.name)), [q.data])

  const createProject = useCallback(
    async (request: SymphonyProjectRequest) => {
      setSaving(true)
      setSaveError(null)
      try {
        const created = await api.createSymphonyProject(token, request)
        await navigate({ to: '/symphony/$projectName', params: { projectName: created.name } })
      } catch (error) {
        if (isUnauthorizedError(error)) {
          onUnauthorized()
          return
        }
        setSaveError(error instanceof Error ? error.message : String(error))
      } finally {
        setSaving(false)
      }
    },
    [navigate, onUnauthorized, token],
  )

  return (
    <>
      <Topbar title="Symphony" subtitle="GitHub Projects-backed orchestration queues, runtime state, and operator controls." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <Card>
          <CardHeader title="Create Symphony Project" right={<ScopeBadge scope="symphony-project:write" />} />
          <SymphonyProjectForm submitLabel="Create Project" busy={saving} error={saveError} onSubmit={createProject} />
          {token.trim() === '' ? (
            <ErrorBanner>
              No bearer token set. Open <Link className="underline" to="/settings">Settings</Link> to configure auth.
            </ErrorBanner>
          ) : null}
        </Card>

        <Card>
          <CardHeader
            title="Configured Queues"
            right={(
              <div className="flex items-center gap-2">
                <Badge variant="info">polling 5s</Badge>
                <Badge variant="neutral">GitHub Projects v2</Badge>
              </div>
            )}
          />
          <Table label="symphony projects table">
            <thead>
              <tr className="border-b border-border/40">
                <Th>Name</Th>
                <Th>Board</Th>
                <Th>Repos</Th>
                <Th>Phase</Th>
                <Th>Active</Th>
                <Th>Retry</Th>
                <Th>Next Sync</Th>
              </tr>
            </thead>
            <tbody>
              {projects.length === 0 ? (
                <EmptyRow cols={7} loading={q.loading} message="No Symphony projects configured." />
              ) : (
                projects.map((project) => (
                  <tr key={project.name} className="border-b border-border/20 last:border-b-0 hover:bg-muted/30 transition-colors">
                    <Td className="font-mono">
                      <Link className="text-primary hover:underline" to="/symphony/$projectName" params={{ projectName: project.name }}>
                        {project.name}
                      </Link>
                    </Td>
                    <Td className="font-mono text-muted-foreground">
                      {project.spec.source.project.owner}/{project.spec.source.project.number}
                    </Td>
                    <Td>{project.spec.repositories.length}</Td>
                    <Td>
                      <div className="flex items-center gap-2">
                        <StatusPill phase={project.status.phase} />
                        {project.paused ? <Badge variant="warn">paused</Badge> : null}
                      </div>
                    </Td>
                    <Td>{project.status.runningItems ?? 0}</Td>
                    <Td>{project.status.retryingItems ?? 0}</Td>
                    <Td className="text-xs text-muted-foreground">{formatTimestamp(project.status.nextSyncTime)}</Td>
                  </tr>
                ))
              )}
            </tbody>
          </Table>
          {q.error ? <ErrorBanner>{q.error}</ErrorBanner> : null}
        </Card>
      </div>
    </>
  )
}
