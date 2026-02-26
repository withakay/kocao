import { useEffect } from 'react'
import { cn } from '@/lib/utils'
import { Btn } from './primitives'

type InspectorPanelProps = {
  open: boolean
  title: string
  onClose: () => void
  children: React.ReactNode
  width?: string
}

export function InspectorPanel({ open, title, onClose, children, width = 'w-80' }: InspectorPanelProps) {
  // Handle Escape key to close
  useEffect(() => {
    if (!open) return
    
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    
    window.addEventListener('keydown', handleEscape)
    return () => window.removeEventListener('keydown', handleEscape)
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      className={cn(
        'fixed top-0 right-0 h-screen border-l border-border/60 bg-card shadow-2xl z-40',
        'transition-transform duration-200 ease-in-out',
        width,
        open ? 'translate-x-0' : 'translate-x-full'
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-border/40 bg-card/50">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          {title}
        </h2>
        <Btn variant="ghost" onClick={onClose} type="button" className="text-xs px-2 py-1">
          Close
        </Btn>
      </div>

      {/* Content */}
      <div className="overflow-y-auto h-[calc(100vh-48px)] p-3">
        {children}
      </div>
    </div>
  )
}
