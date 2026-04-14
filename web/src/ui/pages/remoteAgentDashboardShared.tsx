import { Link } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'
import type {
  RemoteAgent,
  RemoteAgentArtifactRef,
  RemoteAgentTask,
  RemoteAgentTaskArtifactsResponse,
  RemoteAgentTaskDetail,
  RemoteAgentTaskState,
  RemoteAgentTranscriptEntry,
} from '../lib/api'
import { remoteAgentTaskStateGroups, summarizeTranscript } from '../lib/remoteAgentDashboard'
import { Badge, Card, CardHeader, CollapsibleSection, DetailRow, EmptyRow, Table, Td, Th, btnClass } from '../components/primitives'

export function formatTimestamp(value?: string): string {
  const raw = (value ?? '').trim()
  if (!raw) return '—'
  const parsed = Date.parse(raw)
  if (!Number.isFinite(parsed)) return raw
  return new Date(parsed).toLocaleString()
}

export function formatBytes(value?: number): string {
  if (!value || value <= 0) return '—'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / (1024 * 1024)).toFixed(1)} MB`
}

export function isRecent(value?: string, windowHours = 24): boolean {
  const raw = (value ?? '').trim()
  if (!raw) return false
  const parsed = Date.parse(raw)
  if (!Number.isFinite(parsed)) return false
  return Date.now() - parsed <= windowHours * 60 * 60 * 1000
}

export function RemoteAgentAvailabilityBadge({ availability }: { availability?: string }) {
  const value = (availability ?? 'unknown').trim().toLowerCase()
  const variant = value === 'busy' ? 'info' : value === 'idle' ? 'ok' : value === 'offline' ? 'neutral' : 'warn'
  return <Badge variant={variant}>{value || 'unknown'}</Badge>
}

export function RemoteAgentTaskStateBadge({ state }: { state: RemoteAgentTaskState }) {
  const variant =
    state === 'running' ? 'info' :
    state === 'assigned' ? 'warn' :
    state === 'completed' ? 'ok' :
    state === 'failed' || state === 'timed_out' || state === 'cancelled' ? 'bad' :
    'neutral'
  return <Badge variant={variant}>{state}</Badge>
}

export function RemoteAgentArtifactKindBadge({ kind }: { kind: string }) {
  const variant = kind === 'report' ? 'info' : kind === 'patch' ? 'warn' : kind === 'bundle' ? 'ok' : 'neutral'
  return <Badge variant={variant}>{kind}</Badge>
}

export function useSelectableList<T extends { id: string }>(items: T[]) {
  const [selectedId, setSelectedId] = useState('')

  useEffect(() => {
    if (items.length === 0) {
      setSelectedId('')
      return
    }
    if (!items.some((item) => item.id === selectedId)) {
      setSelectedId(items[0]?.id ?? '')
    }
  }, [items, selectedId])

  const selected = useMemo(() => items.find((item) => item.id === selectedId) ?? null, [items, selectedId])

  return { selectedId, setSelectedId, selected }
}

export function RemoteAgentOverviewCards({ agents, tasks }: { agents: RemoteAgent[]; tasks: RemoteAgentTask[] }) {
  const activeTasks = tasks.filter((task) => remoteAgentTaskStateGroups.active.includes(task.state)).length
  const activeAgents = agents.filter((agent) => (agent.availability ?? '').toLowerCase() !== 'offline').length
  const terminalTasks24h = tasks.filter((task) => remoteAgentTaskStateGroups.terminal.includes(task.state) && isRecent(task.lastTransitionAt)).length
  const artifacts24h = tasks.reduce((count, task) => {
    if (!isRecent(task.lastTransitionAt)) return count
    return count + (task.result?.outputArtifactCount ?? 0)
  }, 0)

  const cards = [
    { label: 'Active Agents', value: String(activeAgents), to: '/remote-agents/agents' as const },
    { label: 'Active Tasks', value: String(activeTasks), to: '/remote-agents/tasks' as const },
    { label: 'Terminal Tasks (24h)', value: String(terminalTasks24h), to: '/remote-agents/tasks' as const },
    { label: 'Artifacts (24h)', value: String(artifacts24h), to: '/remote-agents/tasks' as const },
  ]

  return (
    <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
      {cards.map((card) => (
        <Link
          key={card.label}
          className="rounded-lg border border-border/60 bg-card px-3 py-2 transition-colors hover:bg-muted/30"
          to={card.to}
        >
          <div className="text-[10px] uppercase tracking-[0.18em] text-muted-foreground">{card.label}</div>
          <div className="mt-1 text-lg font-semibold text-foreground">{card.value}</div>
        </Link>
      ))}
    </div>
  )
}

export function TaskDetailSections({ task, transcript, artifacts }: {
  task: RemoteAgentTaskDetail
  transcript: RemoteAgentTranscriptEntry[]
  artifacts: RemoteAgentTaskArtifactsResponse | null
}) {
  const inputArtifacts = artifacts?.inputArtifacts ?? []
  const outputArtifacts = artifacts?.outputArtifacts ?? []
  const artifactList = [...inputArtifacts, ...outputArtifacts]

  return (
    <div className="space-y-3">
      <CollapsibleSection title="Summary" defaultOpen={true}>
        <DetailRow label="Task ID">{task.id}</DetailRow>
        <DetailRow label="State"><RemoteAgentTaskStateBadge state={task.state} /></DetailRow>
        <DetailRow label="Outcome">{task.result?.outcome || '—'}</DetailRow>
        <DetailRow label="Summary">{task.result?.summary || '—'}</DetailRow>
        <DetailRow label="Prompt">{task.prompt || '—'}</DetailRow>
      </CollapsibleSection>

      <CollapsibleSection title="Assignment" defaultOpen={true}>
        <DetailRow label="Agent">
          {task.agentId ? <Link className="text-primary hover:underline" to="/remote-agents/agents/$agentId" params={{ agentId: task.agentId }}>{task.agentName || task.agentId}</Link> : (task.agentName || '—')}
        </DetailRow>
        <DetailRow label="Pool">{task.poolName || '—'}</DetailRow>
        <DetailRow label="Requested By">{task.requestedBy || '—'}</DetailRow>
        <DetailRow label="Attempt">{String(task.attempt ?? 0)}</DetailRow>
        <DetailRow label="Retries">{String(task.retryCount ?? 0)}</DetailRow>
      </CollapsibleSection>

      <CollapsibleSection title="Session" defaultOpen={true}>
        <DetailRow label="Workspace">{task.workspaceSessionId || '—'}</DetailRow>
        <DetailRow label="Harness Run">{task.currentSession?.harnessRunId || '—'}</DetailRow>
        <DetailRow label="Session ID">{task.currentSession?.sessionId || '—'}</DetailRow>
        <DetailRow label="Pod">{task.currentSession?.podName || '—'}</DetailRow>
        <DetailRow label="Runtime">{task.currentSession?.runtime || '—'}</DetailRow>
      </CollapsibleSection>

      <CollapsibleSection title="Timeline" defaultOpen={true}>
        <DetailRow label="Created">{formatTimestamp(task.createdAt)}</DetailRow>
        <DetailRow label="Assigned">{formatTimestamp(task.assignedAt)}</DetailRow>
        <DetailRow label="Started">{formatTimestamp(task.startedAt)}</DetailRow>
        <DetailRow label="Completed">{formatTimestamp(task.completedAt)}</DetailRow>
        <DetailRow label="Cancelled">{formatTimestamp(task.cancelledAt)}</DetailRow>
        <DetailRow label="Last Transition">{formatTimestamp(task.lastTransitionAt)}</DetailRow>
      </CollapsibleSection>

      <CollapsibleSection
        title="Transcript"
        defaultOpen={true}
        headerRight={(
          <Link className={btnClass('ghost')} to="/remote-agents/tasks/$taskId/transcript" params={{ taskId: task.id }}>
            Open transcript
          </Link>
        )}
      >
        {transcript.length === 0 ? (
          <div className="rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-sm text-muted-foreground">No transcript entries were persisted for this task.</div>
        ) : (
          <div className="space-y-2">
            {transcript.slice(0, 5).map((entry) => (
              <div key={entry.sequence} className="rounded-md border border-border/60 bg-muted/20 px-3 py-2">
                <div className="text-[10px] font-mono uppercase tracking-[0.18em] text-muted-foreground">#{entry.sequence} · {entry.role} · {entry.kind || 'message'}</div>
                <div className="mt-1 text-sm text-foreground">{summarizeTranscript(entry)}</div>
              </div>
            ))}
          </div>
        )}
      </CollapsibleSection>

      <CollapsibleSection
        title="Artifacts"
        defaultOpen={true}
        headerRight={(
          <Link className={btnClass('ghost')} to="/remote-agents/tasks/$taskId/artifacts" params={{ taskId: task.id }}>
            Open artifacts
          </Link>
        )}
      >
        {artifactList.length === 0 ? (
          <div className="rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-sm text-muted-foreground">No task artifacts are available yet.</div>
        ) : (
          <div className="space-y-2">
            {artifactList.slice(0, 5).map((artifact) => (
              <div key={artifact.id} className="rounded-md border border-border/60 bg-muted/20 px-3 py-2">
                <div className="flex items-center gap-2">
                  <div className="min-w-0 flex-1 truncate font-mono text-sm text-foreground">{artifact.name}</div>
                  <RemoteAgentArtifactKindBadge kind={artifact.kind} />
                </div>
                <div className="mt-1 text-xs text-muted-foreground">{artifact.mediaType || 'unknown media type'} · {formatBytes(artifact.sizeBytes)}</div>
              </div>
            ))}
          </div>
        )}
      </CollapsibleSection>

      <CollapsibleSection title="Metadata" defaultOpen={false}>
        <DetailRow label="Transcript Rows">{String(task.result?.transcriptEntries ?? transcript.length)}</DetailRow>
        <DetailRow label="Artifacts">{String(task.result?.outputArtifactCount ?? outputArtifacts.length)}</DetailRow>
        <DetailRow label="Timeout">{task.timeoutSeconds ? `${task.timeoutSeconds}s` : '—'}</DetailRow>
      </CollapsibleSection>
    </div>
  )
}

export function AgentDetailSections({ agent, task }: { agent: RemoteAgent; task?: RemoteAgentTaskDetail | null }) {
  return (
    <div className="space-y-3">
      <CollapsibleSection title="Summary" defaultOpen={true}>
        <DetailRow label="Agent">{agent.displayName || agent.name}</DetailRow>
        <DetailRow label="Availability"><RemoteAgentAvailabilityBadge availability={agent.availability} /></DetailRow>
        <DetailRow label="Pool">{agent.poolName || '—'}</DetailRow>
        <DetailRow label="Description">{agent.description || '—'}</DetailRow>
      </CollapsibleSection>

      <CollapsibleSection title="Assignment" defaultOpen={true}>
        <DetailRow label="Current Task">
          {agent.currentTaskId ? <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId" params={{ taskId: agent.currentTaskId }}>{task?.id || agent.currentTaskId}</Link> : '—'}
        </DetailRow>
        <DetailRow label="Task State">{task ? <RemoteAgentTaskStateBadge state={task.state} /> : '—'}</DetailRow>
        <DetailRow label="Requested By">{task?.requestedBy || '—'}</DetailRow>
        <DetailRow label="Workspace">{agent.workspaceSessionId || task?.workspaceSessionId || '—'}</DetailRow>
      </CollapsibleSection>

      <CollapsibleSection title="Session" defaultOpen={true}>
        <DetailRow label="Harness Run">{agent.currentSession?.harnessRunId || '—'}</DetailRow>
        <DetailRow label="Session ID">{agent.currentSession?.sessionId || '—'}</DetailRow>
        <DetailRow label="Pod">{agent.currentSession?.podName || '—'}</DetailRow>
        <DetailRow label="Runtime">{agent.currentSession?.runtime || agent.runtime || '—'}</DetailRow>
        <DetailRow label="Agent Type">{agent.currentSession?.agent || agent.agent || '—'}</DetailRow>
      </CollapsibleSection>

      <CollapsibleSection title="Timeline" defaultOpen={true}>
        <DetailRow label="Last Activity">{formatTimestamp(agent.lastActivityAt)}</DetailRow>
        <DetailRow label="Created">{formatTimestamp(agent.createdAt)}</DetailRow>
        <DetailRow label="Updated">{formatTimestamp(agent.updatedAt)}</DetailRow>
      </CollapsibleSection>

      <CollapsibleSection title="Metadata" defaultOpen={false}>
        <DetailRow label="Agent ID">{agent.id}</DetailRow>
        <DetailRow label="Pool ID">{agent.poolId || '—'}</DetailRow>
      </CollapsibleSection>
    </div>
  )
}

export function TaskPreviewTable({ artifacts }: { artifacts: RemoteAgentTaskArtifactsResponse | null }) {
  const rows: Array<{ scope: string; artifact: RemoteAgentArtifactRef }> = [
    ...(artifacts?.inputArtifacts ?? []).map((artifact) => ({ scope: 'input', artifact })),
    ...(artifacts?.outputArtifacts ?? []).map((artifact) => ({ scope: 'output', artifact })),
  ]

  return (
    <Table label="artifact preview table">
      <thead>
        <tr className="border-b border-border/40">
          <Th>Scope</Th>
          <Th>Artifact</Th>
          <Th>Kind</Th>
          <Th>Size</Th>
          <Th>Created</Th>
        </tr>
      </thead>
      <tbody>
        {rows.length === 0 ? (
          <EmptyRow cols={5} loading={false} message="No task artifacts are available yet." />
        ) : (
          rows.map(({ scope, artifact }) => (
            <tr key={`${scope}:${artifact.id}`} className="border-b border-border/20 last:border-b-0">
              <Td className="font-mono text-xs text-muted-foreground">{scope}</Td>
              <Td className="font-mono">{artifact.name}</Td>
              <Td><RemoteAgentArtifactKindBadge kind={artifact.kind} /></Td>
              <Td className="font-mono text-xs">{formatBytes(artifact.sizeBytes)}</Td>
              <Td className="text-xs text-muted-foreground">{formatTimestamp(artifact.createdAt)}</Td>
            </tr>
          ))
        )}
      </tbody>
    </Table>
  )
}

export function TaskTranscriptTable({ taskId, transcript }: { taskId: string; transcript: RemoteAgentTranscriptEntry[] }) {
  return (
    <Table label="transcript table">
      <thead>
        <tr className="border-b border-border/40">
          <Th>Sequence</Th>
          <Th>At</Th>
          <Th>Role</Th>
          <Th>Kind</Th>
          <Th>Event Ref</Th>
          <Th>Summary</Th>
        </tr>
      </thead>
      <tbody>
        {transcript.length === 0 ? (
          <EmptyRow cols={6} loading={false} message="No transcript entries were persisted for this task." />
        ) : (
          transcript.map((entry) => (
            <tr key={`${taskId}:${entry.sequence}`} className="border-b border-border/20 last:border-b-0">
              <Td className="font-mono">{entry.sequence}</Td>
              <Td className="text-xs text-muted-foreground">{formatTimestamp(entry.at)}</Td>
              <Td><Badge variant="neutral">{entry.role}</Badge></Td>
              <Td className="font-mono text-xs">{entry.kind || 'message'}</Td>
              <Td className="font-mono text-xs text-muted-foreground">{entry.eventRef || '—'}</Td>
              <Td className="text-sm">{summarizeTranscript(entry)}</Td>
            </tr>
          ))
        )}
      </tbody>
    </Table>
  )
}

export function ResourcePanel({ title, description, children, right }: { title: string; description?: string; children: React.ReactNode; right?: React.ReactNode }) {
  return (
    <Card className="h-full">
      <CardHeader title={title} right={right} />
      {description ? <p className="mb-3 text-xs text-muted-foreground">{description}</p> : null}
      {children}
    </Card>
  )
}
