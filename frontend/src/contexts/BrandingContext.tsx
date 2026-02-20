import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import type { Branding } from '../lib/api'
import * as api from '../lib/api'
import { useAuth } from './AuthContext'

const BRANDING_CACHE_KEY = 'prontuario-branding-cache'

function getCachedBranding(): Branding | null {
  try {
    const raw = sessionStorage.getItem(BRANDING_CACHE_KEY)
    return raw ? (JSON.parse(raw) as Branding) : null
  } catch {
    return null
  }
}

function setCachedBranding(b: Branding | null) {
  try {
    if (b) sessionStorage.setItem(BRANDING_CACHE_KEY, JSON.stringify(b))
    else sessionStorage.removeItem(BRANDING_CACHE_KEY)
  } catch {
    // ignore
  }
}

type BrandingState = {
  branding: Branding | null
  loading: boolean
  refetch: () => Promise<void>
}

const BrandingContext = createContext<BrandingState | null>(null)

export function BrandingProvider({ children }: { children: React.ReactNode }) {
  const { user } = useAuth()
  const userRoleRef = useRef(user?.role)
  userRoleRef.current = user?.role
  const [branding, setBranding] = useState<Branding | null>(getCachedBranding)
  const [loading, setLoading] = useState(false)

  const refetch = useCallback(async () => {
    if (userRoleRef.current !== 'PROFESSIONAL') {
      setBranding(null)
      setCachedBranding(null)
      return
    }
    setLoading(true)
    try {
      const b = await api.getBranding()
      setBranding(b)
      setCachedBranding(b)
    } catch {
      setBranding(null)
      setCachedBranding(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refetch()
  }, [refetch])

  const value = useMemo(
    () => ({ branding, loading, refetch }),
    [branding, loading, refetch]
  )

  return <BrandingContext.Provider value={value}>{children}</BrandingContext.Provider>
}

export function useBranding() {
  const ctx = useContext(BrandingContext)
  return ctx
}
