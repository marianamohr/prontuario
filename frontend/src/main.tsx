import React from 'react'
import ReactDOM from 'react-dom/client'
import { CssBaseline } from '@mui/material'
import App from './App'
import { ThemeSettingsProvider } from './contexts/ThemeSettingsContext'
import './index.css'

function setupGlobalFrontendErrorHandlers() {
  try {
    const BASE = (import.meta as any)?.env?.VITE_API_URL ? String((import.meta as any).env.VITE_API_URL).replace(/\/$/, '') : ''
    if (!BASE) return
    const post = (payload: any) => {
      try {
        const token = localStorage.getItem('token')
        const headers: Record<string, string> = { 'Content-Type': 'application/json' }
        if (token) headers['Authorization'] = `Bearer ${token}`
        fetch(`${BASE}/api/errors/frontend`, { method: 'POST', headers, body: JSON.stringify(payload) }).catch(() => {})
      } catch {
        // no-op
      }
    }
    window.addEventListener('error', (e) => {
      post({
        severity: 'ERROR',
        kind: 'WINDOW_ERROR',
        message: String((e as any)?.message || 'window error'),
        stack: (e as any)?.error?.stack,
        metadata: { filename: (e as any)?.filename, lineno: (e as any)?.lineno, colno: (e as any)?.colno },
      })
    })
    window.addEventListener('unhandledrejection', (e) => {
      const reason = (e as any)?.reason
      post({
        severity: 'ERROR',
        kind: 'UNHANDLED_REJECTION',
        message: String(reason?.message || reason || 'unhandled rejection'),
        stack: reason?.stack,
      })
    })
  } catch {
    // no-op
  }
}

setupGlobalFrontendErrorHandlers()

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ThemeSettingsProvider>
      <CssBaseline />
      <App />
    </ThemeSettingsProvider>
  </React.StrictMode>,
)
