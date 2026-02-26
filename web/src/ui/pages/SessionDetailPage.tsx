import { useCallback, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { StatusPill } from '../components/StatusPill'
import { Topbar } from '../components/Topbar'
import {
  Btn, CollapsibleSection, DetailRow, ErrorBanner, FormRow,
  Input, Select, ScopeBadge, Table, Td, Textarea, Th, EmptyRow,
} from '../components/primitives'

export function SessionDetailPage() {
  const { workspaceSessionID } = useParams()
  const id = workspaceSessionID ?? ''
  const { token, invalidateToken } = useAuth()
  const nav = useNavigate()

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const sessQ = usePollingQuery(
    `workspace-session:${id}:${token}`,
    () => api.getWorkspaceSession(token, id),
    { intervalMs: 2500, enabled: token.trim() !== '' && id !== '', onUnauthorized }
  )
  const runsQ = usePollingQuery(
    `harness-runs:${id}:${token}`,
    () => api.listHarnessRuns(token, id),
    { intervalMs: 2500, enabled: token.trim() !== '' && id !== '', onUnauthorized }
  )
  const auditQ = usePollingQuery(
    `audit:${token}`,
    () => api.listAudit(token, 200),
    { intervalMs: 3000, enabled: token.trim() !== '', onUnauthorized }
  )

  const runs = useMemo(() => (runsQ.data?.harnessRuns ?? []).slice().sort((a, b) => b.id.localeCompare(a.id)), [runsQ.data])
  const events = useMemo(() => {
    const evs = auditQ.data?.events ?? []
    return evs.filter((e) => e.resourceID === id).slice(-30)
  }, [auditQ.data, id])

  const [repoURL, setRepoURL] = useState('')
  const [repoRevision, setRepoRevision] = useState('')
  const [task, setTask] = useState('')
  const [advancedArgs, setAdvancedArgs] = useState('')
  const [image, setImage] = useState('kocao/harness-runtime:dev')
  const [egressMode, setEgressMode] = useState<'restricted' | 'full'>('restricted')
  const [starting, setStarting] = useState(false)
  const [startErr, setStartErr] = useState<string | null>(null)

  const start = useCallback(async () => {
    setStarting(true)
    setStartErr(null)
    try {
      const trimmedTask = task.trim()
      const trimmedAdvancedArgs = advancedArgs.trim()
      let args: string[] | undefined
      if (trimmedAdvancedArgs !== '') {
        let parsed: unknown
        try { parsed = JSON.parse(trimmedAdvancedArgs) } catch {
          throw new Error('Advanced args must be valid JSON (array of strings).')
        }
        if (!Array.isArray(parsed) || parsed.some((v) => typeof v !== 'string')) {
          throw new Error('Advanced args must be a JSON array of strings.')
        }
        args = parsed as string[]
      } else if (trimmedTask !== '') {
        args = ['bash', '-lc', trimmedTask]
      }

      const out = await api.startHarnessRun(token, id, {
        repoURL: repoURL.trim() !== '' ? repoURL.trim() : sessQ.data?.repoURL ?? '',
        repoRevision: repoRevision.trim() !== '' ? repoRevision.trim() : undefined,
        image: image.trim(),
        egressMode,
        args,
      })
      nav(`/harness-runs/${encodeURIComponent(out.id)}`)
    } catch (e) {
      if (isUnauthorizedError(e)) { onUnauthorized(); return }
      setStartErr(e instanceof Error ? e.message : String(e))
    } finally {
      setStarting(false)
    }
  }, [token, id, repoURL, repoRevision, task, advancedArgs, image, egressMode, nav, sessQ.data, onUnauthorized])

  const sess = sessQ.data
  const effectiveRepo = repoURL.trim() !== '' ? repoURL.trim() : sess?.repoURL ?? ''

  return (
    <>
      <Topbar title={`Session ${id}`} subtitle="Session context, harness run dispatch, and audit trail." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        {/* Session Info */}
        <CollapsibleSection
          title="Session Info"
          persistKey="kocao.section.session.info"
          defaultOpen={true}
          headerRight={<Btn onClick={() => { sessQ.reload(); runsQ.reload() }} type="button">Refresh</Btn>}
        >
          {sess ? (
            <>
              <DetailRow label="Repo">{sess.repoURL && sess.repoURL.trim() !== '' ? sess.repoURL : '\u2014'}</DetailRow>
              <DetailRow label="Phase"><StatusPill phase={sess.phase} /></DetailRow>
            </>
          ) : (
            <div className="text-xs text-muted-foreground">{sessQ.loading ? 'Loading\u2026' : sessQ.error ?? 'No data.'}</div>
          )}
          {sessQ.error ? <ErrorBanner>{sessQ.error}</ErrorBanner> : null}
        </CollapsibleSection>

        {/* Start Harness Run */}
        <CollapsibleSection
          title="Start Harness Run"
          persistKey="kocao.section.session.start"
          defaultOpen={true}
          headerRight={<ScopeBadge scope="harness-run:write" />}
        >
          <FormRow label="Repo URL">
            <Input value={effectiveRepo} onChange={(e) => setRepoURL(e.target.value)} placeholder="defaults to workspace session repoURL" />
          </FormRow>
          <FormRow label="Revision">
            <Input value={repoRevision} onChange={(e) => setRepoRevision(e.target.value)} placeholder="main (optional)" />
          </FormRow>
          <FormRow label="Task" hint={<>Executed as <code className="font-mono text-foreground/60">bash -lc "&lt;task&gt;"</code>. Never embed secrets here.</>}>
            <Textarea rows={2} value={task} onChange={(e) => setTask(e.target.value)} placeholder="make ci" />
          </FormRow>
          <FormRow label="Args (JSON)" hint="Overrides task field when set.">
            <Textarea rows={2} value={advancedArgs} onChange={(e) => setAdvancedArgs(e.target.value)} placeholder='["go", "test", "./..."]' />
          </FormRow>
          <FormRow label="Image">
            <Input value={image} onChange={(e) => setImage(e.target.value)} placeholder="kocao/harness-runtime:dev" />
          </FormRow>
          <FormRow label="Egress">
            <Select value={egressMode} onChange={(e) => setEgressMode(e.target.value as 'restricted' | 'full')}>
              <option value="restricted">restricted (GitHub-only)</option>
              <option value="full">full (internet)</option>
            </Select>
          </FormRow>
          <div className="flex items-center gap-2 mt-1 pl-27">
            <Btn
              variant="primary"
              disabled={starting || token.trim() === '' || effectiveRepo.trim() === ''}
              onClick={start}
              type="button"
            >
              {starting ? 'Starting\u2026' : 'Start Harness Run'}
            </Btn>
          </div>
          {startErr ? <ErrorBanner>{startErr}</ErrorBanner> : null}
        </CollapsibleSection>

        {/* Runs */}
        <CollapsibleSection
          title="Runs"
          persistKey="kocao.section.session.runs"
          defaultOpen={true}
          headerRight={<span className="text-[10px] text-muted-foreground/50 font-mono">polling 2.5s</span>}
        >
          <Table label="harness runs table">
            <thead>
              <tr className="border-b border-border/40">
                <Th>ID</Th>
                <Th>Repo</Th>
                <Th className="w-28">Phase</Th>
              </tr>
            </thead>
            <tbody>
              {runs.length === 0 ? (
                <EmptyRow cols={3} loading={runsQ.loading} message="No harness runs for this workspace session." />
              ) : (
                runs.map((r) => (
                  <tr key={r.id} className="border-b border-border/20 last:border-b-0 hover:bg-muted/30 transition-colors">
                    <Td className="font-mono">
                      <Link to={`/harness-runs/${encodeURIComponent(r.id)}`} className="text-primary hover:underline">{r.id}</Link>
                    </Td>
                    <Td className="font-mono text-muted-foreground">{r.repoURL}</Td>
                    <Td><StatusPill phase={r.phase} /></Td>
                  </tr>
                ))
              )}
            </tbody>
          </Table>
          {runsQ.error ? <ErrorBanner>{runsQ.error}</ErrorBanner> : null}
        </CollapsibleSection>

        {/* Audit Trail */}
        <CollapsibleSection
          title="Audit Trail"
          persistKey="kocao.section.session.audit"
          defaultOpen={true}
          headerRight={<span className="text-[10px] text-muted-foreground/50 font-mono">source: /api/v1/audit</span>}
        >
          {events.length === 0 ? (
            <div className="text-xs text-muted-foreground">{auditQ.loading ? 'Loading\u2026' : 'No audit events for this session.'}</div>
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
