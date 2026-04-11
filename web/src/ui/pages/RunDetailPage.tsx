import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'
import {
  Btn, btnClass, CollapsibleSection, DetailRow, ErrorBanner,
  FormRow, ScopeBadge, Table, Td, Textarea, Th, EmptyRow,
} from '../components/primitives'

export function RunDetailPage() {
  const { harnessRunID } = useParams({ strict: false })
  const id = harnessRunID ?? ''
  const { token, invalidateToken } = useAuth()
  const nav = useNavigate()
  const [actionErr, setActionErr] = useState<string | null>(null)
  const [acting, setActing] = useState(false)
  const [agentPrompt, setAgentPrompt] = useState('')
  const [agentErr, setAgentErr] = useState<string | null>(null)
  const [agentActing, setAgentActing] = useState(false)

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in Settings.')
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
  const agentSessionQ = usePollingQuery(
    `agent-session:${id}:${token}`,
    () => api.getAgentSession(token, id),
    { intervalMs: 1500, enabled: token.trim() !== '' && id !== '' && Boolean(runQ.data?.agentSession?.agent), onUnauthorized }
  )
  const agentEventsQ = usePollingQuery(
    `agent-session-events:${id}:${token}`,
    () => api.listAgentSessionEvents(token, id, { limit: 200 }),
    { intervalMs: 1500, enabled: token.trim() !== '' && id !== '' && Boolean(runQ.data?.agentSession?.agent), onUnauthorized }
  )

  const run = runQ.data
  const events = useMemo(() => {
    const evs = auditQ.data?.events ?? []
    return evs.filter((e) => e.resourceID === id).slice(-40)
  }, [auditQ.data, id])
  const agentEvents = useMemo(() => agentEventsQ.data?.events ?? [], [agentEventsQ.data])

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
      nav({ to: '/harness-runs/$harnessRunID', params: { harnessRunID: out.id } })
    } catch (e) {
      if (isUnauthorizedError(e)) { onUnauthorized(); return }
      setActionErr(e instanceof Error ? e.message : String(e))
    } finally {
      setActing(false)
    }
  }, [token, id, nav, onUnauthorized])

  const startAgentSession = useCallback(async () => {
    setAgentActing(true)
    setAgentErr(null)
    try {
      await api.createAgentSession(token, id)
      agentSessionQ.reload()
      agentEventsQ.reload()
    } catch (e) {
      if (isUnauthorizedError(e)) { onUnauthorized(); return }
      setAgentErr(e instanceof Error ? e.message : String(e))
    } finally {
      setAgentActing(false)
    }
  }, [token, id, agentSessionQ, agentEventsQ, onUnauthorized])

  const promptAgentSession = useCallback(async () => {
    if (agentPrompt.trim() === '') return
    setAgentActing(true)
    setAgentErr(null)
    try {
      await api.promptAgentSession(token, id, agentPrompt.trim())
      setAgentPrompt('')
      agentSessionQ.reload()
      agentEventsQ.reload()
    } catch (e) {
      if (isUnauthorizedError(e)) { onUnauthorized(); return }
      setAgentErr(e instanceof Error ? e.message : String(e))
    } finally {
      setAgentActing(false)
    }
  }, [token, id, agentPrompt, agentSessionQ, agentEventsQ, onUnauthorized])

  const stopAgentSession = useCallback(async () => {
    setAgentActing(true)
    setAgentErr(null)
    try {
      await api.stopAgentSession(token, id)
      agentSessionQ.reload()
      agentEventsQ.reload()
    } catch (e) {
      if (isUnauthorizedError(e)) { onUnauthorized(); return }
      setAgentErr(e instanceof Error ? e.message : String(e))
    } finally {
      setAgentActing(false)
    }
  }, [token, id, agentSessionQ, agentEventsQ, onUnauthorized])

  const attachLinks = run?.workspaceSessionID
    ? {
        viewer: {
          to: '/workspace-sessions/$workspaceSessionID/attach' as const,
          params: { workspaceSessionID: run.workspaceSessionID },
          search: { role: 'viewer' as const },
        },
        driver: {
          to: '/workspace-sessions/$workspaceSessionID/attach' as const,
          params: { workspaceSessionID: run.workspaceSessionID },
          search: { role: 'driver' as const },
        },
        collabDriver: {
          to: '/workspace-sessions/$workspaceSessionID/attach' as const,
          params: { workspaceSessionID: run.workspaceSessionID },
          search: { role: 'driver' as const, mode: 'collab' as const },
        },
      }
    : null

  return (
    <>
      <Topbar title={`Run ${run?.displayName ?? id}`} subtitle="Run lifecycle, attach entry points, and GitHub outcome." />

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
                    <Link to="/workspace-sessions/$workspaceSessionID" params={{ workspaceSessionID: run.workspaceSessionID }} className="text-primary hover:underline">{run.workspaceSessionID}</Link>
                ) : '\u2014'}
              </DetailRow>
              {run.displayName ? <DetailRow label="Name">{run.displayName}</DetailRow> : null}
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

        {run?.agentSession?.agent ? (
          <CollapsibleSection title="Agent Session" persistKey="kocao.section.run.agent-session" defaultOpen={true} headerRight={<span className="text-[10px] text-muted-foreground/50 font-mono">sandbox-agent</span>}>
            <DetailRow label="Runtime">{agentSessionQ.data?.runtime ?? run.agentSession.runtime ?? '\u2014'}</DetailRow>
            <DetailRow label="Agent">{agentSessionQ.data?.agent ?? run.agentSession.agent ?? '\u2014'}</DetailRow>
            <DetailRow label="Session ID">{agentSessionQ.data?.sessionId ?? run.agentSession.sessionId ?? '\u2014'}</DetailRow>
            <DetailRow label="Phase"><StatusPill phase={agentSessionQ.data?.phase ?? run.agentSession.phase} /></DetailRow>
            <div className="flex items-center gap-2 mt-3 mb-2">
              <Btn disabled={agentActing || token.trim() === ""} onClick={startAgentSession} type="button">
                {agentActing ? "Working…" : "Start / Resume Agent Session"}
              </Btn>
              <Btn variant="danger" disabled={agentActing || token.trim() === ""} onClick={stopAgentSession} type="button">
                Stop Agent Session
              </Btn>
            </div>
            <FormRow label="Prompt">
              <Textarea rows={3} value={agentPrompt} onChange={(e) => setAgentPrompt(e.target.value)} placeholder="Ask the agent to inspect or modify the repository…" />
            </FormRow>
            <div className="flex items-center gap-2 pl-27">
              <Btn variant="primary" disabled={agentActing || token.trim() === "" || agentPrompt.trim() === ""} onClick={promptAgentSession} type="button">
                Send Prompt
              </Btn>
            </div>
            {agentErr ? <ErrorBanner>{agentErr}</ErrorBanner> : null}
            <div className="mt-4 space-y-2">
              <div className="text-xs font-medium text-foreground/80">Transcript / Events</div>
              {agentEvents.length === 0 ? (
                <div className="text-xs text-muted-foreground">{agentEventsQ.loading ? "Loading…" : "No agent session events yet."}</div>
              ) : (
                <div className="rounded-md border border-border/50 bg-muted/20 p-3 space-y-2 max-h-96 overflow-y-auto">
                  {agentEvents.map((event) => (
                    <div key={event.sequence} className="rounded border border-border/30 bg-background/60 p-2">
                      <div className="text-[10px] font-mono text-muted-foreground mb-1">#{event.sequence} · {new Date(event.at).toLocaleTimeString()}</div>
                      <pre className="whitespace-pre-wrap break-words text-[11px] leading-relaxed text-foreground/90">{JSON.stringify(event.envelope, null, 2)}</pre>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </CollapsibleSection>
        ) : null}

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
                <Link className={btnClass('secondary')} to={attachLinks.viewer.to} params={attachLinks.viewer.params} search={attachLinks.viewer.search}>Open Viewer</Link>
                <Link className={btnClass('primary')} to={attachLinks.driver.to} params={attachLinks.driver.params} search={attachLinks.driver.search}>Open Driver</Link>
                <Link className={btnClass('secondary')} to={attachLinks.collabDriver.to} params={attachLinks.collabDriver.params} search={attachLinks.collabDriver.search}>Open Collab Driver</Link>
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
