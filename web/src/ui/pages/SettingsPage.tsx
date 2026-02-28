import { useEffect, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useAuth } from '../auth'
import { Topbar } from '../components/Topbar'
import { Badge, Btn, Card, CardHeader, ErrorBanner, FormRow, Input, ScopeBadge } from '../components/primitives'

export function SettingsPage() {
  const { token, remember, notice, setNotice, setRemember, setToken, clearToken } = useAuth()
  const [draft, setDraft] = useState(token)
  const [reveal, setReveal] = useState(false)

  useEffect(() => {
    setDraft(token)
  }, [token])

  const hasToken = token.trim() !== ''

  return (
    <>
      <Topbar title="Settings" subtitle="Authentication and local UI preferences." />

      <div className="flex-1 overflow-y-auto p-4 space-y-3">
        <Card>
          <CardHeader
            title="API Authentication"
            right={hasToken ? <Badge variant="ok">token set</Badge> : <Badge variant="warn">no token</Badge>}
          />

          <FormRow label="Token" hint={<span>Stored in {remember ? 'localStorage' : 'sessionStorage'} when saved.</span>}>
            <Input
              className="!text-xs"
              type={reveal ? 'text' : 'password'}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Bearer token"
              aria-label="API token"
            />
          </FormRow>

          <div className="flex items-center gap-3 mb-1.5">
            <label className="flex items-center gap-2 text-xs text-muted-foreground select-none cursor-pointer">
              <input
                type="checkbox"
                checked={remember}
                onChange={(e) => setRemember(e.target.checked)}
                aria-label="Remember token"
                className="accent-primary"
              />
              Remember across browser restarts
            </label>
          </div>

          <div className="flex items-center gap-2">
            <Btn variant="ghost" onClick={() => setReveal((v) => !v)} type="button">
              {reveal ? 'Hide' : 'Show'}
            </Btn>
            <Btn variant="primary" onClick={() => { setToken(draft); setNotice(null) }} type="button">
              Save
            </Btn>
            <Btn
              variant="danger"
              onClick={() => {
                clearToken()
                setDraft('')
                setNotice(null)
              }}
              type="button"
            >
              Clear
            </Btn>
            <div className="flex-1" />
            <Link to="/workspace-sessions" className="text-xs text-muted-foreground hover:text-foreground hover:underline">
              Back to sessions
            </Link>
          </div>

          {notice ? (
            <ErrorBanner>
              {notice}{' '}
              <button className="underline" type="button" onClick={() => setNotice(null)}>
                dismiss
              </button>
            </ErrorBanner>
          ) : null}

          <div className="mt-3 text-[11px] text-muted-foreground/80 leading-relaxed">
            <div className="mb-1">Required scopes for the UI:</div>
            <div className="flex flex-wrap gap-1.5">
              <ScopeBadge scope="workspace-session:read" />
              <ScopeBadge scope="workspace-session:write" />
              <ScopeBadge scope="harness-run:read" />
              <ScopeBadge scope="harness-run:write" />
              <ScopeBadge scope="control:write" />
            </div>
          </div>
        </Card>
      </div>
    </>
  )
}
