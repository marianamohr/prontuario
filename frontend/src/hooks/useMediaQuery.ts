import { useEffect, useState } from 'react'

/**
 * Retorna true quando a mídia corresponde à query (ex.: (max-width: 767px)).
 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window === 'undefined') return false
    return window.matchMedia(query).matches
  })

  useEffect(() => {
    const m = window.matchMedia(query)
    setMatches(m.matches)
    const listener = () => setMatches(m.matches)
    m.addEventListener('change', listener)
    return () => m.removeEventListener('change', listener)
  }, [query])

  return matches
}

export const BREAKPOINT_MOBILE = 768

/** True quando largura < 768px */
export function useIsMobile(): boolean {
  return useMediaQuery(`(max-width: ${BREAKPOINT_MOBILE - 1}px)`)
}
