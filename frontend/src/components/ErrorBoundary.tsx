import { Component, type ErrorInfo, type ReactNode } from 'react'

type Props = { children: ReactNode; fallback?: ReactNode }
type State = { hasError: boolean; error?: Error }

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('ErrorBoundary:', error, errorInfo)
    try {
      const BASE = (import.meta as any)?.env?.VITE_API_URL ? String((import.meta as any).env.VITE_API_URL).replace(/\/$/, '') : ''
      if (!BASE) return
      const token = localStorage.getItem('token')
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (token) headers['Authorization'] = `Bearer ${token}`
      fetch(`${BASE}/api/errors/frontend`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          severity: 'ERROR',
          kind: 'REACT_ERROR_BOUNDARY',
          message: error?.message || 'react error',
          stack: error?.stack || undefined,
          metadata: { componentStack: errorInfo?.componentStack ? String(errorInfo.componentStack).slice(0, 4000) : undefined },
        }),
      }).catch(() => {})
    } catch {
      // no-op
    }
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) return this.props.fallback
      return (
        <div style={{ padding: '2rem', textAlign: 'center', color: '#6b7280' }}>
          <p>Algo deu errado ao carregar esta página.</p>
          <p style={{ fontSize: '0.9rem', marginTop: '0.5rem' }}>Tente novamente ou use o link que recebeu por e-mail.</p>
        </div>
      )
    }
    return this.props.children
  }
}
