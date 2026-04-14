import { useCallback } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { Topbar } from '../components/Topbar'
import { Btn, ErrorBanner } from '../components/primitives'
import { AgentDetailSections, formatTimestamp } from './remoteAgentDashboardShared'

export function RemoteAgentDetailPage() {
  const { agentId } = useParams({ strict: false })
  const id = agentId ?? ''
  const { token, invalidateToken } = useAuth()

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in Settings.')
  }, [invalidateToken])

  const agentQ = usePollingQuery(`remote-agent:${id}:${token}`, () => api.getRemoteAgent(token, id), {
    intervalMs: 5000,
    enabled: token.trim() !== '' && id !== '',
    onUnauthorized,
  })
  const taskId = agentQ.data?.currentTaskId ?? ''
  const taskQ = usePollingQuery(`remote-agent-current-task:${taskId}:${token}`, () => api.getRemoteAgentTask(token, taskId), {
    intervalMs: 2500,
    enabled: token.trim() !== '' && taskId !== '',
    onUnauthorized,
  })

  const agent = agentQ.data

  return (
    <>
      <Topbar
        title={`Agent ${agent?.displayName || agent?.name || id}`}
        subtitle={agent ? `Current pool ${agent.poolName || 'unassigned'} with last activity ${formatTimestamp(agent.lastActivityAt)}.` : 'Agent assignment, session binding, and current task context.'}
        right={<Btn onClick={() => { agentQ.reload(); taskQ.reload() }} type="button">Refresh</Btn>}
      />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <Link className="text-primary hover:underline" to="/remote-agents/agents">Back to agents</Link>
          {agent?.currentTaskId ? <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId" params={{ taskId: agent.currentTaskId }}>Open current task</Link> : null}
        </div>

        {agentQ.error ? <ErrorBanner>{agentQ.error}</ErrorBanner> : null}

        {agent ? (
          <AgentDetailSections agent={agent} task={taskQ.data} />
        ) : (
          <div className="rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-sm text-muted-foreground">
            {agentQ.loading ? 'Loading…' : 'Agent not found.'}
          </div>
        )}
      </div>
    </>
  )
}
