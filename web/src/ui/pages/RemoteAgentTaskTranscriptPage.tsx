import { useCallback } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { Topbar } from '../components/Topbar'
import { Btn, CollapsibleSection, DetailRow, ErrorBanner } from '../components/primitives'
import { formatTimestamp, TaskTranscriptTable } from './remoteAgentDashboardShared'

export function RemoteAgentTaskTranscriptPage() {
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
  const transcriptQ = usePollingQuery(`remote-agent-task-transcript:${id}:${token}`, () => api.getRemoteAgentTaskTranscript(token, id), {
    intervalMs: 2500,
    enabled: token.trim() !== '' && id !== '',
    onUnauthorized,
  })

  const task = taskQ.data
  const transcript = transcriptQ.data?.transcript ?? []

  return (
    <>
      <Topbar
        title={`Transcript ${id}`}
        subtitle="Persisted task conversation and tool history anchored to the durable task id."
        right={<Btn onClick={() => { taskQ.reload(); transcriptQ.reload() }} type="button">Refresh</Btn>}
      />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId" params={{ taskId: id }}>Back to task</Link>
          <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId/artifacts" params={{ taskId: id }}>Open artifacts</Link>
        </div>

        {transcriptQ.error ? <ErrorBanner>{transcriptQ.error}</ErrorBanner> : null}

        <CollapsibleSection title="Summary" defaultOpen={true}>
          <DetailRow label="Task ID">{id}</DetailRow>
          <DetailRow label="State">{task?.state || '—'}</DetailRow>
          <DetailRow label="Agent">{task?.agentName || '—'}</DetailRow>
          <DetailRow label="Transcript Rows">{String(task?.result?.transcriptEntries ?? transcript.length)}</DetailRow>
          <DetailRow label="Last Transition">{formatTimestamp(task?.lastTransitionAt)}</DetailRow>
        </CollapsibleSection>

        <CollapsibleSection title="Transcript" defaultOpen={true}>
          <TaskTranscriptTable taskId={id} transcript={transcript} />
        </CollapsibleSection>

        <CollapsibleSection title="Metadata" defaultOpen={false}>
          <DetailRow label="Prompt">{task?.prompt || '—'}</DetailRow>
          <DetailRow label="Outcome">{task?.result?.outcome || '—'}</DetailRow>
        </CollapsibleSection>
      </div>
    </>
  )
}
