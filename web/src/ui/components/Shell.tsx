import { NavLink, Outlet, useLocation } from 'react-router-dom'

function isActive(locationPath: string, href: string) {
  return locationPath === href || locationPath.startsWith(href + '/')
}

export function Shell() {
  const loc = useLocation()
  const path = loc.pathname

  return (
    <div className="appShell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brandMark" />
          <div>
            <div className="brandTitle">kocao</div>
            <div className="brandSub">workspace sessions • harness runs • PR outcomes</div>
          </div>
        </div>
        <nav className="nav">
          <NavLink
            className={() => (isActive(path, '/workspace-sessions') ? 'navLink navLinkActive' : 'navLink')}
            to="/workspace-sessions"
          >
            Workspace Sessions
          </NavLink>
          <NavLink className={() => (isActive(path, '/harness-runs') ? 'navLink navLinkActive' : 'navLink')} to="/harness-runs">
            Harness Runs
          </NavLink>
        </nav>
        <div style={{ marginTop: 18 }} className="faint">
          API proxy: <span className="mono">/api</span> → <span className="mono">http://localhost:30080</span>
        </div>
      </aside>

      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
