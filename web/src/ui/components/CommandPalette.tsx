import { Command } from 'cmdk'
import { useNavigate, useLocation } from 'react-router-dom'
import { usePalette, useSidebarCollapsed } from '../lib/useLayoutState'

export function CommandPalette() {
  const { open, setOpen } = usePalette()
  const navigate = useNavigate()
  const location = useLocation()
  const { collapsed, toggle: toggleSidebar } = useSidebarCollapsed()

  if (!open) return null

  const go = (path: string) => {
    navigate(path)
    setOpen(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[20vh]">
      {/* Backdrop — click to dismiss */}
      <button
        type="button"
        className="absolute inset-0 bg-background/60 backdrop-blur-sm cursor-default"
        onClick={() => setOpen(false)}
        aria-label="Close command palette"
        tabIndex={-1}
      />

      <div className="relative w-full max-w-md">
        <Command
          className="rounded-lg border border-border bg-card shadow-2xl overflow-hidden"
          onKeyDown={(e: React.KeyboardEvent) => {
            if (e.key === 'Escape') setOpen(false)
          }}
        >
          <Command.Input
            placeholder="Type a command..."
            className="w-full border-b border-border/40 bg-transparent px-4 py-3 text-sm text-foreground placeholder:text-muted-foreground outline-none"
            autoFocus
          />
          <Command.List className="max-h-64 overflow-y-auto p-1.5">
            <Command.Empty className="px-3 py-4 text-center text-xs text-muted-foreground">
              No results.
            </Command.Empty>

            <Command.Group heading="Navigation" className="[&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-[10px] [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-wider [&_[cmdk-group-heading]]:text-muted-foreground">
              <Item onSelect={() => go('/workspace-sessions')}>Go to Sessions</Item>
              <Item onSelect={() => go('/harness-runs')}>Go to Runs</Item>
            </Command.Group>

            <Command.Separator className="my-1 h-px bg-border/40" />

            <Command.Group heading="Layout" className="[&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:py-1.5 [&_[cmdk-group-heading]]:text-[10px] [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:uppercase [&_[cmdk-group-heading]]:tracking-wider [&_[cmdk-group-heading]]:text-muted-foreground">
              <Item onSelect={() => { toggleSidebar(); setOpen(false) }}>
                {collapsed ? 'Show Sidebar' : 'Hide Sidebar'}
              </Item>
            </Command.Group>
          </Command.List>

          <div className="border-t border-border/40 px-3 py-1.5 text-[10px] text-muted-foreground flex gap-3">
            <span><kbd className="font-mono">Esc</kbd> close</span>
            <span><kbd className="font-mono">&uarr;&darr;</kbd> navigate</span>
            <span><kbd className="font-mono">Enter</kbd> select</span>
          </div>
        </Command>
      </div>
    </div>
  )
}

function Item({ children, onSelect }: { children: React.ReactNode; onSelect: () => void }) {
  return (
    <Command.Item
      onSelect={onSelect}
      className="rounded-md px-3 py-2 text-sm text-foreground cursor-pointer aria-selected:bg-primary/10 aria-selected:text-primary transition-colors"
    >
      {children}
    </Command.Item>
  )
}
