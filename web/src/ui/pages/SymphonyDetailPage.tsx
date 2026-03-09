import { useCallback, useMemo, useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api, isUnauthorizedError, type SymphonyProjectClaim, type SymphonyProjectRequest, type SymphonyProjectRetry, type SymphonyProjectSkip } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { Badge, Btn, Card, CardHeader, EmptyRow, ErrorBanner, ScopeBadge, Table, Td, Th } from '../components/primitives'
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

function issueLabel(repository?: string, number?: number, title?: string): string {
  const repo = (repository ?? '').trim()
  const id = repo && number ? `${repo}#${number}` : repo || 'unknown issue'
  return title ? `${id} — ${title}` : id
}

function ClaimRows({ claims }: { claims: SymphonyProjectClaim[] }) {
  return (
    <Table label="active symphony claims table">
      <thead>
        <tr className="border-b border-border/40">
          <Th>Issue</Th>
          <Th>Attempt</Th>
          <Th>Phase</Th>
          <Th>Session</Th>
          <Th>Run</Th>
        </tr>
      </thead>
      <tbody>
        {claims.length === 0 ? (
          <EmptyRow cols={5} loading={false} message="No active claims." />
        ) : (
          claims.map((claim) => (
            <tr key={claim.itemId} className="border-b border-border/20 last:border-b-0">
              <Td className="font-mono text-xs">
                {claim.issue?.url ? (
                  <a className="text-primary hover:underline" href={claim.issue.url} target="_blank" rel="noreferrer">
                    {issueLabel(claim.issue.repository, claim.issue.number, claim.issue.title)}
                  </a>
                ) : (
                  issueLabel(claim.issue?.repository, claim.issue?.number, claim.issue?.title)
                )}
              </Td>
              <Td>{claim.attempt ?? 0}</Td>
              <Td><StatusPill phase={claim.phase} /></Td>
              <Td className="font-mono text-xs">
                {claim.runRef?.sessionName ? <Link className="text-primary hover:underline" to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: claim.runRef.sessionName }}>{claim.runRef.sessionName}</Link> : '—'}
              </Td>
              <Td className="font-mono text-xs">
                {claim.runRef?.harnessRunName ? <Link className="text-primary hover:underline" to="/harness-runs/$harnessRunID" params={{ harnessRunID: claim.runRef.harnessRunName }}>{claim.runRef.harnessRunName}</Link> : '—'}
              </Td>
            </tr>
          ))
        )}
      </tbody>
    </Table>
  )
}

function RetryRows({ retries }: { retries: SymphonyProjectRetry[] }) {
  return (
    <Table label="symphony retry queue table">
      <thead>
        <tr className="border-b border-border/40">
          <Th>Issue</Th>
          <Th>Attempt</Th>
          <Th>Reason</Th>
          <Th>Ready At</Th>
        </tr>
      </thead>
      <tbody>
        {retries.length === 0 ? (
          <EmptyRow cols={4} loading={false} message="Retry queue is empty." />
        ) : (
          retries.map((retry) => (
            <tr key={retry.itemId} className="border-b border-border/20 last:border-b-0">
              <Td className="font-mono text-xs">{issueLabel(retry.issue?.repository, retry.issue?.number, retry.issue?.title)}</Td>
              <Td>{retry.attempt ?? 0}</Td>
              <Td>{retry.reason || '—'}</Td>
              <Td className="text-xs text-muted-foreground">{formatTimestamp(retry.readyAt)}</Td>
            </tr>
          ))
        )}
      </tbody>
    </Table>
  )
}

function SkipRows({ skips }: { skips: SymphonyProjectSkip[] }) {
  return (
    <Table label="recent symphony skips table">
      <thead>
        <tr className="border-b border-border/40">
          <Th>Issue</Th>
          <Th>Reason</Th>
          <Th>Message</Th>
          <Th>Observed</Th>
        </tr>
      </thead>
      <tbody>
        {skips.length === 0 ? (
          <EmptyRow cols={4} loading={false} message="No recent skips." />
        ) : (
          skips.map((skip) => (
            <tr key={skip.itemId} className="border-b border-border/20 last:border-b-0">
              <Td className="font-mono text-xs">{issueLabel(skip.issue?.repository ?? skip.repository, skip.issue?.number, skip.issue?.title)}</Td>
              <Td>{skip.reason || '—'}</Td>
              <Td>{skip.message || '—'}</Td>
              <Td className="text-xs text-muted-foreground">{formatTimestamp(skip.observedTime)}</Td>
            </tr>
          ))
        )}
      </tbody>
    </Table>
  )
}

export function SymphonyDetailPage() {
  const { token, invalidateToken } = useAuth()
  const { projectName } = useParams({ strict: false })
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [action, setAction] = useState<'pause' | 'resume' | 'refresh' | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in Settings.')
  }, [invalidateToken])

  const q = usePollingQuery(`symphony-project:${token}:${projectName}`, () => api.getSymphonyProject(token, projectName), {
    intervalMs: 5000,
    enabled: token.trim() !== '' && typeof projectName === 'string' && projectName.trim() !== '',
    onUnauthorized,
  })

  const project = q.data
  const stats = useMemo(
    () => [
      { label: 'Eligible', value: String(project?.status.eligibleItems ?? 0) },
      { label: 'Running', value: String(project?.status.runningItems ?? 0) },
      { label: 'Retrying', value: String(project?.status.retryingItems ?? 0) },
      { label: 'Completed', value: String(project?.status.completedItems ?? 0) },
      { label: 'Skipped', value: String(project?.status.skippedItems ?? 0) },
    ],
    [project],
  )

  const saveProject = useCallback(
    async (request: SymphonyProjectRequest) => {
      if (!projectName) return
      setSaving(true)
      setSaveError(null)
      try {
        await api.updateSymphonyProject(token, projectName, request)
        await q.reload()
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
    [onUnauthorized, projectName, q, token],
  )

  const runAction = useCallback(
    async (nextAction: 'pause' | 'resume' | 'refresh') => {
      if (!projectName) return
      setAction(nextAction)
      setActionError(null)
      try {
        if (nextAction === 'pause') await api.pauseSymphonyProject(token, projectName)
        if (nextAction === 'resume') await api.resumeSymphonyProject(token, projectName)
        if (nextAction === 'refresh') await api.refreshSymphonyProject(token, projectName)
        await q.reload()
      } catch (error) {
        if (isUnauthorizedError(error)) {
          onUnauthorized()
          return
        }
        setActionError(error instanceof Error ? error.message : String(error))
      } finally {
        setAction(null)
      }
    },
    [onUnauthorized, projectName, q, token],
  )

  return (
    <>
      <Topbar title={`Symphony ${projectName ?? ''}`} subtitle="Queue detail, runtime state, retries, recent skips, and live controls." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <Card>
          <CardHeader
            title="Runtime"
            right={(
              <div className="flex items-center gap-2">
                <ScopeBadge scope="symphony-project:control" />
                <Btn type="button" onClick={() => runAction('refresh')} disabled={action !== null || !project}>{action === 'refresh' ? 'Refreshing…' : 'Refresh'}</Btn>
                {project?.paused ? (
                  <Btn type="button" variant="primary" onClick={() => runAction('resume')} disabled={action !== null}>{action === 'resume' ? 'Resuming…' : 'Resume'}</Btn>
                ) : (
                  <Btn type="button" variant="danger" onClick={() => runAction('pause')} disabled={action !== null || !project}>{action === 'pause' ? 'Pausing…' : 'Pause'}</Btn>
                )}
              </div>
            )}
          />
          {project ? (
            <>
              <div className="flex flex-wrap items-center gap-2 mb-3">
                <StatusPill phase={project.status.phase} />
                {project.paused ? <Badge variant="warn">paused</Badge> : <Badge variant="ok">active</Badge>}
                <Badge variant="neutral">board {project.spec.source.project.owner}/{project.spec.source.project.number}</Badge>
                <Badge variant="info">field {project.status.resolvedFieldName || project.spec.source.fieldName || 'Status'}</Badge>
              </div>
              <div className="grid gap-2 md:grid-cols-5">
                {stats.map((stat) => (
                  <div key={stat.label} className="rounded-md border border-border/60 bg-muted/20 px-3 py-2">
                    <div className="text-[10px] uppercase tracking-wider text-muted-foreground">{stat.label}</div>
                    <div className="text-lg font-mono text-foreground">{stat.value}</div>
                  </div>
                ))}
              </div>
              <div className="mt-3 grid gap-2 md:grid-cols-2 text-xs text-muted-foreground">
                <div>Last sync: <span className="font-mono text-foreground/80">{formatTimestamp(project.status.lastSyncTime)}</span></div>
                <div>Next sync: <span className="font-mono text-foreground/80">{formatTimestamp(project.status.nextSyncTime)}</span></div>
              </div>
              {project.status.lastError ? <ErrorBanner>{project.status.lastError}</ErrorBanner> : null}
              {project.status.unsupportedRepositories?.length ? (
                <div className="mt-3 rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-xs">
                  <div className="mb-1 font-medium uppercase tracking-wider text-muted-foreground">Unsupported Repositories</div>
                  <div className="flex flex-wrap gap-1">
                    {project.status.unsupportedRepositories.map((repo) => <Badge key={repo} variant="warn">{repo}</Badge>)}
                  </div>
                </div>
              ) : null}
            </>
          ) : (
            <div className="text-sm text-muted-foreground">{q.loading ? 'Loading Symphony project…' : 'Symphony project not found.'}</div>
          )}
          {actionError ? <ErrorBanner>{actionError}</ErrorBanner> : null}
          {q.error ? <ErrorBanner>{q.error}</ErrorBanner> : null}
        </Card>

        <Card>
          <CardHeader title="Edit Configuration" right={<ScopeBadge scope="symphony-project:write" />} />
          {project ? <SymphonyProjectForm initialProject={project} submitLabel="Save Changes" busy={saving} error={saveError} onSubmit={saveProject} /> : null}
        </Card>

        <Card>
          <CardHeader title="Active Claims" />
          <ClaimRows claims={project?.status.activeClaims ?? []} />
        </Card>

        <Card>
          <CardHeader title="Retry Queue" />
          <RetryRows retries={project?.status.retryQueue ?? []} />
        </Card>

        <Card>
          <CardHeader title="Recent Skips" />
          <SkipRows skips={project?.status.recentSkips ?? []} />
        </Card>
      </div>
    </>
  )
}
