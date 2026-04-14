import type {
  RemoteAgent,
  RemoteAgentArtifactKind,
  RemoteAgentTask,
  RemoteAgentTaskState,
  RemoteAgentTranscriptEntry,
} from './api'

export const remoteAgentDashboardBaseRoute = '/remote-agents'
export const remoteAgentTaskListSearchDefaults = {
  q: '',
  pool: 'all',
  state: 'all',
  artifacts: 'all',
} as const

export type RemoteAgentDashboardResource = 'agents' | 'tasks' | 'transcripts' | 'artifacts'

export type RemoteAgentDashboardDetailSection =
  | 'summary'
  | 'assignment'
  | 'session'
  | 'timeline'
  | 'transcript'
  | 'artifacts'
  | 'metadata'

export type RemoteAgentDashboardColumn = {
  key: string
  label: string
  emphasis?: 'primary' | 'secondary'
}

export type RemoteAgentDashboardDrillDown = {
  label: string
  resource: RemoteAgentDashboardResource
  route: string
  reason: string
}

export type RemoteAgentDashboardSupportingDimension = {
  key: 'pools'
  treatment: 'subordinate'
  reason: string
}

export type RemoteAgentDashboardCollection = {
  resource: RemoteAgentDashboardResource
  title: string
  route: string
  description: string
  columns: RemoteAgentDashboardColumn[]
  defaultSort: string
  defaultEmptyMessage: string
  detailSections: RemoteAgentDashboardDetailSection[]
  drillDowns: RemoteAgentDashboardDrillDown[]
}

export const remoteAgentTaskStateGroups: Record<'active' | 'terminal', RemoteAgentTaskState[]> = {
  active: ['assigned', 'running'],
  terminal: ['completed', 'failed', 'timed_out', 'cancelled'],
}

export const remoteAgentArtifactKinds: RemoteAgentArtifactKind[] = ['file', 'patch', 'bundle', 'report']

export const remoteAgentDashboardCollections: RemoteAgentDashboardCollection[] = [
  {
    resource: 'agents',
    title: 'Active Remote Agents',
    route: '/remote-agents/agents',
    description: 'Operational list of named agents and pools, keyed by agent identity instead of harness-run ids.',
    columns: [
      { key: 'displayName', label: 'Agent', emphasis: 'primary' },
      { key: 'poolName', label: 'Pool' },
      { key: 'availability', label: 'Availability' },
      { key: 'currentTaskId', label: 'Current Task' },
      { key: 'lastActivityAt', label: 'Last Activity' },
      { key: 'workspaceSessionId', label: 'Workspace Session' },
    ],
    defaultSort: 'lastActivityAt:desc',
    defaultEmptyMessage: 'No remote agents are currently registered.',
    detailSections: ['summary', 'assignment', 'session', 'timeline', 'metadata'],
    drillDowns: [
      {
        label: 'Open current task',
        resource: 'tasks',
        route: '/remote-agents/tasks/$taskId',
        reason: 'Operators need the assigned work record before inspecting transcript or artifacts.',
      },
    ],
  },
  {
    resource: 'tasks',
    title: 'Current Tasks',
    route: '/remote-agents/tasks',
    description: 'Primary operations queue keyed by durable task records and explicit lifecycle states.',
    columns: [
      { key: 'id', label: 'Task', emphasis: 'primary' },
      { key: 'state', label: 'State' },
      { key: 'agentName', label: 'Agent' },
      { key: 'poolName', label: 'Pool' },
      { key: 'attempt', label: 'Attempt' },
      { key: 'lastTransitionAt', label: 'Last Transition' },
    ],
    defaultSort: 'lastTransitionAt:desc',
    defaultEmptyMessage: 'No remote-agent tasks match the current filters.',
    detailSections: ['summary', 'assignment', 'session', 'timeline', 'transcript', 'artifacts', 'metadata'],
    drillDowns: [
      {
        label: 'Open transcript',
        resource: 'transcripts',
        route: '/remote-agents/tasks/$taskId/transcript',
        reason: 'Transcript review is task-scoped and should stay anchored to the durable task id.',
      },
      {
        label: 'Open artifacts',
        resource: 'artifacts',
        route: '/remote-agents/tasks/$taskId/artifacts',
        reason: 'Artifacts belong to the task result, not to an ephemeral runtime session.',
      },
      {
        label: 'Open assigned agent',
        resource: 'agents',
        route: '/remote-agents/agents/$agentId',
        reason: 'Operators often pivot from a task back to the named worker handling it.',
      },
    ],
  },
  {
    resource: 'transcripts',
    title: 'Transcripts',
    route: '/remote-agents/tasks/$taskId/transcript',
    description: 'Task-level transcript explorer for persisted prompts, tool calls, and agent responses.',
    columns: [
      { key: 'sequence', label: 'Sequence', emphasis: 'primary' },
      { key: 'at', label: 'At' },
      { key: 'role', label: 'Role' },
      { key: 'kind', label: 'Kind' },
      { key: 'eventRef', label: 'Event Ref' },
    ],
    defaultSort: 'sequence:asc',
    defaultEmptyMessage: 'No transcript entries were persisted for this task.',
    detailSections: ['summary', 'transcript', 'metadata'],
    drillDowns: [
      {
        label: 'Return to task',
        resource: 'tasks',
        route: '/remote-agents/tasks/$taskId',
        reason: 'Operators should preserve task context while reading transcript details.',
      },
      {
        label: 'Open referenced artifacts',
        resource: 'artifacts',
        route: '/remote-agents/tasks/$taskId/artifacts',
        reason: 'Transcript lines often explain how a generated patch or report was produced.',
      },
    ],
  },
  {
    resource: 'artifacts',
    title: 'Artifacts',
    route: '/remote-agents/tasks/$taskId/artifacts',
    description: 'Task output browser for generated files, patch bundles, and reports that survive session teardown.',
    columns: [
      { key: 'name', label: 'Artifact', emphasis: 'primary' },
      { key: 'kind', label: 'Kind' },
      { key: 'mediaType', label: 'Media Type' },
      { key: 'sizeBytes', label: 'Size' },
      { key: 'createdAt', label: 'Created' },
    ],
    defaultSort: 'createdAt:desc',
    defaultEmptyMessage: 'No task artifacts are available yet.',
    detailSections: ['summary', 'artifacts', 'metadata'],
    drillDowns: [
      {
        label: 'Return to task',
        resource: 'tasks',
        route: '/remote-agents/tasks/$taskId',
        reason: 'Artifact actions stay grounded in the parent task result.',
      },
      {
        label: 'Open transcript context',
        resource: 'transcripts',
        route: '/remote-agents/tasks/$taskId/transcript',
        reason: 'Operators need the surrounding transcript to understand why an artifact was produced.',
      },
    ],
  },
]

export const remoteAgentDashboardInformationArchitecture = {
  navLabel: 'Remote Agents',
  baseRoute: remoteAgentDashboardBaseRoute,
  landingRoute: '/remote-agents/tasks',
  supportingDimensions: [
    {
      key: 'pools',
      treatment: 'subordinate',
      reason:
        'Wave 3 keeps pools inside agent and task detail because operators act on named agents and durable tasks, while pool management remains a grouping/filtering concern rather than a standalone workflow.',
    },
  ] as const satisfies readonly RemoteAgentDashboardSupportingDimension[],
  overviewCards: [
    'active_agents',
    'active_tasks',
    'terminal_tasks_24h',
    'artifacts_24h',
  ] as const,
  splitView: {
    primaryPane: 'resource list',
    secondaryPane: 'selected detail',
    tertiaryPane: 'transcript or artifact preview when task detail is selected',
  },
  collections: remoteAgentDashboardCollections,
  taskStateGroups: remoteAgentTaskStateGroups,
  artifactKinds: remoteAgentArtifactKinds,
} as const

export function getRemoteAgentTaskDrillDownRoute(task: Pick<RemoteAgentTask, 'id'>, target: 'task' | 'transcript' | 'artifacts') {
  if (target === 'task') {
    return `/remote-agents/tasks/${task.id}`
  }
  if (target === 'transcript') {
    return `/remote-agents/tasks/${task.id}/transcript`
  }
  return `/remote-agents/tasks/${task.id}/artifacts`
}

export function getRemoteAgentRoute(agent: Pick<RemoteAgent, 'id'>) {
  return `/remote-agents/agents/${agent.id}`
}

export function summarizeTranscript(entry: Pick<RemoteAgentTranscriptEntry, 'role' | 'text' | 'eventRef'>) {
  if (entry.text && entry.text.trim() !== '') {
    return entry.text.trim()
  }
  return entry.eventRef?.trim() || `${entry.role} event`
}
