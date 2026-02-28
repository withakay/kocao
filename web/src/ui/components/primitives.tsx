import { cn } from '@/lib/utils'
import { useCallback, useState } from 'react'
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TdHTMLAttributes, TextareaHTMLAttributes, ThHTMLAttributes } from 'react'

/* ------------------------------------------------------------------ */
/*  Button                                                            */
/* ------------------------------------------------------------------ */

type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost'

const btnBase =
  'inline-flex items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed select-none whitespace-nowrap'

const btnVariants: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-primary-foreground hover:bg-primary/85 active:bg-primary/75',
  secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/70 active:bg-secondary/60',
  danger: 'bg-destructive/15 text-destructive hover:bg-destructive/25 active:bg-destructive/35 border border-destructive/20',
  ghost: 'text-muted-foreground hover:text-foreground hover:bg-secondary/60',
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
}

export function Btn({ variant = 'secondary', className, ...props }: ButtonProps) {
  return <button className={cn(btnBase, btnVariants[variant], className)} {...props} />
}

/* ------------------------------------------------------------------ */
/*  Link styled as button (for router Links passed as children)       */
/* ------------------------------------------------------------------ */

export function btnClass(variant: ButtonVariant = 'secondary') {
  return cn(btnBase, btnVariants[variant])
}

/* ------------------------------------------------------------------ */
/*  Input / Textarea / Select                                         */
/* ------------------------------------------------------------------ */

const fieldBase =
  'w-full rounded-md border border-input bg-background px-2.5 py-1.5 text-[15px] text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring/40 focus:border-ring'

export function Input({ className, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn(fieldBase, className)} {...props} />
}

export function Textarea({ className, ...props }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn(fieldBase, 'resize-y', className)} {...props} />
}

export function Select({ className, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className={cn(fieldBase, 'w-auto', className)} {...props} />
}

/* ------------------------------------------------------------------ */
/*  Card                                                              */
/* ------------------------------------------------------------------ */

export function Card({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <section className={cn('rounded-lg border border-border/60 bg-card p-2.5', className)}>
      {children}
    </section>
  )
}

export function CardHeader({ title, right }: { title: string; right?: ReactNode }) {
  return (
    <div className="flex items-center justify-between mb-1.5">
      <h2 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">{title}</h2>
      {right}
    </div>
  )
}

/* ------------------------------------------------------------------ */
/*  Form row — label + control                                        */
/* ------------------------------------------------------------------ */

export function FormRow({ label, children, hint }: { label: string; children: ReactNode; hint?: ReactNode }) {
  return (
    <div className="flex items-start gap-3 mb-1.5">
      <div className="text-sm text-muted-foreground w-24 shrink-0 pt-1.5 text-right">{label}</div>
      <div className="flex-1 min-w-0">
        {children}
        {hint ? <div className="mt-0.5 text-[11px] text-muted-foreground/70">{hint}</div> : null}
      </div>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/*  Inline label (read-only key-value pairs — NOT a button)           */
/* ------------------------------------------------------------------ */

export function FieldLabel({ children }: { children: ReactNode }) {
  return <span className="text-sm text-muted-foreground">{children}</span>
}

export function FieldValue({ children, mono = true }: { children: ReactNode; mono?: boolean }) {
  return <span className={cn('text-sm', mono && 'font-mono')}>{children}</span>
}

export function DetailRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-baseline gap-2 py-1">
      <div className="text-sm text-muted-foreground w-24 shrink-0 text-right">{label}</div>
      <div className="text-sm font-mono min-w-0 break-all">{children}</div>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/*  Error / notice banner                                             */
/* ------------------------------------------------------------------ */

export function ErrorBanner({ children }: { children: ReactNode }) {
  return (
    <div className="mt-2 rounded-md bg-destructive/10 border border-destructive/20 px-3 py-1.5 text-sm text-destructive">
      {children}
    </div>
  )
}

export function NoticeBanner({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-md bg-status-warn/10 border border-status-warn/20 px-3 py-1.5 text-sm text-status-warn">
      {children}
    </div>
  )
}

/* ------------------------------------------------------------------ */
/*  Badge / tag (non-interactive label with color)                    */
/* ------------------------------------------------------------------ */

type BadgeVariant = 'ok' | 'warn' | 'bad' | 'neutral' | 'info'

const badgeVariants: Record<BadgeVariant, string> = {
  ok: 'bg-status-ok/10 text-status-ok border-status-ok/20',
  warn: 'bg-status-warn/10 text-status-warn border-status-warn/20',
  bad: 'bg-destructive/10 text-destructive border-destructive/20',
  neutral: 'bg-muted/50 text-muted-foreground border-border/60',
  info: 'bg-primary/10 text-primary border-primary/20',
}

export function Badge({ variant = 'neutral', children }: { variant?: BadgeVariant; children: ReactNode }) {
  return (
    <span className={cn('inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium', badgeVariants[variant])}>
      {children}
    </span>
  )
}

/* ------------------------------------------------------------------ */
/*  Scope badge (small mono label for API scope hints)                */
/* ------------------------------------------------------------------ */

export function ScopeBadge({ scope }: { scope: string }) {
  return (
    <span className="text-[10px] font-mono text-muted-foreground/60 bg-muted/30 rounded px-1.5 py-0.5">
      {scope}
    </span>
  )
}

/* ------------------------------------------------------------------ */
/*  Table primitives                                                  */
/* ------------------------------------------------------------------ */

export function Table({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-[15px]" aria-label={label}>
        {children}
      </table>
    </div>
  )
}

export function Th({ children, className, ...props }: ThHTMLAttributes<HTMLTableCellElement>) {
  return <th className={cn('py-1 pr-3 text-xs font-medium text-muted-foreground text-left', className)} {...props}>{children}</th>
}

export function Td({ children, className, ...props }: TdHTMLAttributes<HTMLTableCellElement>) {
  return <td className={cn('py-1 pr-3 text-[15px]', className)} {...props}>{children}</td>
}

export function EmptyRow({ cols, loading, message }: { cols: number; loading: boolean; message: string }) {
  return (
    <tr>
      <td colSpan={cols} className="py-4 text-center text-sm text-muted-foreground">
        {loading ? 'Loading\u2026' : message}
      </td>
    </tr>
  )
}

/* ------------------------------------------------------------------ */
/*  CollapsibleSection                                                */
/* ------------------------------------------------------------------ */

type CollapsibleSectionProps = {
  /** Section title shown in the header */
  title: string
  /** Optional localStorage key to persist open/closed state */
  persistKey?: string
  /** Start expanded (default: true) */
  defaultOpen?: boolean
  /** Content */
  children: ReactNode
  /** Optional right-side content in header (e.g., action buttons) */
  headerRight?: ReactNode
  /** CSS class for the outer wrapper */
  className?: string
}

export function CollapsibleSection({
  title,
  persistKey,
  defaultOpen = true,
  children,
  headerRight,
  className,
}: CollapsibleSectionProps) {
  const getInitialOpen = (): boolean => {
    if (!persistKey) return defaultOpen
    try {
      const stored = localStorage.getItem(persistKey)
      return stored !== null ? stored === 'true' : defaultOpen
    } catch {
      return defaultOpen
    }
  }

  const [isOpen, setIsOpen] = useState(getInitialOpen)

  const toggle = useCallback(() => {
    setIsOpen((prev) => {
      const next = !prev
      if (persistKey) {
        try {
          localStorage.setItem(persistKey, String(next))
        } catch {
          // ignore localStorage errors
        }
      }
      return next
    })
  }, [persistKey])

  return (
    <section className={cn('rounded-lg border border-border/60 bg-card', className)}>
      {/* Header */}
      <div
        className="flex items-center gap-2 px-2.5 py-1.5 cursor-pointer hover:bg-secondary/30 transition-colors rounded-t-lg"
        onClick={toggle}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            toggle()
          }
        }}
        aria-expanded={isOpen}
      >
        {/* Chevron */}
        <svg
          className="shrink-0 transition-transform duration-200"
          style={{ transform: isOpen ? 'rotate(90deg)' : 'rotate(0deg)' }}
          width="12"
          height="12"
          viewBox="0 0 12 12"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          aria-hidden="true"
        >
          <path d="M4.5 2L8.5 6L4.5 10" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
        </svg>

        {/* Title */}
        <h2 className="flex-1 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
          {title}
        </h2>

        {/* Optional right content */}
        {headerRight ? (
          <div onClick={(e) => e.stopPropagation()}>
            {headerRight}
          </div>
        ) : null}
      </div>

      {/* Collapsible content */}
      <div
        className="grid transition-[grid-template-rows] duration-200 ease-in-out"
        style={{
          gridTemplateRows: isOpen ? '1fr' : '0fr',
        }}
      >
        <div className="overflow-hidden min-h-0">
          <div className="p-3 pt-0">
            {children}
          </div>
        </div>
      </div>
    </section>
  )
}
