import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'
import {
  Btn, btnClass, CollapsibleSection, DetailRow, ErrorBanner,
  ScopeBadge, Table, Td, Th, EmptyRow,
} from '../components/primitives'

export function RunDetailPage() {
  const { harnessRunID } = useParams()
  const id = harnessRunID ?? ''
  const { token, invalidateToken } = useAuth()
  const nav = useNavigate()
  const [actionErr, setActionErr] = useState<string | null>(null)
  const [acting, setActing] = useState(false)

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const runQ = usePollingQuery(
    `harness-run:${id}:${token}`,
    () => api.getHarnessRun(token, id),
    { intervalMs: 2000, enabled: token.trim() !== '' && id !== '', onUnauthorized }
  )
  const auditQ = usePollingQuery(
    `audit:${token}`,
    () => api.listAudit(token, 250),
    { intervalMs: 3000, enabled: token.trim() !== '', onUnauthorized }
  )

  const run = runQ.data
  const events = useMemo(() => {
    const evs = auditQ.data?.events ?? []
    return evs.filter((e) => e.resourceID === id).slice(-40)
  }, [auditQ.data, id])

  const stop = useCallback(async () => {
    setActing(true)
    setActionErr(null)
    try {
      await api.stopHarnessRun(token, id)
      runQ.reload()
    } catch (e) {
      if (isUnauthorizedError(e)) { onUnauthorized(); return }
      setActionErr(e instanceof Error ? e.message : String(e))
    } finally {
      setActing(false)
    }
  }, [token, id, runQ, onUnauthorized])

  const resume = useCallback(async () => {
    setActing(true)
    setActionErr(null)
    try {
      const out = await api.resumeHarnessRun(token, id)
      nav(`/harness-runs/${encodeURIComponent(out.id)}`)
    } catch (e) {
      if (isUnauthorizedError(e)) { onUnauthorized(); return }
      setActionErr(e instanceof Error ? e.message : String(e))
    } finally {
      setActing(false)
    }
  }, [token, id, nav, onUnauthorized])

  const attachLinks = run?.workspaceSessionID
    ? {
        viewer: `/workspace-sessions/${encodeURIComponent(run.workspaceSessionID)}/attach?role=viewer`,
        driver: `/workspace-sessions/${encodeURIComponent(run.workspaceSessionID)}/attach?role=driver`,
      }
    : null

  return (
    <>
      <Topbar title={`Run ${id}`} subtitle="Run lifecycle, attach entry points, and GitHub outcome." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {/* Run Info */}
        <CollapsibleSection
          title="Run Info"
          persistKey="kocao.section.run.info"
          defaultOpen={true}
          headerRight={<Btn onClick={runQ.reload} type="button">Refresh</Btn>}
        >
          {run ? (
            <>
              <DetailRow label="Session">
                {run.workspaceSessionID ? (
                  <Link to={`/workspace-sessions/${encodeURIComponent(run.workspaceSessionID)}`} className="text-primary hover:underline">{run.workspaceSessionID}</Link>
                ) : '\u2014'}
              </DetailRow>
              <DetailRow label="Repo">{run.repoURL}</DetailRow>
              <DetailRow label="Revision">{run.repoRevision && run.repoRevision.trim() !== '' ? run.repoRevision : '\u2014'}</DetailRow>
              <DetailRow label="Image">{run.image}</DetailRow>
              <DetailRow label="Phase"><StatusPill phase={run.phase} /></DetailRow>
              <DetailRow label="Pod">{run.podName && run.podName.trim() !== '' ? run.podName : '\u2014'}</DetailRow>
            </>
          ) : (
            <div className="text-xs text-muted-foreground">{runQ.loading ? 'Loading\u2026' : runQ.error ?? 'No data.'}</div>
          )}
          {runQ.error ? <ErrorBanner>{runQ.error}</ErrorBanner> : null}
        </CollapsibleSection>

        {/* Actions */}
        <CollapsibleSection
          title="Actions"
          persistKey="kocao.section.run.actions"
          defaultOpen={true}
          headerRight={<ScopeBadge scope="harness-run:write" />}
        >
          <div className="flex items-center gap-2">
            <Btn variant="danger" disabled={acting || token.trim() === ''} onClick={stop} type="button">
              Stop
            </Btn>
            <Btn disabled={acting || token.trim() === ''} onClick={resume} type="button">
              Resume
            </Btn>
          </div>
          {actionErr ? <ErrorBanner>{actionErr}</ErrorBanner> : null}
        </CollapsibleSection>

        {/* Attach */}
        <CollapsibleSection
          title="Attach"
          persistKey="kocao.section.run.attach"
          defaultOpen={true}
          headerRight={<span className="text-[10px] text-muted-foreground/50 font-mono">websocket</span>}
        >
          {attachLinks ? (
            <>
              <p className="text-xs text-muted-foreground mb-2">Attach tokens are ephemeral. Opens a websocket terminal session.</p>
              <div className="flex items-center gap-2">
                <Link className={btnClass('secondary')} to={attachLinks.viewer}>Open Viewer</Link>
                <Link className={btnClass('primary')} to={attachLinks.driver}>Open Driver</Link>
              </div>
            </>
          ) : (
            <p className="text-xs text-muted-foreground">Not associated with a workspace session.</p>
          )}
        </CollapsibleSection>

        {/* GitHub Outcome */}
        {run?.pullRequestURL && run.pullRequestURL.trim() !== '' ? (
          <CollapsibleSection
            title="GitHub Outcome"
            persistKey="kocao.section.run.github"
            defaultOpen={true}
            headerRight={<span className="text-[10px] text-muted-foreground/50 font-mono">run metadata</span>}
          >
            <DetailRow label="Branch">{run.gitHubBranch && run.gitHubBranch.trim() !== '' ? run.gitHubBranch : '\u2014'}</DetailRow>
            <DetailRow label="PR URL">
              <a href={run.pullRequestURL} target="_blank" rel="noreferrer" className="text-primary hover:underline">
                {run.pullRequestURL}
              </a>
            </DetailRow>
            <DetailRow label="PR Status">{run.pullRequestStatus && run.pullRequestStatus.trim() !== '' ? run.pullRequestStatus : '\u2014'}</DetailRow>
          </CollapsibleSection>
        ) : null}

        {/* Audit Trail */}
        <CollapsibleSection
          title="Audit Trail"
          persistKey="kocao.section.run.audit"
          defaultOpen={true}
          headerRight={<span className="text-[10px] text-muted-foreground/50 font-mono">source: /api/v1/audit</span>}
        >
          {events.length === 0 ? (
            <div className="text-xs text-muted-foreground">{auditQ.loading ? 'Loading\u2026' : 'No audit events for this run.'}</div>
          ) : (
            <Table label="audit table">
              <thead>
                <tr className="border-b border-border/40">
                  <Th>At</Th>
                  <Th>Action</Th>
                  <Th>Outcome</Th>
                </tr>
              </thead>
              <tbody>
                {events.map((e) => (
                  <tr key={e.id} className="border-b border-border/20 last:border-b-0">
                    <Td className="font-mono">{new Date(e.at).toLocaleTimeString()}</Td>
                    <Td className="font-mono">{e.action}</Td>
                    <Td className="font-mono">{e.outcome}</Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          )}
        </CollapsibleSection>
      </div>
    </>
  )
}
