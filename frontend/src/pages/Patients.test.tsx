import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { Patients } from './Patients'

const listPatients = vi.fn()
const createPatient = vi.fn()
const createPatientInvite = vi.fn()
const softDeletePatient = vi.fn()
const getPatient = vi.fn()
const updatePatient = vi.fn()
const softDeleteGuardian = vi.fn()

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return {
    ...actual,
    listPatients: (...args: unknown[]) => listPatients(...args),
    createPatient: (...args: unknown[]) => createPatient(...args),
    createPatientInvite: (...args: unknown[]) => createPatientInvite(...args),
    softDeletePatient: (...args: unknown[]) => softDeletePatient(...args),
    getPatient: (...args: unknown[]) => getPatient(...args),
    updatePatient: (...args: unknown[]) => updatePatient(...args),
    softDeleteGuardian: (...args: unknown[]) => softDeleteGuardian(...args),
  }
})

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 'u1', email: 'pro@x.com', full_name: 'Prof', role: 'PROFESSIONAL' },
    loading: false,
    isImpersonated: false,
  }),
}))

vi.mock('../contexts/BrandingContext', () => ({
  useBranding: () => ({ branding: null }),
}))

describe('Patients (validações do formulário)', () => {
  beforeEach(() => {
    listPatients.mockResolvedValue({ patients: [] })
    createPatient.mockResolvedValue({ id: 'p1' })
  })

  async function openNewPatient() {
    const user = userEvent.setup()
    render(
      <MemoryRouter>
        <Patients />
      </MemoryRouter>,
    )
    // Aguarda o load() inicial terminar, evitando warning de act().
    await screen.findByText(/nenhum paciente cadastrado/i)
    const btn = await screen.findByRole('button', { name: /novo paciente/i })
    await user.click(btn)
    await screen.findByRole('heading', { name: 'Novo paciente' })
    return { user }
  }

  it('exige nome do responsável quando email do responsável é preenchido', async () => {
    const { user } = await openNewPatient()
    await user.type(screen.getByLabelText(/^e-mail do responsável/i), 'teste@exemplo.com')
    fireEvent.submit(document.getElementById('new-patient-form')!)
    expect(await screen.findByText('Nome do responsável é obrigatório.')).toBeInTheDocument()
  })

  it('valida email do responsável via regex', async () => {
    const { user } = await openNewPatient()
    await user.type(screen.getByLabelText(/^nome do responsável \(guardião legal\)/i), 'Maria')
    await user.type(screen.getByLabelText(/^e-mail do responsável/i), 'invalido')
    fireEvent.submit(document.getElementById('new-patient-form')!)
    expect(await screen.findByText('E-mail do responsável inválido.')).toBeInTheDocument()
  })

  it('exige CPF do responsável quando email do responsável é preenchido', async () => {
    const { user } = await openNewPatient()
    await user.type(screen.getByLabelText(/^nome do responsável \(guardião legal\)/i), 'Maria')
    await user.type(screen.getByLabelText(/^e-mail do responsável/i), 'maria@exemplo.com')
    fireEvent.submit(document.getElementById('new-patient-form')!)
    expect(await screen.findByText('CPF do responsável é obrigatório.')).toBeInTheDocument()
  })

  it('valida CPF do responsável (DV) no frontend', async () => {
    const { user } = await openNewPatient()
    await user.type(screen.getByLabelText(/^nome do responsável \(guardião legal\)/i), 'Maria')
    await user.type(screen.getByLabelText(/^e-mail do responsável/i), 'maria@exemplo.com')
    await user.type(screen.getByLabelText(/^cpf do responsável/i), '111.111.111-11')
    fireEvent.submit(document.getElementById('new-patient-form')!)
    expect(await screen.findByText('CPF do responsável inválido.')).toBeInTheDocument()
  })

  it('exige endereço completo e CEP com 8 dígitos quando email do responsável é preenchido', async () => {
    const { user } = await openNewPatient()
    await user.type(screen.getByLabelText(/^nome do responsável \(guardião legal\)/i), 'Maria')
    await user.type(screen.getByLabelText(/^e-mail do responsável/i), 'maria@exemplo.com')
    await user.type(screen.getByLabelText(/^cpf do responsável/i), '529.982.247-25')
    // endereço incompleto
    await user.type(screen.getByLabelText(/^rua$/i), 'Rua X')
    fireEvent.submit(document.getElementById('new-patient-form')!)
    expect(await screen.findByText(/preencha todos os campos do endereço/i)).toBeInTheDocument()

    // completa e coloca CEP inválido
    await user.type(screen.getByLabelText(/^bairro$/i), 'Bairro')
    await user.type(screen.getByLabelText(/^cidade$/i), 'Cidade')
    await user.type(screen.getByLabelText(/^estado$/i), 'SC')
    await user.type(screen.getByLabelText(/^país$/i), 'Brasil')
    await user.type(screen.getByLabelText(/^cep$/i), '123')
    await user.type(screen.getByLabelText(/^data de nascimento do responsável/i), '2000-01-01')
    await user.type(screen.getByLabelText(/^data de nascimento do paciente/i), '2020-01-01')
    fireEvent.submit(document.getElementById('new-patient-form')!)
    expect(await screen.findByText('CEP deve ter 8 dígitos.')).toBeInTheDocument()
  })

  it('valida CPF opcional do paciente (quando preenchido) e envia no payload', async () => {
    const { user } = await openNewPatient()
    await user.type(screen.getByLabelText(/nome do paciente/i), 'Paciente')
    await user.type(screen.getByLabelText(/^cpf do paciente/i), '529.982.247-25')
    fireEvent.submit(document.getElementById('new-patient-form')!)

    await waitFor(() => expect(createPatient).toHaveBeenCalled())
    const arg = createPatient.mock.calls[0][0]
    expect(arg.patient_cpf).toBe('529.982.247-25')
  })
})

