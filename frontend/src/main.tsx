import ReactDOM from 'react-dom/client'
import { CssBaseline } from '@mui/material'
import App from './App'
import { ThemeSettingsProvider } from './contexts/ThemeSettingsContext'
import './index.css'

type FrontendErrorPayload = {
  severity: string
  kind: string
  message: string
  stack?: string
  path?: string
  metadata?: Record<string, unknown>
}

function setupGlobalFrontendErrorHandlers() {
  try {
    const BASE = import.meta.env?.VITE_API_URL ? String(import.meta.env.VITE_API_URL).replace(/\/$/, '') : ''
    if (!BASE) return
    const post = (payload: FrontendErrorPayload) => {
      try {
        const token = localStorage.getItem('token')
        const headers: Record<string, string> = { 'Content-Type': 'application/json' }
        if (token) headers['Authorization'] = `Bearer ${token}`
        fetch(`${BASE}/api/errors/frontend`, { method: 'POST', headers, body: JSON.stringify(payload) }).catch(() => {})
      } catch {
        // no-op
      }
    }
    const pagePath = typeof window?.location?.pathname === 'string' ? window.location.pathname : ''
    window.addEventListener('error', (e: ErrorEvent) => {
      post({
        severity: 'ERROR',
        kind: 'WINDOW_ERROR',
        message: String(e?.message || 'window error'),
        stack: e?.error?.stack,
        path: pagePath || undefined,
        metadata: { filename: e?.filename, lineno: e?.lineno, colno: e?.colno },
      })
    })
    window.addEventListener('unhandledrejection', (e: PromiseRejectionEvent) => {
      const reason = e?.reason as Error | undefined
      post({
        severity: 'ERROR',
        kind: 'UNHANDLED_REJECTION',
        message: String(reason?.message ?? reason ?? 'unhandled rejection'),
        stack: reason?.stack,
        path: pagePath || undefined,
      })
    })
  } catch {
    // no-op
  }
}

setupGlobalFrontendErrorHandlers()

// StrictMode disabled: in dev it double-mounts the tree, causing Auth/Branding providers to remount
// and route components (e.g. Appearance, ScheduleConfig) to unmount/remount and appear to freeze.
ReactDOM.createRoot(document.getElementById('root')!).render(
  <ThemeSettingsProvider>
    <CssBaseline />
    <App />
  </ThemeSettingsProvider>,
)
