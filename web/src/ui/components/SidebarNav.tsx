import { Link, useRouterState } from '@tanstack/react-router'
import { cn } from '@/lib/utils'

function isActive(locationPath: string, href: string) {
  return locationPath === href || locationPath.startsWith(href + '/')
}

export function SidebarNav() {
  const path = useRouterState({ select: (s) => s.location.pathname })

  const linkClass = (href: string) =>
    cn(
      'px-3 py-1.5 rounded-md text-xs font-medium transition-colors flex items-center gap-2',
      isActive(path, href)
        ? 'bg-primary/15 text-primary'
        : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
    )

  return (
    <nav className="flex flex-col gap-3 p-2 flex-1">
      <div>
        <div className="px-3 mb-1 text-[10px] font-medium uppercase tracking-[0.18em] text-muted-foreground/60">
          Workspace
        </div>
        <div className="flex flex-col gap-0.5">
          <Link className={linkClass('/workspace-sessions')} to="/workspace-sessions">
            <SessionsIcon />
            Sessions
          </Link>
          <Link className={linkClass('/harness-runs')} to="/harness-runs">
            <RunsIcon />
            Runs
          </Link>
          <Link className={linkClass('/cluster')} to="/cluster">
            <ClusterIcon />
            Cluster
          </Link>
        </div>
      </div>

      <div>
        <div className="px-3 mb-1 text-[10px] font-medium uppercase tracking-[0.18em] text-muted-foreground/60">
          Account
        </div>
        <div className="flex flex-col gap-0.5">
          <Link className={linkClass('/settings')} to="/settings">
            <UserIcon />
            Settings
          </Link>
        </div>
      </div>
    </nav>
  )
}

function SessionsIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M4 7h16M4 12h16M4 17h16" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
    </svg>
  )
}

function RunsIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M8 7h12M8 12h12M8 17h12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      <path d="M4 7h.01M4 12h.01M4 17h.01" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
    </svg>
  )
}

function ClusterIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M12 2l8 4v12l-8 4-8-4V6l8-4Z" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" />
      <path d="M12 22V12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      <path d="M20 6l-8 6-8-6" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" />
    </svg>
  )
}

function UserIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M20 21a8 8 0 1 0-16 0" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      <circle cx="12" cy="8" r="4" stroke="currentColor" strokeWidth="2" />
    </svg>
  )
}
