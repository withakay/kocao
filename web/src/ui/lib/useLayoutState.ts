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
/*  Attach layout state — shared across sibling trees                 */
/* ------------------------------------------------------------------ */

const ATTACH_INSPECTOR_KEY = 'kocao.attach.inspectorOpen'
const ATTACH_ACTIVITY_KEY = 'kocao.attach.activityOpen'

type AttachLayoutState = {
  fullscreen: boolean
  inspectorOpen: boolean
  activityOpen: boolean
}

function readStoredBool(key: string, fallback = false): boolean {
  if (typeof window === 'undefined') return fallback
  try {
    return localStorage.getItem(key) === 'true'
  } catch {
    return fallback
  }
}

let attachLayoutState: AttachLayoutState = {
  fullscreen: false,
  inspectorOpen: readStoredBool(ATTACH_INSPECTOR_KEY),
  activityOpen: readStoredBool(ATTACH_ACTIVITY_KEY),
}

const attachLayoutListeners = new Set<() => void>()

function subscribeAttachLayout(cb: () => void) {
  attachLayoutListeners.add(cb)
  return () => { attachLayoutListeners.delete(cb) }
}

function getAttachLayoutSnapshot(): AttachLayoutState {
  return attachLayoutState
}

function getAttachLayoutServerSnapshot(): AttachLayoutState {
  return { fullscreen: false, inspectorOpen: false, activityOpen: false }
}

function setAttachFullscreen(val: boolean) {
  attachLayoutState = { ...attachLayoutState, fullscreen: val }
  for (const cb of attachLayoutListeners) cb()
}

function setAttachInspectorOpen(val: boolean) {
  attachLayoutState = { ...attachLayoutState, inspectorOpen: val }
  try {
    localStorage.setItem(ATTACH_INSPECTOR_KEY, String(val))
  } catch {
    // localStorage unavailable
  }
  for (const cb of attachLayoutListeners) cb()
}

function setAttachActivityOpen(val: boolean) {
  attachLayoutState = { ...attachLayoutState, activityOpen: val }
  try {
    localStorage.setItem(ATTACH_ACTIVITY_KEY, String(val))
  } catch {
    // localStorage unavailable
  }
  for (const cb of attachLayoutListeners) cb()
}

export function useAttachLayout() {
  const state = useSyncExternalStore(subscribeAttachLayout, getAttachLayoutSnapshot, getAttachLayoutServerSnapshot)

  const toggleFullscreen = useCallback(() => setAttachFullscreen(!state.fullscreen), [state.fullscreen])
  const toggleInspector = useCallback(() => setAttachInspectorOpen(!state.inspectorOpen), [state.inspectorOpen])
  const toggleActivity = useCallback(() => setAttachActivityOpen(!state.activityOpen), [state.activityOpen])

  return {
    fullscreen: state.fullscreen,
    inspectorOpen: state.inspectorOpen,
    activityOpen: state.activityOpen,
    setFullscreen: setAttachFullscreen,
    setInspectorOpen: setAttachInspectorOpen,
    setActivityOpen: setAttachActivityOpen,
    toggleFullscreen,
    toggleInspector,
    toggleActivity,
  }
}
