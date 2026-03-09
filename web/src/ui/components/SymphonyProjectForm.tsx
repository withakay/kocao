import { useEffect, useMemo, useState } from 'react'
import type { SymphonyProject, SymphonyProjectRequest, SymphonyProjectSpec } from '../lib/api'
import { Btn, ErrorBanner, FormRow, Input, Textarea } from './primitives'

type SymphonyProjectFormProps = {
  initialProject?: SymphonyProject
  submitLabel: string
  busy?: boolean
  error?: string | null
  onSubmit: (request: SymphonyProjectRequest) => Promise<void> | void
}

type SymphonyDraft = {
  name: string
  paused: boolean
  projectOwner: string
  projectNumber: string
  tokenSecretName: string
  fieldName: string
  activeStatesText: string
  terminalStatesText: string
  repositoriesText: string
  image: string
  defaultRepoRevision: string
  maxConcurrentItems: string
}

const defaultDraft: SymphonyDraft = {
  name: '',
  paused: false,
  projectOwner: 'withakay',
  projectNumber: '1',
  tokenSecretName: 'github-token',
  fieldName: 'Status',
  activeStatesText: 'Todo',
  terminalStatesText: 'Done',
  repositoriesText: 'withakay/kocao',
  image: 'ghcr.io/withakay/kocao-harness:latest',
  defaultRepoRevision: 'main',
  maxConcurrentItems: '1',
}

function toDraft(project?: SymphonyProject): SymphonyDraft {
  if (!project) return defaultDraft
  return {
    name: project.name,
    paused: project.paused,
    projectOwner: project.spec.source.project.owner,
    projectNumber: String(project.spec.source.project.number),
    tokenSecretName: project.spec.source.tokenSecretRef.name,
    fieldName: project.spec.source.fieldName ?? 'Status',
    activeStatesText: (project.spec.source.activeStates ?? []).join(', '),
    terminalStatesText: (project.spec.source.terminalStates ?? []).join(', '),
    repositoriesText: (project.spec.repositories ?? [])
      .map((repo) => `${repo.owner}/${repo.name}${repo.branch ? `@${repo.branch}` : ''}`)
      .join('\n'),
    image: project.spec.runtime.image,
    defaultRepoRevision: project.spec.runtime.defaultRepoRevision ?? 'main',
    maxConcurrentItems: String(project.spec.runtime.maxConcurrentItems ?? 1),
  }
}

function parseList(raw: string): string[] {
  return raw
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
}

function parseRepositories(raw: string) {
  return raw
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [repoPart, branchPart] = line.split('@', 2)
      const [owner = '', name = ''] = repoPart.split('/', 2)
      return {
        owner: owner.trim(),
        name: name.trim(),
        branch: branchPart?.trim() || undefined,
      }
    })
}

function buildRequest(draft: SymphonyDraft): SymphonyProjectRequest {
  const projectNumber = Number(draft.projectNumber)
  const maxConcurrentItems = Number(draft.maxConcurrentItems)
  const repositories = parseRepositories(draft.repositoriesText)
  const spec: SymphonyProjectSpec = {
    paused: draft.paused,
    source: {
      project: {
        owner: draft.projectOwner.trim(),
        number: Number.isFinite(projectNumber) ? projectNumber : 0,
      },
      tokenSecretRef: {
        name: draft.tokenSecretName.trim(),
      },
      fieldName: draft.fieldName.trim(),
      activeStates: parseList(draft.activeStatesText),
      terminalStates: parseList(draft.terminalStatesText),
    },
    repositories,
    runtime: {
      image: draft.image.trim(),
      defaultRepoRevision: draft.defaultRepoRevision.trim(),
      maxConcurrentItems: Number.isFinite(maxConcurrentItems) ? maxConcurrentItems : 0,
    },
  }
  return {
    name: draft.name.trim(),
    spec,
  }
}

export function SymphonyProjectForm({ initialProject, submitLabel, busy = false, error, onSubmit }: SymphonyProjectFormProps) {
  const [draft, setDraft] = useState<SymphonyDraft>(() => toDraft(initialProject))
  const [localError, setLocalError] = useState<string | null>(null)

  useEffect(() => {
    setDraft(toDraft(initialProject))
    setLocalError(null)
  }, [initialProject])

  const helperText = useMemo(
    () => ({
      states: 'Comma-separated GitHub Projects field values.',
      repositories: 'One repository per line as owner/name or owner/name@branch.',
    }),
    [],
  )

  return (
    <form
      className="space-y-2"
      onSubmit={async (event) => {
        event.preventDefault()
        setLocalError(null)
        try {
          await onSubmit(buildRequest(draft))
        } catch (submitError) {
          setLocalError(submitError instanceof Error ? submitError.message : String(submitError))
        }
      }}
    >
      <div className="grid gap-2 md:grid-cols-2">
        <FormRow label="Name">
          <Input
            aria-label="Name"
            value={draft.name}
            onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))}
            disabled={busy || !!initialProject}
            placeholder="triage-board"
          />
        </FormRow>
        <FormRow label="Paused">
          <label className="inline-flex items-center gap-2 rounded-md border border-input px-3 py-2 text-sm">
            <input
              aria-label="Paused"
              type="checkbox"
              checked={draft.paused}
              disabled={busy}
              onChange={(event) => setDraft((current) => ({ ...current, paused: event.target.checked }))}
            />
            Start in paused mode
          </label>
        </FormRow>
        <FormRow label="GitHub Owner">
          <Input aria-label="GitHub Owner" value={draft.projectOwner} onChange={(event) => setDraft((current) => ({ ...current, projectOwner: event.target.value }))} disabled={busy} />
        </FormRow>
        <FormRow label="Project #">
          <Input aria-label="Project #" value={draft.projectNumber} onChange={(event) => setDraft((current) => ({ ...current, projectNumber: event.target.value }))} disabled={busy} inputMode="numeric" />
        </FormRow>
        <FormRow label="Token Secret">
          <Input aria-label="Token Secret" value={draft.tokenSecretName} onChange={(event) => setDraft((current) => ({ ...current, tokenSecretName: event.target.value }))} disabled={busy} />
        </FormRow>
        <FormRow label="Field Name">
          <Input aria-label="Field Name" value={draft.fieldName} onChange={(event) => setDraft((current) => ({ ...current, fieldName: event.target.value }))} disabled={busy} placeholder="Status" />
        </FormRow>
        <FormRow label="Active States" hint={helperText.states}>
          <Input aria-label="Active States" value={draft.activeStatesText} onChange={(event) => setDraft((current) => ({ ...current, activeStatesText: event.target.value }))} disabled={busy} placeholder="Todo, In Progress" />
        </FormRow>
        <FormRow label="Terminal States" hint={helperText.states}>
          <Input aria-label="Terminal States" value={draft.terminalStatesText} onChange={(event) => setDraft((current) => ({ ...current, terminalStatesText: event.target.value }))} disabled={busy} placeholder="Done" />
        </FormRow>
        <FormRow label="Runtime Image">
          <Input aria-label="Runtime Image" value={draft.image} onChange={(event) => setDraft((current) => ({ ...current, image: event.target.value }))} disabled={busy} />
        </FormRow>
        <FormRow label="Default Branch">
          <Input aria-label="Default Branch" value={draft.defaultRepoRevision} onChange={(event) => setDraft((current) => ({ ...current, defaultRepoRevision: event.target.value }))} disabled={busy} placeholder="main" />
        </FormRow>
        <FormRow label="Max Parallel">
          <Input aria-label="Max Parallel" value={draft.maxConcurrentItems} onChange={(event) => setDraft((current) => ({ ...current, maxConcurrentItems: event.target.value }))} disabled={busy} inputMode="numeric" />
        </FormRow>
      </div>

      <FormRow label="Repositories" hint={helperText.repositories}>
        <Textarea
          aria-label="Repositories"
          value={draft.repositoriesText}
          onChange={(event) => setDraft((current) => ({ ...current, repositoriesText: event.target.value }))}
          disabled={busy}
          rows={4}
          placeholder={'withakay/kocao\nwithakay/other-repo@main'}
        />
      </FormRow>

      <div className="flex justify-end">
        <Btn variant="primary" type="submit" disabled={busy}>
          {busy ? 'Saving…' : submitLabel}
        </Btn>
      </div>

      {error ? <ErrorBanner>{error}</ErrorBanner> : null}
      {localError ? <ErrorBanner>{localError}</ErrorBanner> : null}
    </form>
  )
}
