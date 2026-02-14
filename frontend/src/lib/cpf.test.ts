import { describe, expect, it } from 'vitest'
import { isValidCPF, normalizeCPF } from './cpf'

describe('cpf', () => {
  it('normalizeCPF removes non-digits', () => {
    expect(normalizeCPF('123.456.789-09')).toBe('12345678909')
  })

  it('isValidCPF rejects repeated digits', () => {
    expect(isValidCPF('000.000.000-00')).toBe(false)
    expect(isValidCPF('11111111111')).toBe(false)
  })

  it('isValidCPF accepts a known valid CPF', () => {
    // CPF de exemplo amplamente usado em testes
    expect(isValidCPF('529.982.247-25')).toBe(true)
    expect(isValidCPF('52998224725')).toBe(true)
  })

  it('isValidCPF rejects invalid check digits', () => {
    expect(isValidCPF('529.982.247-24')).toBe(false)
  })
})

