import { useEffect, useRef, useState } from 'react'
import { cn } from '@/lib/utils'

type ResizablePanelProps = {
  /** Unique ID for localStorage persistence */
  id: string
  /** Direction of the split */
  direction: 'horizontal' | 'vertical'
  /** Default size of the first panel (px or %) */
  defaultSize?: number
  /** Min size in px for first panel */
  minSize?: number
  /** Max size in px for first panel */
  maxSize?: number
  /** First panel content */
  children: [React.ReactNode, React.ReactNode]
  /** Called when panel resizes */
  onResize?: () => void
  /** CSS class for outer container */
  className?: string
}

export function ResizablePanel({
  id,
  direction,
  defaultSize = 200,
  minSize = 100,
  maxSize = 500,
  children,
  onResize,
  className,
}: ResizablePanelProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [size, setSize] = useState<number>(() => {
    try {
      const stored = localStorage.getItem(`kocao.panel.${id}`)
      if (stored) {
        const parsed = Number(stored)
        if (!isNaN(parsed) && parsed >= minSize && parsed <= maxSize) {
          return parsed
        }
      }
    } catch {
      // localStorage unavailable
    }
    return defaultSize
  })

  const [isDragging, setIsDragging] = useState(false)

  useEffect(() => {
    try {
      localStorage.setItem(`kocao.panel.${id}`, String(size))
    } catch {
      // localStorage unavailable
    }
  }, [id, size])

  useEffect(() => {
    if (!isDragging) return

    const handlePointerMove = (e: PointerEvent) => {
      if (!containerRef.current) return

      const rect = containerRef.current.getBoundingClientRect()
      let newSize: number

      if (direction === 'vertical') {
        newSize = e.clientY - rect.top
      } else {
        newSize = e.clientX - rect.left
      }

      newSize = Math.max(minSize, Math.min(maxSize, newSize))
      setSize(newSize)
      onResize?.()
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
  }, [isDragging, direction, minSize, maxSize, onResize])

  const handlePointerDown = (e: React.PointerEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const isVertical = direction === 'vertical'

  return (
    <div
      ref={containerRef}
      className={cn('flex', isVertical ? 'flex-col' : 'flex-row', className)}
    >
      {/* First panel */}
      <div
        style={{
          [isVertical ? 'height' : 'width']: `${size}px`,
        }}
        className="shrink-0 overflow-hidden"
      >
        {children[0]}
      </div>

      {/* Resize handle */}
      <div
        onPointerDown={handlePointerDown}
        className={cn(
          'shrink-0 group relative transition-colors',
          isVertical ? 'h-1 cursor-row-resize' : 'w-1 cursor-col-resize',
          isDragging
            ? 'bg-primary'
            : 'bg-border/60 hover:bg-primary/60',
        )}
        style={{ touchAction: 'none' }}
      />

      {/* Second panel */}
      <div className="flex-1 min-h-0 min-w-0 overflow-hidden">
        {children[1]}
      </div>
    </div>
  )
}
