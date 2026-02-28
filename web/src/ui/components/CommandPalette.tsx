import { Command } from 'cmdk'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { usePalette, useSidebarCollapsed, useAttachLayout } from '../lib/useLayoutState'

export function CommandPalette() {
  const { open, setOpen } = usePalette()
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const { collapsed, toggle: toggleSidebar } = useSidebarCollapsed()
  const { toggleFullscreen: toggleAttachFullscreen, toggleInspector, toggleActivity } = useAttachLayout()

  if (!open) return null

  const go = (to: '/workspace-sessions' | '/harness-runs' | '/cluster' | '/settings') => {
    navigate({ to })
    setOpen(false)
  }

  // Check if we're on the attach page (where fullscreen is available)
  const isAttachPage = pathname.includes('/attach')

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[18vh]">
      {/* Backdrop — click to dismiss */}
      <button
        type="button"
        className="absolute inset-0 bg-black/70 backdrop-blur-xl cursor-default"
        onClick={() => setOpen(false)}
        aria-label="Close command palette"
        tabIndex={-1}
      />

      <div className="relative w-full max-w-lg">
        <Command
          className="rounded-xl border border-border/60 bg-card/95 shadow-2xl shadow-black/50 overflow-hidden"
          onKeyDown={(e: React.KeyboardEvent) => {
            if (e.key === 'Escape') setOpen(false)
          }}
        >
          <Command.Input
            placeholder="Type a command or search..."
            className="w-full border-b border-border/60 bg-transparent px-5 py-3.5 text-base text-foreground/85 placeholder:text-muted-foreground/60 outline-none"
            autoFocus
          />
          <Command.List className="max-h-64 overflow-y-auto p-1.5">
            <Command.Empty className="px-4 py-4 text-center text-xs text-muted-foreground/70">
              No results.
            </Command.Empty>

            <Command.Group heading="Navigation" className="[&_[cmdk-group-heading]]:px-4 [&_[cmdk-group-heading]]:py-2 [&_[cmdk-group-heading]]:text-[10px] [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-[0.15em] [&_[cmdk-group-heading]]:text-muted-foreground/60">
              <Item onSelect={() => go('/workspace-sessions')} icon={<ArrowRightIcon />}>
                Go to Sessions
              </Item>
              <Item onSelect={() => go('/harness-runs')} icon={<ArrowRightIcon />}>
                Go to Runs
              </Item>
              <Item onSelect={() => go('/cluster')} icon={<ArrowRightIcon />}>
                Go to Cluster
              </Item>
              <Item onSelect={() => go('/settings')} icon={<ArrowRightIcon />}>
                Go to Settings
              </Item>
            </Command.Group>

            <Command.Separator className="my-1 h-px bg-border/40" />

            <Command.Group heading="Layout" className="[&_[cmdk-group-heading]]:px-4 [&_[cmdk-group-heading]]:py-2 [&_[cmdk-group-heading]]:text-[10px] [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-[0.15em] [&_[cmdk-group-heading]]:text-muted-foreground/60">
              <Item onSelect={() => { toggleSidebar(); setOpen(false) }} icon={<PanelsIcon />} shortcut="Cmd+\\">
                Toggle Sidebar
              </Item>
              {isAttachPage && (
                <>
                  <Item onSelect={() => { toggleAttachFullscreen(); setOpen(false) }} icon={<PanelsIcon />}>
                    Toggle Fullscreen
                  </Item>
                  <Item onSelect={() => { toggleInspector(); setOpen(false) }} icon={<PanelsIcon />} shortcut="Cmd+I">
                    Toggle Inspector
                  </Item>
                  <Item onSelect={() => { toggleActivity(); setOpen(false) }} icon={<PanelsIcon />} shortcut="Cmd+J">
                    Toggle Activity Panel
                  </Item>
                </>
              )}
            </Command.Group>

            <Command.Separator className="my-1 h-px bg-border/40" />

            <Command.Group heading="Actions" className="[&_[cmdk-group-heading]]:px-4 [&_[cmdk-group-heading]]:py-2 [&_[cmdk-group-heading]]:text-[10px] [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-[0.15em] [&_[cmdk-group-heading]]:text-muted-foreground/60">
              <Item onSelect={() => { window.location.reload() }} icon={<RefreshIcon />} shortcut="Cmd+R">
                Refresh Data
              </Item>
            </Command.Group>
          </Command.List>

          <div className="border-t border-border/60 px-4 py-2 text-[10px] text-muted-foreground/70 flex gap-3">
            <span><kbd className="text-[10px] font-mono bg-muted/50 border border-border/60 rounded px-1.5 py-0.5 text-muted-foreground/80">Esc</kbd> close</span>
            <span><kbd className="text-[10px] font-mono bg-muted/50 border border-border/60 rounded px-1.5 py-0.5 text-muted-foreground/80">&uarr;&darr;</kbd> navigate</span>
            <span><kbd className="text-[10px] font-mono bg-muted/50 border border-border/60 rounded px-1.5 py-0.5 text-muted-foreground/80">Enter</kbd> select</span>
          </div>
        </Command>
      </div>
    </div>
  )
}

function Item({ children, onSelect, icon, shortcut }: { children: React.ReactNode; onSelect: () => void; icon?: React.ReactNode; shortcut?: string }) {
  return (
    <Command.Item
      onSelect={onSelect}
      className="rounded-lg px-4 py-2.5 text-sm text-foreground/85 cursor-pointer aria-selected:bg-secondary/70 aria-selected:text-foreground transition-colors flex items-center gap-3"
    >
      {icon && <span className="shrink-0">{icon}</span>}
      <span className="flex-1">{children}</span>
      {shortcut && <kbd className="text-[10px] font-mono bg-muted/50 border border-border/60 rounded px-1.5 py-0.5 text-muted-foreground/80">{shortcut}</kbd>}
    </Command.Item>
  )
}

function ArrowRightIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M6 12L10 8L6 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function PanelsIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="2" y="3" width="5" height="10" rx="1" stroke="currentColor" strokeWidth="1.5" />
      <rect x="9" y="3" width="5" height="10" rx="1" stroke="currentColor" strokeWidth="1.5" />
    </svg>
  )
}

function RefreshIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M13 8C13 10.7614 10.7614 13 8 13C5.23858 13 3 10.7614 3 8C3 5.23858 5.23858 3 8 3C9.36 3 10.59 3.52 11.5 4.36M11.5 4.36V2M11.5 4.36H9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
