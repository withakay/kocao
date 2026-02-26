import { useEffect } from 'react'

type ShortcutMap = Record<string, () => void>

const isMac = typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.userAgent)

/**
 * Register global keyboard shortcuts.
 *
 * Key format: 'mod+k' where 'mod' maps to Cmd (Mac) or Ctrl (non-Mac).
 * Only fires when no input/textarea is focused (unless the shortcut uses mod key).
 */
export function useKeyboardShortcuts(shortcuts: ShortcutMap) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const mod = isMac ? e.metaKey : e.ctrlKey
      const key = e.key.toLowerCase()

      for (const [combo, action] of Object.entries(shortcuts)) {
        const parts = combo.toLowerCase().split('+')
        const needsMod = parts.includes('mod')
        const targetKey = parts[parts.length - 1]

        if (needsMod && !mod) continue
        if (!needsMod && mod) continue
        if (key !== targetKey) continue

        // Allow mod shortcuts even when in inputs
        if (!needsMod) {
          const tag = (e.target as HTMLElement)?.tagName
          if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') continue
        }

        e.preventDefault()
        e.stopPropagation()
        action()
        return
      }
    }

    window.addEventListener('keydown', handler, { capture: true })
    return () => window.removeEventListener('keydown', handler, { capture: true })
  }, [shortcuts])
}
