import { createContext, useContext, useCallback, useSyncExternalStore } from 'react'

/* ------------------------------------------------------------------ */
/*  Sidebar state — persisted in localStorage                         */
/* ------------------------------------------------------------------ */

const STORAGE_KEY = 'kocao.sidebar.collapsed'

function getSnapshot(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'true'
  } catch {
    return false
  }
}

function getServerSnapshot(): boolean {
  return false
}

const listeners = new Set<() => void>()

function subscribe(cb: () => void) {
  listeners.add(cb)
  return () => { listeners.delete(cb) }
}

function setCollapsed(val: boolean) {
  try {
    localStorage.setItem(STORAGE_KEY, String(val))
  } catch {
    // localStorage unavailable (private browsing, quota exceeded)
  }
  for (const cb of listeners) cb()
}

export function useSidebarCollapsed() {
  const collapsed = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
  const toggle = useCallback(() => setCollapsed(!getSnapshot()), [])
  return { collapsed, toggle, setCollapsed }
}

/* ------------------------------------------------------------------ */
/*  Command palette context                                           */
/* ------------------------------------------------------------------ */

export type PaletteContextValue = {
  open: boolean
  setOpen: (v: boolean) => void
}

export const PaletteContext = createContext<PaletteContextValue>({
  open: false,
  setOpen: () => { throw new Error('usePalette must be used within a PaletteContext.Provider') },
})

export function usePalette() {
  return useContext(PaletteContext)
}

/* ------------------------------------------------------------------ */
/*  Fullscreen context                                                */
/* ------------------------------------------------------------------ */

export type FullscreenContextValue = {
  fullscreen: boolean
  toggleFullscreen: () => void
}

export const FullscreenContext = createContext<FullscreenContextValue>({
  fullscreen: false,
  toggleFullscreen: () => {},
})

export function useFullscreen() {
  return useContext(FullscreenContext)
}
