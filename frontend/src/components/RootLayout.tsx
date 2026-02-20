import { Outlet, useLocation, Navigate } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { AppShell } from './ui/AppShell'

export function RootLayout() {
  const { pathname } = useLocation()
  const { user } = useAuth()

  if (pathname === '/') {
    if (user) return <Navigate to="/home" replace />
    return <Outlet />
  }

  if (pathname === '/login') {
    return <Outlet />
  }

  return (
    <AppShell>
      <Outlet />
    </AppShell>
  )
}
