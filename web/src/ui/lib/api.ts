export type WorkspaceSession = {
  id: string
  displayName?: string
  repoURL?: string
  phase?: string
  createdAt?: string
}

export type HarnessRun = {
  id: string
  displayName?: string
  workspaceSessionID?: string
  repoURL: string
  repoRevision?: string
  image: string
  phase?: string
  podName?: string
  gitHubBranch?: string
  pullRequestURL?: string
  pullRequestStatus?: string
}

export type AuditEvent = {
  id: string
  at: string
  actor: string
  action: string
  resourceType: string
  resourceID: string
  outcome: string
  metadata?: unknown
}

export type ClusterSummary = {
  sessionCount: number
  harnessRunCount: number
  podCount: number
  runningPods: number
  pendingPods: number
  failedPods: number
}

export type ClusterDeploymentStatus = {
  name: string
  readyReplicas: number
  availableReplicas: number
  desiredReplicas: number
  updatedReplicas: number
  unavailableReplicas: number
}

export type ClusterPodStatus = {
  name: string
  phase: string
  ready: string
  restarts: number
  nodeName?: string
  ageSeconds: number
}

export type ClusterConfigSnapshot = {
  environment?: string
  auditPathConfigured: boolean
  bootstrapTokenDetected: boolean
  gitHubCIDRsConfigured: boolean
}

export type ClusterOverview = {
  namespace: string
  collectedAt: string
  summary: ClusterSummary
  deployments: ClusterDeploymentStatus[]
  pods: ClusterPodStatus[]
  config: ClusterConfigSnapshot
}

export type PodLogs = {
  podName: string
  container?: string
  tailLines: number
  logs: string
}

export type SymphonyProjectCondition = {
  type: string
  status: string
  reason?: string
  message?: string
  lastTransitionTime?: string
}

export type SymphonyProjectIssueRef = {
  repository?: string
  number?: number
  nodeId?: string
  url?: string
  title?: string
}

export type SymphonyProjectRunRef = {
  sessionName?: string
  harnessRunName?: string
}

export type SymphonyProjectClaim = {
  itemId: string
  issue?: SymphonyProjectIssueRef
  attempt?: number
  phase?: string
  claimedAt?: string
  lastUpdatedTime?: string
  runRef?: SymphonyProjectRunRef
}

export type SymphonyProjectRetry = {
  itemId: string
  issue?: SymphonyProjectIssueRef
  attempt?: number
  reason?: string
  readyAt?: string
  lastErrorTime?: string
}

export type SymphonyProjectSkip = {
  itemId: string
  issue?: SymphonyProjectIssueRef
  repository?: string
  reason?: string
  message?: string
  observedTime?: string
}

export type SymphonyProjectError = {
  itemId: string
  issue?: SymphonyProjectIssueRef
  attempt?: number
  reason?: string
  lastErrorTime?: string
  harnessRunName?: string
}

export type SymphonyProjectEvent = {
  itemId: string
  issue?: SymphonyProjectIssueRef
  sessionId?: string
  threadId?: string
  turnId?: string
  event?: string
  message?: string
  observedTime?: string
  harnessRunName?: string
}

export type SymphonyProjectTokenTotals = {
  inputTokens?: number
  outputTokens?: number
  totalTokens?: number
  secondsRunning?: number
}

export type SymphonyProjectRepository = {
  owner: string
  name: string
  repoURL?: string
  localPath?: string
  workflowPath?: string
  branch?: string
  egressMode?: string
}

export type SymphonyProjectSpec = {
  paused?: boolean
  source: {
    project: {
      owner: string
      number: number
    }
    tokenSecretRef: {
      name: string
      key?: string
    }
    activeStates: string[]
    terminalStates: string[]
    fieldName?: string
    pollIntervalSeconds?: number
  }
  repositories: SymphonyProjectRepository[]
  runtime: {
    image: string
    command?: string[]
    args?: string[]
    workingDir?: string
    maxConcurrentItems?: number
    retryBaseDelaySeconds?: number
    retryMaxDelaySeconds?: number
    ttlSecondsAfterFinished?: number | null
    recentSkipLimit?: number
    recentErrorLimit?: number
    activeStatusItemLimit?: number
    defaultRepoRevision?: string
    defaultEgressMode?: string
  }
}

export type SymphonyProjectStatus = {
  observedGeneration?: number
  phase?: string
  conditions?: SymphonyProjectCondition[]
  resolvedFieldName?: string
  lastSyncTime?: string
  lastSuccessfulSyncTime?: string
  nextSyncTime?: string
  activeClaims?: SymphonyProjectClaim[]
  retryQueue?: SymphonyProjectRetry[]
  recentErrors?: SymphonyProjectError[]
  recentEvents?: SymphonyProjectEvent[]
  tokenTotals?: SymphonyProjectTokenTotals
  recentSkips?: SymphonyProjectSkip[]
  unsupportedRepositories?: string[]
  lastError?: string
  eligibleItems?: number
  runningItems?: number
  retryingItems?: number
  completedItems?: number
  failedItems?: number
  skippedItems?: number
}

export type SymphonyProject = {
  name: string
  namespace?: string
  createdAt?: string
  generation?: number
  paused: boolean
  spec: SymphonyProjectSpec
  status: SymphonyProjectStatus
}

export type SymphonyProjectRequest = {
  name: string
  spec: Omit<SymphonyProjectSpec, 'source'> & {
    source: SymphonyProjectSpec['source'] & {
      githubToken?: string
    }
  }
}

type FetchOptions = {
  method?: string
  body?: unknown
  token?: string
  credentials?: RequestCredentials
}

export class APIError extends Error {
  readonly status: number
  readonly bodyText: string

  constructor(message: string, status: number, bodyText: string) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.bodyText = bodyText
  }
}

export function isUnauthorizedError(e: unknown): e is APIError {
  return e instanceof APIError && e.status === 401
}

async function apiFetch<T>(path: string, opts: FetchOptions = {}): Promise<T> {
  const headers: Record<string, string> = {}
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  if (opts.token && opts.token.trim() !== '') {
    headers['Authorization'] = `Bearer ${opts.token.trim()}`
  }
  const res = await fetch(path, {
    method: opts.method ?? 'GET',
    headers,
    body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
    credentials: opts.credentials
  })
  const text = await res.text()
  if (!res.ok) {
    throw new APIError(`Request failed: ${res.status}`, res.status, text)
  }
  return text === '' ? (undefined as T) : (JSON.parse(text) as T)
}

export const api = {
  listWorkspaceSessions: (token: string) =>
    apiFetch<{ workspaceSessions: WorkspaceSession[] }>('/api/v1/workspace-sessions', { token }),
  getWorkspaceSession: (token: string, id: string) =>
    apiFetch<WorkspaceSession>(`/api/v1/workspace-sessions/${encodeURIComponent(id)}`, { token }),
  createWorkspaceSession: (token: string, repoURL: string, displayName?: string) =>
    apiFetch<WorkspaceSession>('/api/v1/workspace-sessions', { method: 'POST', body: { repoURL, displayName }, token }),
  deleteWorkspaceSession: (token: string, id: string) =>
    apiFetch<{ deleted: boolean }>(`/api/v1/workspace-sessions/${encodeURIComponent(id)}`, { method: 'DELETE', token }),

  listHarnessRuns: (token: string, workspaceSessionID?: string) => {
    const q = workspaceSessionID ? `?workspaceSessionID=${encodeURIComponent(workspaceSessionID)}` : ''
    return apiFetch<{ harnessRuns: HarnessRun[] }>(`/api/v1/harness-runs${q}`, { token })
  },
  getHarnessRun: (token: string, id: string) => apiFetch<HarnessRun>(`/api/v1/harness-runs/${encodeURIComponent(id)}`, { token }),
  startHarnessRun: (
    token: string,
    workspaceSessionID: string,
    input: {
      repoURL: string
      repoRevision?: string
      image: string
      egressMode?: string
      args?: string[]
      ttlSecondsAfterFinished?: number
    }
  ) =>
    apiFetch<HarnessRun>(`/api/v1/workspace-sessions/${encodeURIComponent(workspaceSessionID)}/harness-runs`, {
      method: 'POST',
      token,
      body: {
        repoURL: input.repoURL,
        repoRevision: input.repoRevision,
        image: input.image,
        egressMode: input.egressMode,
        args: input.args,
        ttlSecondsAfterFinished: input.ttlSecondsAfterFinished
      }
    }),
  stopHarnessRun: (token: string, harnessRunID: string) =>
    apiFetch<{ stopped: boolean }>(`/api/v1/harness-runs/${encodeURIComponent(harnessRunID)}/stop`, { method: 'POST', token }),
  resumeHarnessRun: (token: string, harnessRunID: string) =>
    apiFetch<HarnessRun>(`/api/v1/harness-runs/${encodeURIComponent(harnessRunID)}/resume`, { method: 'POST', token }),

  listAudit: (token: string, limit = 100) =>
    apiFetch<{ events: AuditEvent[] }>(`/api/v1/audit?limit=${encodeURIComponent(String(limit))}`, { token }),

  getClusterOverview: (token: string) =>
    apiFetch<ClusterOverview>('/api/v1/cluster-overview', { token }),

  getPodLogs: (token: string, podName: string, opts?: { container?: string; tailLines?: number }) => {
    const qs = new URLSearchParams()
    if (opts?.container && opts.container.trim() !== '') qs.set('container', opts.container.trim())
    if (opts?.tailLines && opts.tailLines > 0) qs.set('tailLines', String(opts.tailLines))
    const suffix = qs.toString() ? `?${qs.toString()}` : ''
    return apiFetch<PodLogs>(`/api/v1/pods/${encodeURIComponent(podName)}/logs${suffix}`, { token })
  },

  createAttachToken: (token: string, workspaceSessionID: string, role: 'viewer' | 'driver', mode: 'exclusive' | 'collab' = 'exclusive') =>
    apiFetch<{ token: string; expiresAt: string; workspaceSessionID: string; clientID: string; role: string; mode?: string }>(
      `/api/v1/workspace-sessions/${encodeURIComponent(workspaceSessionID)}/attach-token`,
      { method: 'POST', token, body: { role, mode } }
    ),

  createAttachCookie: (token: string, workspaceSessionID: string, role: 'viewer' | 'driver', mode: 'exclusive' | 'collab' = 'exclusive') =>
    apiFetch<{ expiresAt: string; workspaceSessionID: string; clientID: string; role: string; mode?: string }>(
      `/api/v1/workspace-sessions/${encodeURIComponent(workspaceSessionID)}/attach-cookie`,
      { method: 'POST', token, body: { role, mode }, credentials: 'include' }
    ),

  listSymphonyProjects: (token: string) =>
    apiFetch<{ symphonyProjects: SymphonyProject[] }>('/api/v1/symphony-projects', { token }),
  getSymphonyProject: (token: string, projectName: string) =>
    apiFetch<SymphonyProject>(`/api/v1/symphony-projects/${encodeURIComponent(projectName)}`, { token }),
  createSymphonyProject: (token: string, input: SymphonyProjectRequest) =>
    apiFetch<SymphonyProject>('/api/v1/symphony-projects', { method: 'POST', token, body: input }),
  updateSymphonyProject: (token: string, projectName: string, input: SymphonyProjectRequest) =>
    apiFetch<SymphonyProject>(`/api/v1/symphony-projects/${encodeURIComponent(projectName)}`, {
      method: 'PATCH',
      token,
      body: input,
    }),
  pauseSymphonyProject: (token: string, projectName: string) =>
    apiFetch<SymphonyProject>(`/api/v1/symphony-projects/${encodeURIComponent(projectName)}/pause`, { method: 'POST', token }),
  resumeSymphonyProject: (token: string, projectName: string) =>
    apiFetch<SymphonyProject>(`/api/v1/symphony-projects/${encodeURIComponent(projectName)}/resume`, { method: 'POST', token }),
  refreshSymphonyProject: (token: string, projectName: string) =>
    apiFetch<SymphonyProject>(`/api/v1/symphony-projects/${encodeURIComponent(projectName)}/refresh`, { method: 'POST', token }),
}
