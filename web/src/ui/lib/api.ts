export type Session = {
  id: string
  repoURL?: string
  phase?: string
}

export type Run = {
  id: string
  sessionID?: string
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

type FetchOptions = {
  method?: string
  body?: unknown
  token?: string
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
    body: opts.body === undefined ? undefined : JSON.stringify(opts.body)
  })
  const text = await res.text()
  if (!res.ok) {
    throw new APIError(`Request failed: ${res.status}`, res.status, text)
  }
  return text === '' ? (undefined as T) : (JSON.parse(text) as T)
}

export const api = {
  listSessions: (token: string) => apiFetch<{ sessions: Session[] }>('/api/v1/sessions', { token }),
  getSession: (token: string, id: string) => apiFetch<Session>(`/api/v1/sessions/${encodeURIComponent(id)}`, { token }),
  createSession: (token: string, repoURL: string) =>
    apiFetch<Session>('/api/v1/sessions', { method: 'POST', body: { repoURL }, token }),

  listRuns: (token: string, sessionID?: string) => {
    const q = sessionID ? `?sessionID=${encodeURIComponent(sessionID)}` : ''
    return apiFetch<{ runs: Run[] }>(`/api/v1/runs${q}`, { token })
  },
  getRun: (token: string, id: string) => apiFetch<Run>(`/api/v1/runs/${encodeURIComponent(id)}`, { token }),
  startRun: (
    token: string,
    sessionID: string,
    input: {
      repoURL: string
      repoRevision?: string
      image: string
      egressMode?: string
      ttlSecondsAfterFinished?: number
    }
  ) =>
    apiFetch<Run>(`/api/v1/sessions/${encodeURIComponent(sessionID)}/runs`, {
      method: 'POST',
      token,
      body: {
        repoURL: input.repoURL,
        repoRevision: input.repoRevision,
        image: input.image,
        egressMode: input.egressMode,
        ttlSecondsAfterFinished: input.ttlSecondsAfterFinished
      }
    }),
  stopRun: (token: string, runID: string) =>
    apiFetch<{ stopped: boolean }>(`/api/v1/runs/${encodeURIComponent(runID)}/stop`, { method: 'POST', token }),
  resumeRun: (token: string, runID: string) =>
    apiFetch<Run>(`/api/v1/runs/${encodeURIComponent(runID)}/resume`, { method: 'POST', token }),

  listAudit: (token: string, limit = 100) =>
    apiFetch<{ events: AuditEvent[] }>(`/api/v1/audit?limit=${encodeURIComponent(String(limit))}`, { token }),

  createAttachToken: (token: string, sessionID: string, role: 'viewer' | 'driver') =>
    apiFetch<{ token: string; expiresAt: string; sessionID: string; clientID: string; role: string }>(
      `/api/v1/sessions/${encodeURIComponent(sessionID)}/attach-token`,
      { method: 'POST', token, body: { role } }
    )
}
