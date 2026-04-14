import { useCallback } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { Topbar } from '../components/Topbar'
import { Btn, CollapsibleSection, DetailRow, ErrorBanner } from '../components/primitives'
import { formatTimestamp, TaskPreviewTable } from './remoteAgentDashboardShared'

export function RemoteAgentTaskArtifactsPage() {
  const { taskId } = useParams({ strict: false })
  const id = taskId ?? ''
  const { token, invalidateToken } = useAuth()

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in Settings.')
  }, [invalidateToken])

  const taskQ = usePollingQuery(`remote-agent-task:${id}:${token}`, () => api.getRemoteAgentTask(token, id), {
    intervalMs: 2500,
    enabled: token.trim() !== '' && id !== '',
    onUnauthorized,
  })
  const artifactsQ = usePollingQuery(`remote-agent-task-artifacts:${id}:${token}`, () => api.getRemoteAgentTaskArtifacts(token, id), {
    intervalMs: 2500,
    enabled: token.trim() !== '' && id !== '',
    onUnauthorized,
  })

  const task = taskQ.data

  return (
    <>
      <Topbar
        title={`Artifacts ${id}`}
        subtitle="Generated files, patches, bundles, and reports persisted with the task result."
        right={<Btn onClick={() => { taskQ.reload(); artifactsQ.reload() }} type="button">Refresh</Btn>}
      />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId" params={{ taskId: id }}>Back to task</Link>
          <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId/transcript" params={{ taskId: id }}>Open transcript</Link>
        </div>

        {artifactsQ.error ? <ErrorBanner>{artifactsQ.error}</ErrorBanner> : null}

        <CollapsibleSection title="Summary" defaultOpen={true}>
          <DetailRow label="Task ID">{id}</DetailRow>
          <DetailRow label="State">{task?.state || '—'}</DetailRow>
          <DetailRow label="Agent">{task?.agentName || '—'}</DetailRow>
          <DetailRow label="Output Count">{String(task?.result?.outputArtifactCount ?? artifactsQ.data?.outputArtifacts?.length ?? 0)}</DetailRow>
          <DetailRow label="Last Transition">{formatTimestamp(task?.lastTransitionAt)}</DetailRow>
        </CollapsibleSection>

        <CollapsibleSection title="Artifacts" defaultOpen={true}>
          <TaskPreviewTable artifacts={artifactsQ.data} />
        </CollapsibleSection>

        <CollapsibleSection title="Metadata" defaultOpen={false}>
          <DetailRow label="Outcome">{task?.result?.outcome || '—'}</DetailRow>
          <DetailRow label="Summary">{task?.result?.summary || '—'}</DetailRow>
        </CollapsibleSection>
      </div>
    </>
  )
}
