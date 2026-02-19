import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import type { Branding } from '../lib/api'
import * as api from '../lib/api'
import { useAuth } from './AuthContext'

type BrandingState = {
  branding: Branding | null
  loading: boolean
  refetch: () => Promise<void>
}

const BrandingContext = createContext<BrandingState | null>(null)

export function BrandingProvider({ children }: { children: React.ReactNode }) {
  const { user } = useAuth()
  const [branding, setBranding] = useState<Branding | null>(null)
  const [loading, setLoading] = useState(false)

  const refetch = useCallback(async () => {
    if (user?.role !== 'PROFESSIONAL') {
      setBranding(null)
      return
    }
    setLoading(true)
    try {
      const b = await api.getBranding()
      setBranding(b)
    } catch {
      setBranding(null)
    } finally {
      setLoading(false)
    }
  }, [user?.role])

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
