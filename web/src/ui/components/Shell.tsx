import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { cn } from '@/lib/utils'

function isActive(locationPath: string, href: string) {
  return locationPath === href || locationPath.startsWith(href + '/')
}

export function Shell() {
  const loc = useLocation()
  const path = loc.pathname

  return (
    <div className="shell-grid grid min-h-screen" style={{ gridTemplateColumns: '240px 1fr' }}>
      <aside className="shell-sidebar border-r border-border bg-sidebar p-5">
        <div className="flex items-center gap-3 mb-4">
          <div
            className="size-8 shrink-0 rounded-md bg-primary/20 border border-primary/30"
          />
          <div>
            <div className="text-sm font-semibold tracking-tight text-sidebar-foreground">kocao</div>
            <div className="text-xs text-muted-foreground">k8s agent orchestration</div>
          </div>
        </div>
        <nav className="flex flex-col gap-1 mt-4">
          <NavLink
            className={() =>
              cn(
                'px-3 py-2 rounded-md text-sm transition-colors border border-transparent',
                isActive(path, '/workspace-sessions')
                  ? 'bg-sidebar-accent border-primary/20 text-sidebar-accent-foreground'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
              )
            }
            to="/workspace-sessions"
           >
            Sessions
          </NavLink>
          <NavLink
            className={() =>
              cn(
                'px-3 py-2 rounded-md text-sm transition-colors border border-transparent',
                isActive(path, '/harness-runs')
                  ? 'bg-sidebar-accent border-primary/20 text-sidebar-accent-foreground'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
              )
            }
            to="/harness-runs"
          >
            Runs
          </NavLink>
        </nav>
        <div className="mt-5 text-xs text-muted-foreground font-mono">
          /api → localhost:30080
        </div>
      </aside>

      <main className="p-6">
        <Outlet />
      </main>
    </div>
  )
}
