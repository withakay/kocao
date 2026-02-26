import { useMemo, useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { useSidebarCollapsed, PaletteContext } from '../lib/useLayoutState'
import { useKeyboardShortcuts } from '../lib/useKeyboardShortcuts'
import { CommandPalette } from './CommandPalette'

function isActive(locationPath: string, href: string) {
  return locationPath === href || locationPath.startsWith(href + '/')
}

export function Shell() {
  const loc = useLocation()
  const path = loc.pathname
  const { collapsed, toggle: toggleSidebar } = useSidebarCollapsed()
  const [paletteOpen, setPaletteOpen] = useState(false)

  // Escape is handled by CommandPalette itself to avoid double-firing
  const shortcuts = useMemo(() => ({
    'mod+k': () => setPaletteOpen((v) => !v),
    'mod+\\': () => toggleSidebar(),
  }), [toggleSidebar])

  useKeyboardShortcuts(shortcuts)

  const linkClass = (href: string) =>
    cn(
      'px-3 py-1.5 rounded-md text-xs font-medium transition-colors',
      isActive(path, href)
        ? 'bg-primary/15 text-primary'
        : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
    )

  const paletteCtx = useMemo(() => ({
    open: paletteOpen,
    setOpen: setPaletteOpen,
  }), [paletteOpen])

  return (
    <PaletteContext.Provider value={paletteCtx}>
      <div className="flex h-screen overflow-hidden">
        {/* Sidebar */}
        <aside
          className={cn(
            'shrink-0 border-r border-border/60 bg-sidebar flex flex-col transition-[width] duration-200 overflow-hidden',
            collapsed ? 'w-0 border-r-0' : 'w-52',
          )}
        >
          <div className="flex items-center gap-2.5 px-4 py-3 border-b border-border/40">
            <div className="size-6 shrink-0 rounded bg-primary/20 border border-primary/30" />
            <div className="min-w-0">
              <div className="text-xs font-semibold tracking-tight text-sidebar-foreground leading-none">kocao</div>
              <div className="text-[10px] text-muted-foreground leading-tight">agent orchestration</div>
            </div>
          </div>

          <nav className="flex flex-col gap-0.5 p-2 flex-1">
            <NavLink className={() => linkClass('/workspace-sessions')} to="/workspace-sessions">
              Sessions
            </NavLink>
            <NavLink className={() => linkClass('/harness-runs')} to="/harness-runs">
              Runs
            </NavLink>
          </nav>

          <div className="px-4 py-2 text-[10px] text-muted-foreground/50 font-mono border-t border-border/40">
            api: localhost:30080
          </div>
        </aside>

        {/* Collapsed sidebar toggle */}
        {collapsed && (
          <button
            type="button"
            className="shrink-0 w-8 flex items-center justify-center border-r border-border/60 bg-sidebar hover:bg-secondary/40 transition-colors text-muted-foreground hover:text-foreground"
            onClick={toggleSidebar}
            aria-label="Expand sidebar"
            title="Expand sidebar (Cmd+\)"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <polyline points="9 18 15 12 9 6" />
            </svg>
          </button>
        )}

        {/* Main content area */}
        <main className="flex-1 flex flex-col overflow-hidden">
          <Outlet />
        </main>
      </div>

      <CommandPalette />
    </PaletteContext.Provider>
  )
}
