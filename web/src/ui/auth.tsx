import React, { createContext, useCallback, useContext, useMemo, useState } from 'react'

const tokenStorageKey = 'kocao.apiToken'

type AuthState = {
  token: string
  remember: boolean
  notice: string | null
  setNotice: (v: string | null) => void
  setRemember: (v: boolean) => void
  setToken: (v: string) => void
  clearToken: () => void
  invalidateToken: (reason: string) => void
}

const AuthContext = createContext<AuthState | null>(null)

function safeGet(s: Storage | undefined, k: string): string {
  try {
    return s?.getItem(k) ?? ''
  } catch {
    return ''
  }
}

function safeSet(s: Storage | undefined, k: string, v: string) {
  try {
    s?.setItem(k, v)
  } catch {
    // ignore
  }
}

function safeRemove(s: Storage | undefined, k: string) {
  try {
    s?.removeItem(k)
  } catch {
    // ignore
  }
}

function readStored(): { token: string; remember: boolean } {
  const st = safeGet(typeof sessionStorage !== 'undefined' ? sessionStorage : undefined, tokenStorageKey)
  if (st.trim() !== '') return { token: st, remember: false }

  const lt = safeGet(typeof localStorage !== 'undefined' ? localStorage : undefined, tokenStorageKey)
  if (lt.trim() !== '') return { token: lt, remember: true }

  return { token: '', remember: false }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setTokenState] = useState(() => readStored().token)
  const [remember, setRememberState] = useState(() => readStored().remember)
  const [notice, setNotice] = useState<string | null>(null)

  const persist = useCallback((nextToken: string, nextRemember: boolean) => {
    const ls = typeof localStorage !== 'undefined' ? localStorage : undefined
    const ss = typeof sessionStorage !== 'undefined' ? sessionStorage : undefined

    if (nextToken.trim() === '') {
      safeRemove(ls, tokenStorageKey)
      safeRemove(ss, tokenStorageKey)
      return
    }

    if (nextRemember) {
      safeSet(ls, tokenStorageKey, nextToken)
      safeRemove(ss, tokenStorageKey)
    } else {
      safeSet(ss, tokenStorageKey, nextToken)
      safeRemove(ls, tokenStorageKey)
    }
  }, [])

  const setRemember = useCallback(
    (v: boolean) => {
      setRememberState(v)
      persist(token, v)
    },
    [persist, token]
  )

  const setToken = useCallback(
    (v: string) => {
      setTokenState(v)
      setNotice(null)
      persist(v, remember)
    },
    [persist, remember]
  )

  const clearToken = useCallback(() => {
    setTokenState('')
    setRememberState(false)
    persist('', false)
  }, [persist])

  const invalidateToken = useCallback(
    (reason: string) => {
      setNotice(reason)
      setTokenState('')
      setRememberState(false)
      persist('', false)
    },
    [persist]
  )

  const value = useMemo(
    () => ({ token, remember, notice, setNotice, setRemember, setToken, clearToken, invalidateToken }),
    [token, remember, notice, setRemember, setToken, clearToken, invalidateToken]
  )
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const v = useContext(AuthContext)
  if (!v) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return v
}
