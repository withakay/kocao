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
            <div className="brandSub">sessions • runs • PR outcomes</div>
          </div>
        </div>
        <nav className="nav">
          <NavLink
            className={() => (isActive(path, '/sessions') ? 'navLink navLinkActive' : 'navLink')}
            to="/sessions"
          >
            Sessions
          </NavLink>
          <NavLink className={() => (isActive(path, '/runs') ? 'navLink navLinkActive' : 'navLink')} to="/runs">
            Runs
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
