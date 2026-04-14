import { useCallback, useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { Topbar } from '../components/Topbar'
import { Btn, EmptyRow, ErrorBanner, FormRow, Input, Select, Table, Td, Th } from '../components/primitives'
import {
  AgentDetailSections,
  RemoteAgentAvailabilityBadge,
  RemoteAgentOverviewCards,
  ResourcePanel,
  useSelectableList,
} from './remoteAgentDashboardShared'
import { remoteAgentDashboardInformationArchitecture } from '../lib/remoteAgentDashboard'

export function RemoteAgentListPage() {
  const { token, invalidateToken } = useAuth()
  const [filter, setFilter] = useState('')
  const [poolFilter, setPoolFilter] = useState('all')

  const onUnauthorized = useCallback(() => {
    invalidateToken('Bearer token rejected (401). Please re-enter a valid token in Settings.')
  }, [invalidateToken])

  const agentsQ = usePollingQuery(`remote-agents:${token}`, () => api.listRemoteAgents(token), {
    intervalMs: 5000,
    enabled: token.trim() !== '',
    onUnauthorized,
  })
  const poolsQ = usePollingQuery(`remote-agent-pools:${token}`, () => api.listRemoteAgentPools(token), {
    intervalMs: 10000,
    enabled: token.trim() !== '',
    onUnauthorized,
  })
  const tasksQ = usePollingQuery(`remote-agent-tasks:${token}`, () => api.listRemoteAgentTasks(token), {
    intervalMs: 2500,
    enabled: token.trim() !== '',
    onUnauthorized,
  })

  const agents = useMemo(() => {
    const all = (agentsQ.data?.remoteAgents ?? []).slice().sort((left, right) => {
      return (Date.parse(right.lastActivityAt ?? '') || 0) - (Date.parse(left.lastActivityAt ?? '') || 0)
    })
    const normalizedFilter = filter.trim().toLowerCase()
    return all.filter((agent) => {
      const matchesFilter = normalizedFilter === '' || [
        agent.id,
        agent.name,
        agent.displayName,
        agent.currentTaskId,
        agent.poolName,
      ].some((value) => (value ?? '').toLowerCase().includes(normalizedFilter))
      const matchesPool = poolFilter === 'all' || (agent.poolName ?? '') === poolFilter
      return matchesFilter && matchesPool
    })
  }, [agentsQ.data, filter, poolFilter])

  const pools = useMemo(() => {
    const names = new Set<string>()
    for (const pool of poolsQ.data?.remoteAgentPools ?? []) {
      if ((pool.name ?? '').trim() !== '') names.add(pool.name)
    }
    for (const agent of agentsQ.data?.remoteAgents ?? []) {
      if ((agent.poolName ?? '').trim() !== '') names.add(agent.poolName ?? '')
    }
    return Array.from(names).sort((left, right) => left.localeCompare(right))
  }, [agentsQ.data, poolsQ.data])

  const { selectedId, setSelectedId, selected } = useSelectableList(agents)
  const selectedAgentQ = usePollingQuery(`remote-agent:${selectedId}:${token}`, () => api.getRemoteAgent(token, selectedId), {
    intervalMs: 5000,
    enabled: token.trim() !== '' && selectedId !== '',
    onUnauthorized,
  })
  const currentTaskId = selectedAgentQ.data?.currentTaskId ?? selected?.currentTaskId ?? ''
  const currentTaskQ = usePollingQuery(`remote-agent-current-task:${currentTaskId}:${token}`, () => api.getRemoteAgentTask(token, currentTaskId), {
    intervalMs: 2500,
    enabled: token.trim() !== '' && currentTaskId !== '',
    onUnauthorized,
  })

  return (
    <>
      <Topbar title="Remote Agents" subtitle="Named workers grouped by pool, current task, and session binding." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <RemoteAgentOverviewCards agents={agentsQ.data?.remoteAgents ?? []} tasks={tasksQ.data?.remoteAgentTasks ?? []} />

        <ResourcePanel
          title="Active Remote Agents"
          description={remoteAgentDashboardInformationArchitecture.collections.find((collection) => collection.resource === 'agents')?.description}
          right={<Btn onClick={() => { agentsQ.reload(); tasksQ.reload() }} type="button">Refresh</Btn>}
        >
          <div className="space-y-3">
            <div className="grid gap-2 md:grid-cols-2">
              <FormRow label="Search">
                <Input value={filter} onChange={(event) => setFilter(event.target.value)} placeholder="agent, pool, task" />
              </FormRow>
              <FormRow label="Pool">
                <Select value={poolFilter} onChange={(event) => setPoolFilter(event.target.value)}>
                  <option value="all">All pools</option>
                  {pools.map((pool) => <option key={pool} value={pool}>{pool}</option>)}
                </Select>
              </FormRow>
            </div>

            {agentsQ.error ? <ErrorBanner>{agentsQ.error}</ErrorBanner> : null}

            <div className="grid gap-3 xl:grid-cols-[minmax(0,1.05fr)_minmax(22rem,0.95fr)]">
              <div className="rounded-lg border border-border/60 bg-background/40 p-2">
                <Table label="remote agents table">
                  <thead>
                    <tr className="border-b border-border/40">
                      <Th>Agent</Th>
                      <Th>Pool</Th>
                      <Th>Availability</Th>
                      <Th>Current Task</Th>
                      <Th>Last Activity</Th>
                      <Th>Workspace Session</Th>
                    </tr>
                  </thead>
                  <tbody>
                    {agents.length === 0 ? (
                      <EmptyRow cols={6} loading={agentsQ.loading} message="No remote agents are currently registered." />
                    ) : (
                      agents.map((agent) => (
                        <tr
                          key={agent.id}
                          className={`border-b border-border/20 last:border-b-0 cursor-pointer transition-colors hover:bg-muted/30 ${selected?.id === agent.id ? 'bg-muted/30' : ''}`}
                          onClick={() => setSelectedId(agent.id)}
                        >
                          <Td className="font-mono">
                            <div>{agent.displayName || agent.name}</div>
                            <div className="text-[10px] text-muted-foreground">{agent.id}</div>
                          </Td>
                          <Td className="font-mono text-xs">{agent.poolName || '—'}</Td>
                          <Td><RemoteAgentAvailabilityBadge availability={agent.availability} /></Td>
                          <Td className="font-mono text-xs">{agent.currentTaskId || '—'}</Td>
                          <Td className="text-xs text-muted-foreground">{agent.lastActivityAt ? new Date(agent.lastActivityAt).toLocaleString() : '—'}</Td>
                          <Td className="font-mono text-xs text-muted-foreground">{agent.workspaceSessionId || '—'}</Td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </Table>
              </div>

              <div className="min-w-0 rounded-lg border border-border/60 bg-background/40 p-2">
                {selectedAgentQ.data ? (
                  <div className="space-y-3">
                    <div className="flex flex-wrap items-center gap-2">
                      <Link className="text-primary hover:underline" to="/remote-agents/agents/$agentId" params={{ agentId: selectedAgentQ.data.id }}>
                        Open full agent detail
                      </Link>
                      {selectedAgentQ.data.currentTaskId ? (
                        <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId" params={{ taskId: selectedAgentQ.data.currentTaskId }}>
                          Open current task
                        </Link>
                      ) : null}
                    </div>
                    <AgentDetailSections agent={selectedAgentQ.data} task={currentTaskQ.data} />
                  </div>
                ) : (
                  <div className="rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-sm text-muted-foreground">
                    {agentsQ.loading ? 'Loading…' : 'Select an agent to inspect its assignment, session binding, and pool context.'}
                  </div>
                )}
              </div>
            </div>
          </div>
        </ResourcePanel>
      </div>
    </>
  )
}
