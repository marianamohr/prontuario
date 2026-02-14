import { useAuth } from '../contexts/AuthContext'
import * as api from '../lib/api'

export function ImpersonateBanner() {
  const { user, isImpersonated } = useAuth()

  if (!user || !isImpersonated) return null

  const handleEnd = async () => {
    try {
      await api.impersonateEnd()

      // Restaura sessão do admin (token + user) e remove a flag.
      const adminToken = localStorage.getItem('impersonate_admin_token')
      const adminUser = localStorage.getItem('impersonate_admin_user')
      if (adminToken) localStorage.setItem('token', adminToken)
      if (adminUser) localStorage.setItem('user', adminUser)
      localStorage.removeItem('impersonate_admin_token')
      localStorage.removeItem('impersonate_admin_user')
      localStorage.removeItem('impersonating')
      window.location.reload()
    } catch {
      alert('Falha ao encerrar impersonate.')
    }
  }

  return (
    <div
      style={{
        background: '#f59e0b',
        color: '#1a1a2e',
        padding: '0.5rem 1rem',
        textAlign: 'center',
        fontWeight: 600,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '1rem',
      }}
    >
      <span>Modo suporte (impersonate) – você está atuando como outro usuário.</span>
      <button
        type="button"
        onClick={handleEnd}
        style={{
          background: '#1a1a2e',
          color: 'white',
          border: 'none',
          padding: '0.35rem 0.75rem',
          borderRadius: 4,
          fontWeight: 600,
          cursor: 'pointer',
        }}
      >
        Encerrar impersonate
      </button>
    </div>
  )
}
