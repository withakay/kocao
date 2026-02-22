import { useState } from 'react'
import { useAuth } from '../auth'

export function Topbar({ title, subtitle }: { title: string; subtitle: string }) {
  const { token, setToken } = useAuth()
  const [draft, setDraft] = useState(token)
  const [reveal, setReveal] = useState(false)

  return (
    <div className="topbar">
      <div className="topbarTitle">
        <h1>{title}</h1>
        <p>{subtitle}</p>
      </div>
      <div className="tokenBox">
        <input
          className="input"
          type={reveal ? 'text' : 'password'}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder="Bearer token (required)"
          aria-label="API token"
        />
        <button className="btn" onClick={() => setReveal((v) => !v)} type="button">
          {reveal ? 'Hide' : 'Show'}
        </button>
        <button className="btn btnPrimary" onClick={() => setToken(draft)} type="button">
          Save
        </button>
      </div>
    </div>
  )
}
