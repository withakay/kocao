import { useCallback, useMemo, useState } from 'react'
import { useAuth } from '../auth'
import { api, isUnauthorizedError } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { Topbar } from '../components/Topbar'
import { Badge, Btn, Card, CardHeader, EmptyRow, ErrorBanner, FormRow, Input, Table, Td, Th } from '../components/primitives'

const DEFAULT_TAIL_LINES = 200

function ageLabel(ageSeconds: number): string {
  if (ageSeconds < 60) return `${ageSeconds}s`
  if (ageSeconds < 3600) return `${Math.floor(ageSeconds / 60)}m`
  if (ageSeconds < 86400) return `${Math.floor(ageSeconds / 3600)}h`
  return `${Math.floor(ageSeconds / 86400)}d`
}

function podPhaseVariant(phase: string): 'ok' | 'warn' | 'bad' | 'neutral' {
  const p = phase.toLowerCase()
  if (p === 'running') return 'ok'
  if (p === 'pending') return 'warn'
  if (p === 'failed') return 'bad'
  return 'neutral'
}

export function ClusterPage() {
  const { token, invalidateToken } = useAuth()
  const [selectedPod, setSelectedPod] = useState('')
  const [container, setContainer] = useState('')
  const [tailLines, setTailLines] = useState(String(DEFAULT_TAIL_LINES))
  const [logs, setLogs] = useState('')
  const [logsErr, setLogsErr] = useState<string | null>(null)
  const [loadingLogs, setLoadingLogs] = useState(false)

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in the top bar.')
  }, [invalidateToken])

  const q = usePollingQuery(
    `cluster-overview:${token}`,
    () => api.getClusterOverview(token),
    { intervalMs: 5000, enabled: token.trim() !== '', onUnauthorized },
  )

  const pods = q.data?.pods ?? []
  const deployments = q.data?.deployments ?? []

  const summaryCards = useMemo(() => {
    const s = q.data?.summary
    if (!s) return []
    return [
      { label: 'Sessions', value: String(s.sessionCount) },
      { label: 'Harness Runs', value: String(s.harnessRunCount) },
      { label: 'Pods', value: String(s.podCount) },
      { label: 'Running', value: String(s.runningPods) },
      { label: 'Pending', value: String(s.pendingPods) },
      { label: 'Failed', value: String(s.failedPods) },
    ]
  }, [q.data])

  const loadLogs = useCallback(async () => {
    if (selectedPod.trim() === '') {
      setLogsErr('Select a pod to inspect logs.')
      return
    }
    const parsedTail = Number(tailLines)
    if (!Number.isFinite(parsedTail) || parsedTail <= 0) {
      setLogsErr('Tail lines must be a positive number.')
      return
    }

    setLoadingLogs(true)
    setLogsErr(null)
    try {
      const out = await api.getPodLogs(token, selectedPod, {
        container: container.trim() === '' ? undefined : container.trim(),
        tailLines: Math.min(2000, Math.max(1, Math.floor(parsedTail))),
      })
      setLogs(out.logs)
    } catch (e) {
      if (isUnauthorizedError(e)) {
        onUnauthorized()
        return
      }
      setLogsErr(e instanceof Error ? e.message : String(e))
    } finally {
      setLoadingLogs(false)
    }
  }, [selectedPod, tailLines, token, container, onUnauthorized])

  return (
    <>
      <Topbar title="Cluster" subtitle="Namespace health, pod status, deployment state, and log inspection." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <Card>
          <CardHeader title="Namespace Summary" right={<Btn onClick={q.reload} type="button">Refresh</Btn>} />
          {q.data ? (
            <>
              <div className="text-xs text-muted-foreground mb-2">
                namespace: <span className="font-mono text-foreground/80">{q.data.namespace}</span>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-2">
                {summaryCards.map((c) => (
                  <div key={c.label} className="rounded-md border border-border/60 bg-muted/20 px-2 py-1.5">
                    <div className="text-[10px] uppercase tracking-wider text-muted-foreground">{c.label}</div>
                    <div className="text-sm font-mono text-foreground">{c.value}</div>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className="text-xs text-muted-foreground">{q.loading ? 'Loading…' : 'No cluster data.'}</div>
          )}
          {q.error ? <ErrorBanner>{q.error}</ErrorBanner> : null}
          {token.trim() === '' ? <ErrorBanner>No bearer token set. Auth required for API calls.</ErrorBanner> : null}
        </Card>

        <Card>
          <CardHeader title="Runtime Config (Non-Secret Indicators)" />
          {q.data ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-xs">
              <div className="rounded-md border border-border/60 bg-muted/20 px-2 py-1.5">
                <div className="text-muted-foreground">Environment</div>
                <div className="font-mono">{q.data.config.environment || 'unknown'}</div>
              </div>
              <div className="rounded-md border border-border/60 bg-muted/20 px-2 py-1.5">
                <div className="text-muted-foreground">Audit Path Configured</div>
                <Badge variant={q.data.config.auditPathConfigured ? 'ok' : 'warn'}>
                  {q.data.config.auditPathConfigured ? 'yes' : 'no'}
                </Badge>
              </div>
              <div className="rounded-md border border-border/60 bg-muted/20 px-2 py-1.5">
                <div className="text-muted-foreground">Bootstrap Token Detected</div>
                <Badge variant={q.data.config.bootstrapTokenDetected ? 'warn' : 'ok'}>
                  {q.data.config.bootstrapTokenDetected ? 'yes' : 'no'}
                </Badge>
              </div>
              <div className="rounded-md border border-border/60 bg-muted/20 px-2 py-1.5">
                <div className="text-muted-foreground">GitHub CIDRs Configured</div>
                <Badge variant={q.data.config.gitHubCIDRsConfigured ? 'ok' : 'neutral'}>
                  {q.data.config.gitHubCIDRsConfigured ? 'yes' : 'no'}
                </Badge>
              </div>
            </div>
          ) : (
            <div className="text-xs text-muted-foreground">No config data.</div>
          )}
        </Card>

        <Card>
          <CardHeader title="Deployments" />
          <Table label="deployments table">
            <thead>
              <tr className="border-b border-border/40">
                <Th>Name</Th>
                <Th>Ready</Th>
                <Th>Available</Th>
                <Th>Desired</Th>
                <Th>Updated</Th>
                <Th>Unavailable</Th>
              </tr>
            </thead>
            <tbody>
              {deployments.length === 0 ? (
                <EmptyRow cols={6} loading={q.loading} message="No deployments found." />
              ) : (
                deployments.map((d) => (
                  <tr key={d.name} className="border-b border-border/20 last:border-b-0 hover:bg-muted/30 transition-colors">
                    <Td className="font-mono">{d.name}</Td>
                    <Td>{d.readyReplicas}</Td>
                    <Td>{d.availableReplicas}</Td>
                    <Td>{d.desiredReplicas}</Td>
                    <Td>{d.updatedReplicas}</Td>
                    <Td>{d.unavailableReplicas}</Td>
                  </tr>
                ))
              )}
            </tbody>
          </Table>
        </Card>

        <Card>
          <CardHeader title="Pods" />
          <Table label="pods table">
            <thead>
              <tr className="border-b border-border/40">
                <Th>Name</Th>
                <Th>Phase</Th>
                <Th>Ready</Th>
                <Th>Restarts</Th>
                <Th>Node</Th>
                <Th>Age</Th>
              </tr>
            </thead>
            <tbody>
              {pods.length === 0 ? (
                <EmptyRow cols={6} loading={q.loading} message="No pods found." />
              ) : (
                pods.map((p) => (
                  <tr key={p.name} className="border-b border-border/20 last:border-b-0 hover:bg-muted/30 transition-colors">
                    <Td className="font-mono">{p.name}</Td>
                    <Td><Badge variant={podPhaseVariant(p.phase)}>{p.phase}</Badge></Td>
                    <Td className="font-mono">{p.ready}</Td>
                    <Td className="font-mono">{p.restarts}</Td>
                    <Td className="font-mono text-xs text-muted-foreground">{p.nodeName || '—'}</Td>
                    <Td className="font-mono">{ageLabel(p.ageSeconds)}</Td>
                  </tr>
                ))
              )}
            </tbody>
          </Table>
        </Card>

        <Card>
          <CardHeader title="Pod Logs" right={<Btn onClick={loadLogs} disabled={loadingLogs || token.trim() === ''} type="button">{loadingLogs ? 'Loading…' : 'Load Logs'}</Btn>} />
          <FormRow label="Pod">
            <Input value={selectedPod} onChange={(e) => setSelectedPod(e.target.value)} placeholder="pod name" list="kocao-pod-list" />
            <datalist id="kocao-pod-list">
              {pods.map((p) => <option key={p.name} value={p.name} />)}
            </datalist>
          </FormRow>
          <FormRow label="Container">
            <Input value={container} onChange={(e) => setContainer(e.target.value)} placeholder="optional (defaults to first container)" />
          </FormRow>
          <FormRow label="Tail Lines">
            <Input value={tailLines} onChange={(e) => setTailLines(e.target.value)} placeholder="200" />
          </FormRow>

          <div className="mt-2 rounded-md border border-border/60 bg-background/70 p-2">
            <pre className="text-[11px] leading-relaxed whitespace-pre-wrap break-words font-mono max-h-72 overflow-y-auto">
              {logs.trim() === '' ? 'No logs loaded.' : logs}
            </pre>
          </div>
          {logsErr ? <ErrorBanner>{logsErr}</ErrorBanner> : null}
        </Card>
      </div>
    </>
  )
}
