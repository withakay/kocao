import { useCallback, useMemo } from 'react'
import { Link, useRouterState } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { api } from '../lib/api'
import { usePollingQuery } from '../lib/usePolling'
import { Topbar } from '../components/Topbar'
import { Btn, EmptyRow, ErrorBanner, FormRow, Input, Select, Table, Td, Th } from '../components/primitives'
import {
  RemoteAgentOverviewCards,
  RemoteAgentTaskStateBadge,
  ResourcePanel,
  TaskDetailSections,
  useSelectableList,
} from './remoteAgentDashboardShared'
import { remoteAgentDashboardInformationArchitecture, remoteAgentTaskListSearchDefaults, remoteAgentTaskStateGroups } from '../lib/remoteAgentDashboard'

export function RemoteAgentsPage() {
  const { token, invalidateToken } = useAuth()
  const locationSearch = useRouterState({ select: (state) => state.location.searchStr })
  const search = new URLSearchParams(locationSearch)
  const stateParam = search.get('state')
  const artifactParam = search.get('artifacts')
  const filter = search.get('q') ?? remoteAgentTaskListSearchDefaults.q
  const poolFilter = search.get('pool') ?? remoteAgentTaskListSearchDefaults.pool
  const stateFilter = stateParam === 'active' || stateParam === 'terminal' ? stateParam : remoteAgentTaskListSearchDefaults.state
  const artifactFilter = artifactParam === 'with-output' ? 'with-output' : remoteAgentTaskListSearchDefaults.artifacts

  const updateFilters = (updates: Partial<Record<'q' | 'pool' | 'state' | 'artifacts', string>>) => {
    const params = new URLSearchParams(locationSearch)
    for (const [key, value] of Object.entries(updates)) {
      params.set(key, value)
    }
    window.location.hash = `#/remote-agents/tasks?${params.toString()}`
  }

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

  const tasks = useMemo(() => {
    const all = (tasksQ.data?.remoteAgentTasks ?? []).slice().sort((left, right) => {
      return (Date.parse(right.lastTransitionAt ?? right.createdAt ?? '') || 0) - (Date.parse(left.lastTransitionAt ?? left.createdAt ?? '') || 0)
    })
    const normalizedFilter = filter.trim().toLowerCase()
    return all.filter((task) => {
      const matchesFilter = normalizedFilter === '' || [
        task.id,
        task.agentName,
        task.poolName,
        task.prompt,
      ].some((value) => (value ?? '').toLowerCase().includes(normalizedFilter))
      const matchesPool = poolFilter === 'all' || (task.poolName ?? '') === poolFilter
      const matchesState =
        stateFilter === 'all' ||
        (stateFilter === 'active' && remoteAgentTaskStateGroups.active.includes(task.state)) ||
        (stateFilter === 'terminal' && remoteAgentTaskStateGroups.terminal.includes(task.state))
      const matchesArtifacts = artifactFilter === 'all' || (task.result?.outputArtifactCount ?? 0) > 0
      return matchesFilter && matchesPool && matchesState && matchesArtifacts
    })
  }, [artifactFilter, filter, poolFilter, stateFilter, tasksQ.data])

  const { selectedId, setSelectedId, selected } = useSelectableList(tasks)
  const selectedTaskQ = usePollingQuery(`remote-agent-task:${selectedId}:${token}`, () => api.getRemoteAgentTask(token, selectedId), {
    intervalMs: 2500,
    enabled: token.trim() !== '' && selectedId !== '',
    onUnauthorized,
  })
  const transcriptQ = usePollingQuery(`remote-agent-task-transcript:${selectedId}:${token}`, () => api.getRemoteAgentTaskTranscript(token, selectedId), {
    intervalMs: 2500,
    enabled: token.trim() !== '' && selectedId !== '',
    onUnauthorized,
  })
  const artifactsQ = usePollingQuery(`remote-agent-task-artifacts:${selectedId}:${token}`, () => api.getRemoteAgentTaskArtifacts(token, selectedId), {
    intervalMs: 2500,
    enabled: token.trim() !== '' && selectedId !== '',
    onUnauthorized,
  })

  const pools = useMemo(() => {
    const names = new Set<string>()
    for (const pool of poolsQ.data?.remoteAgentPools ?? []) {
      if ((pool.name ?? '').trim() !== '') names.add(pool.name)
    }
    for (const task of tasksQ.data?.remoteAgentTasks ?? []) {
      if ((task.poolName ?? '').trim() !== '') names.add(task.poolName ?? '')
    }
    return Array.from(names).sort((left, right) => left.localeCompare(right))
  }, [poolsQ.data, tasksQ.data])

  const selectedTask = selectedTaskQ.data

  return (
    <>
      <Topbar title="Remote Agents" subtitle="Operator dashboard for active agents, current tasks, transcripts, and artifacts." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <RemoteAgentOverviewCards agents={agentsQ.data?.remoteAgents ?? []} tasks={tasksQ.data?.remoteAgentTasks ?? []} />

        <ResourcePanel
          title="Current Tasks"
          description={remoteAgentDashboardInformationArchitecture.collections.find((collection) => collection.resource === 'tasks')?.description}
          right={<Btn onClick={() => { tasksQ.reload(); agentsQ.reload() }} type="button">Refresh</Btn>}
        >
          <div className="space-y-3">
            <div className="grid gap-2 md:grid-cols-4">
              <FormRow label="Search">
                <Input
                  value={filter}
                  onChange={(event) => updateFilters({ q: event.target.value })}
                  placeholder="task, agent, pool, prompt"
                />
              </FormRow>
              <FormRow label="Pool">
                <Select
                  value={poolFilter}
                  onChange={(event) => updateFilters({ pool: event.target.value })}
                >
                  <option value="all">All pools</option>
                  {pools.map((pool) => <option key={pool} value={pool}>{pool}</option>)}
                </Select>
              </FormRow>
              <FormRow label="State">
                <Select
                  value={stateFilter}
                  onChange={(event) => updateFilters({ state: event.target.value })}
                >
                  <option value="all">All states</option>
                  <option value="active">Active only</option>
                  <option value="terminal">Terminal only</option>
                </Select>
              </FormRow>
              <FormRow label="Artifacts">
                <Select
                  value={artifactFilter}
                  onChange={(event) => updateFilters({ artifacts: event.target.value })}
                >
                  <option value="all">All tasks</option>
                  <option value="with-output">With artifacts</option>
                </Select>
              </FormRow>
            </div>

            {tasksQ.error ? <ErrorBanner>{tasksQ.error}</ErrorBanner> : null}

            <div className="grid gap-3 xl:grid-cols-[minmax(0,1.15fr)_minmax(22rem,0.85fr)]">
              <div className="rounded-lg border border-border/60 bg-background/40 p-2">
                <Table label="remote agent tasks table">
                  <thead>
                    <tr className="border-b border-border/40">
                      <Th>Task</Th>
                      <Th>State</Th>
                      <Th>Agent</Th>
                      <Th>Pool</Th>
                      <Th>Attempt</Th>
                      <Th>Last Transition</Th>
                    </tr>
                  </thead>
                  <tbody>
                    {tasks.length === 0 ? (
                      <EmptyRow cols={6} loading={tasksQ.loading} message="No remote-agent tasks match the current filters." />
                    ) : (
                      tasks.map((task) => (
                        <tr
                          key={task.id}
                          className={`border-b border-border/20 last:border-b-0 cursor-pointer transition-colors hover:bg-muted/30 ${selected?.id === task.id ? 'bg-muted/30' : ''}`}
                          onClick={() => setSelectedId(task.id)}
                        >
                          <Td className="font-mono">
                            <div className="flex items-center gap-2">
                              <button type="button" className="font-mono text-left text-primary hover:underline" onClick={() => setSelectedId(task.id)}>{task.id}</button>
                              {selected?.id === task.id ? <span className="text-[10px] uppercase tracking-[0.18em] text-muted-foreground">selected</span> : null}
                            </div>
                          </Td>
                          <Td><RemoteAgentTaskStateBadge state={task.state} /></Td>
                          <Td className="font-mono text-xs">{task.agentName || '—'}</Td>
                          <Td className="font-mono text-xs">{task.poolName || '—'}</Td>
                          <Td className="font-mono">{task.attempt ?? 0}</Td>
                          <Td className="text-xs text-muted-foreground">{task.lastTransitionAt ? new Date(task.lastTransitionAt).toLocaleString() : '—'}</Td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </Table>
              </div>

              <div className="min-w-0 rounded-lg border border-border/60 bg-background/40 p-2">
                {selectedTask ? (
                  <div className="space-y-3">
                    <div className="flex flex-wrap items-center gap-2">
                      <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId" params={{ taskId: selectedTask.id }}>
                        Open full task detail
                      </Link>
                      <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId/transcript" params={{ taskId: selectedTask.id }}>
                        Transcript
                      </Link>
                      <Link className="text-primary hover:underline" to="/remote-agents/tasks/$taskId/artifacts" params={{ taskId: selectedTask.id }}>
                        Artifacts
                      </Link>
                    </div>
                    <TaskDetailSections
                      task={selectedTask}
                      transcript={transcriptQ.data?.transcript ?? []}
                      artifacts={artifactsQ.data}
                    />
                  </div>
                ) : (
                  <div className="rounded-md border border-border/60 bg-muted/20 px-3 py-2 text-sm text-muted-foreground">
                    {tasksQ.loading ? 'Loading…' : 'Select a task to inspect assignment, transcript, and artifact detail.'}
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
