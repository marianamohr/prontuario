import { describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { render, screen } from '@testing-library/react'
import { ProtectedRoute } from './ProtectedRoute'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: null,
    loading: false,
  }),
}))

describe('ProtectedRoute', () => {
  it('redirects unauthenticated users to /login', async () => {
    render(
      <MemoryRouter initialEntries={['/patients']}>
        <Routes>
          <Route
            path="/patients"
            element={
              <ProtectedRoute roles={['PROFESSIONAL']}>
                <div>OK</div>
              </ProtectedRoute>
            }
          />
          <Route path="/login" element={<div>LOGIN</div>} />
        </Routes>
      </MemoryRouter>,
    )
    expect(await screen.findByText('LOGIN')).toBeInTheDocument()
  })
})

