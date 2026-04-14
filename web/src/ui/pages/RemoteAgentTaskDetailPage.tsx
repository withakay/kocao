import { useCallback } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { Topbar } from '../components/Topbar'
import { Btn, ErrorBanner } from '../components/primitives'
import { formatTimestamp, TaskDetailSections } from './remoteAgentDashboardShared'

export function RemoteAgentTaskDetailPage() {
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
  const artifactsQ = usePollingQuery(`remote-agent-task-artifacts:${id}:${token}`, () => api.getRemoteAgentTaskArtifacts(token, id), {
    intervalMs: 2500,
    enabled: token.trim() !== '' && id !== '',
    onUnauthorized,
  })

  const task = taskQ.data

  return (
    <>
      <Topbar
        title={`Task ${task?.id ?? id}`}
        subtitle={task ? `Task detail, transcript preview, artifacts, and assignment context. Last transition ${formatTimestamp(task.lastTransitionAt)}.` : 'Task detail, transcript preview, artifacts, and assignment context.'}
        right={<Btn onClick={() => { taskQ.reload(); transcriptQ.reload(); artifactsQ.reload() }} type="button">Refresh</Btn>}
      />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <Link className="text-primary hover:underline" to="/remote-agents/tasks">Back to tasks</Link>
          {task?.agentId ? <Link className="text-primary hover:underline" to="/remote-agents/agents/$agentId" params={{ agentId: task.agentId }}>Assigned agent</Link> : null}
          <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId/transcript" params={{ taskId: id }}>Transcript</Link>
          <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId/artifacts" params={{ taskId: id }}>Artifacts</Link>
        </div>

        {taskQ.error ? <ErrorBanner>{taskQ.error}</ErrorBanner> : null}

        {task ? (
          <TaskDetailSections
            task={task}
            transcript={transcriptQ.data?.transcript ?? []}
            artifacts={artifactsQ.data}
          />
        ) : (
          <div className="rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-sm text-muted-foreground">
            {taskQ.loading ? 'Loading…' : 'Task not found.'}
          </div>
        )}
      </div>
    </>
  )
}
