import { useMemo, useState, useEffect } from 'react'
import { Outlet, useRouterState } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import { useSidebarCollapsed, PaletteContext, useAttachLayout } from '../lib/useLayoutState'
import { useKeyboardShortcuts } from '../lib/useKeyboardShortcuts'
import { CommandPalette } from './CommandPalette'
import { SidebarNav } from './SidebarNav'

const SIDEBAR_STORAGE_KEY = 'kocao.sidebar.width'
const DEFAULT_SIDEBAR_WIDTH = 208 // w-52 in pixels
const MIN_SIDEBAR_WIDTH = 180
const MAX_SIDEBAR_WIDTH = 320


export function Shell() {
  const path = useRouterState({ select: (s) => s.location.pathname })
  const { collapsed, toggle: toggleSidebar } = useSidebarCollapsed()
  const [paletteOpen, setPaletteOpen] = useState(false)
  const { toggleInspector, toggleActivity } = useAttachLayout()
  
  const isAttachPage = path.includes('/attach')
  
  const [sidebarWidth, setSidebarWidth] = useState<number>(() => {
    try {
      const stored = localStorage.getItem(SIDEBAR_STORAGE_KEY)
      if (stored) {
        const parsed = Number(stored)
        if (!isNaN(parsed) && parsed >= MIN_SIDEBAR_WIDTH && parsed <= MAX_SIDEBAR_WIDTH) {
          return parsed
        }
      }
    } catch {
      // localStorage unavailable
    }
    return DEFAULT_SIDEBAR_WIDTH
  })

  const [isDragging, setIsDragging] = useState(false)

  // Persist sidebar width
  useEffect(() => {
    try {
      localStorage.setItem(SIDEBAR_STORAGE_KEY, String(sidebarWidth))
    } catch {
      // localStorage unavailable
    }
  }, [sidebarWidth])

  // Handle sidebar resize dragging
  useEffect(() => {
    if (!isDragging) return

    const handlePointerMove = (e: PointerEvent) => {
      const newWidth = e.clientX
      const clampedWidth = Math.max(MIN_SIDEBAR_WIDTH, Math.min(MAX_SIDEBAR_WIDTH, newWidth))
      setSidebarWidth(clampedWidth)
    }

    const handlePointerUp = () => {
      setIsDragging(false)
    }

    document.addEventListener('pointermove', handlePointerMove)
    document.addEventListener('pointerup', handlePointerUp)

    return () => {
      document.removeEventListener('pointermove', handlePointerMove)
      document.removeEventListener('pointerup', handlePointerUp)
    }
  }, [isDragging])

  const handleResizePointerDown = (e: React.PointerEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }

  // Escape is handled by CommandPalette itself to avoid double-firing
  const shortcuts = useMemo(() => {
    const base = {
      'mod+k': () => setPaletteOpen((v) => !v),
      'mod+\\': () => toggleSidebar(),
    }
    
    // Add attach-specific shortcuts only on attach page
    if (isAttachPage) {
      return {
        ...base,
        'mod+i': () => toggleInspector(),
        'mod+j': () => toggleActivity(),
      }
    }
    
    return base
  }, [toggleSidebar, isAttachPage, toggleInspector, toggleActivity])

  useKeyboardShortcuts(shortcuts)


  const paletteCtx = useMemo(() => ({
    open: paletteOpen,
    setOpen: setPaletteOpen,
  }), [paletteOpen])

  return (
    <PaletteContext.Provider value={paletteCtx}>
      <div className="flex h-screen overflow-hidden">
        {/* Sidebar */}
        {!collapsed && (
          <>
            <aside
              style={{ width: `${sidebarWidth}px` }}
              className="shrink-0 border-r border-border/60 bg-sidebar flex flex-col overflow-hidden"
            >
              <div className="flex items-center gap-2.5 px-4 py-3 border-b border-border/40">
                <div className="size-6 shrink-0 rounded bg-primary/20 border border-primary/30" />
                <div className="min-w-0">
                  <div className="text-xs font-semibold tracking-tight text-sidebar-foreground leading-none">kocao</div>
                  <div className="text-[10px] text-muted-foreground leading-tight">agent orchestration</div>
                </div>
              </div>

              <SidebarNav />

              <div className="px-4 py-2 text-[10px] text-muted-foreground/50 font-mono border-t border-border/40">
                api: localhost:30080
              </div>
            </aside>

            {/* Resize handle */}
            <div
              onPointerDown={handleResizePointerDown}
              className={cn(
                'shrink-0 w-1 cursor-col-resize transition-colors',
                isDragging
                  ? 'bg-primary'
                  : 'bg-border/60 hover:bg-primary/60',
              )}
              style={{ touchAction: 'none' }}
            />
          </>
        )}

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
