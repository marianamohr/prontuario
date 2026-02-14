import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import type { User } from '../lib/api'
import * as api from '../lib/api'

type AuthState = {
  user: User | null
  token: string | null
  loading: boolean
  isImpersonated: boolean
  login: (token: string, user: User) => void
  logout: () => void
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

const TOKEN_KEY = 'token'
const USER_KEY = 'user'

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(() => {
    try {
      const u = localStorage.getItem(USER_KEY)
      return u ? JSON.parse(u) : null
    } catch {
      return null
    }
  })
  const [token, setToken] = useState<string | null>(() => localStorage.getItem(TOKEN_KEY))
  const [loading, setLoading] = useState(!!token)

  const login = useCallback((t: string, u: User) => {
    localStorage.setItem(TOKEN_KEY, t)
    localStorage.setItem(USER_KEY, JSON.stringify(u))
    setToken(t)
    setUser(u)
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    setToken(null)
    setUser(null)
  }, [])

  const refresh = useCallback(async () => {
    if (!token) return
    try {
      const u = await api.me()
      setUser(u)
      localStorage.setItem(USER_KEY, JSON.stringify(u))
    } catch {
      logout()
    }
  }, [token, logout])

  useEffect(() => {
    if (!token) {
      setLoading(false)
      return
    }
    api
      .me()
      .then((u) => {
        setUser(u)
        localStorage.setItem(USER_KEY, JSON.stringify(u))
      })
      .catch(() => logout())
      .finally(() => setLoading(false))
  }, [token, logout])

  const isImpersonated = !!user && (localStorage.getItem('impersonating') === '1')

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        loading,
        isImpersonated,
        login,
        logout,
        refresh,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
