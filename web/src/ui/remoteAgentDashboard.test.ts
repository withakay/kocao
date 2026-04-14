import { describe, expect, it } from 'vitest'
import {
  getRemoteAgentRoute,
  getRemoteAgentTaskDrillDownRoute,
  remoteAgentDashboardInformationArchitecture,
  summarizeTranscript,
} from './lib/remoteAgentDashboard'

describe('remote-agent-dashboard-information-architecture', () => {
  it('anchors navigation around first-class orchestration resources', () => {
    expect(remoteAgentDashboardInformationArchitecture.baseRoute).toBe('/remote-agents')
    expect(remoteAgentDashboardInformationArchitecture.landingRoute).toBe('/remote-agents/tasks')
    expect(remoteAgentDashboardInformationArchitecture.collections.map((collection) => collection.resource)).toEqual([
      'agents',
      'tasks',
      'transcripts',
      'artifacts',
    ])
  })

  it('documents pools as a subordinate wave-3 grouping, not a standalone dashboard route', () => {
    expect(remoteAgentDashboardInformationArchitecture.supportingDimensions).toEqual([
      {
        key: 'pools',
        treatment: 'subordinate',
        reason:
          'Wave 3 keeps pools inside agent and task detail because operators act on named agents and durable tasks, while pool management remains a grouping/filtering concern rather than a standalone workflow.',
      },
    ])
  })

  it('uses explicit lifecycle states without a queued bucket', () => {
    const states = Object.values(remoteAgentDashboardInformationArchitecture.taskStateGroups).flat()
    expect(states).toEqual(['assigned', 'running', 'completed', 'failed', 'timed_out', 'cancelled'])
    expect(states).not.toContain('queued')
  })

  it('defines operator drill-downs from tasks into transcript and artifacts', () => {
    const taskCollection = remoteAgentDashboardInformationArchitecture.collections.find((collection) => collection.resource === 'tasks')
    expect(taskCollection?.drillDowns.map((item) => item.route)).toEqual([
      '/remote-agents/tasks/$taskId/transcript',
      '/remote-agents/tasks/$taskId/artifacts',
      '/remote-agents/agents/$agentId',
    ])
    expect(taskCollection?.detailSections).toEqual([
      'summary',
      'assignment',
      'session',
      'timeline',
      'transcript',
      'artifacts',
      'metadata',
    ])
  })

  it('builds drill-down urls from durable agent and task identifiers', () => {
    expect(getRemoteAgentRoute({ id: 'agent-reviewer' })).toBe('/remote-agents/agents/agent-reviewer')
    expect(getRemoteAgentTaskDrillDownRoute({ id: 'task-42' }, 'task')).toBe('/remote-agents/tasks/task-42')
    expect(getRemoteAgentTaskDrillDownRoute({ id: 'task-42' }, 'transcript')).toBe('/remote-agents/tasks/task-42/transcript')
    expect(getRemoteAgentTaskDrillDownRoute({ id: 'task-42' }, 'artifacts')).toBe('/remote-agents/tasks/task-42/artifacts')
  })

  it('summarizes transcript rows without depending on live session envelopes', () => {
    expect(summarizeTranscript({ role: 'agent', text: '  completed patch generation  ' })).toBe('completed patch generation')
    expect(summarizeTranscript({ role: 'tool', eventRef: 'evt-17' })).toBe('evt-17')
    expect(summarizeTranscript({ role: 'system' })).toBe('system event')
  })
})
