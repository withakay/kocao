import React, { createContext, useCallback, useContext, useMemo, useState } from 'react'

const tokenStorageKey = 'kocao.apiToken'

type AuthState = {
  token: string
  setToken: (v: string) => void
}

const AuthContext = createContext<AuthState | null>(null)

function readStoredToken(): string {
  try {
    return localStorage.getItem(tokenStorageKey) ?? ''
  } catch {
    return ''
  }
}

function writeStoredToken(v: string) {
  try {
    localStorage.setItem(tokenStorageKey, v)
  } catch {
    // ignore
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setTokenState] = useState(() => readStoredToken())
  const setToken = useCallback((v: string) => {
    setTokenState(v)
    writeStoredToken(v)
  }, [])
  const value = useMemo(() => ({ token, setToken }), [token, setToken])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const v = useContext(AuthContext)
  if (!v) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return v
}
