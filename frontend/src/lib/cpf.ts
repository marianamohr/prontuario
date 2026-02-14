export function normalizeCPF(value: string): string {
  return String(value || '').replace(/\D/g, '')
}

// Validação de CPF (11 dígitos + dígitos verificadores).
export function isValidCPF(value: string): boolean {
  const cpf = normalizeCPF(value)
  if (cpf.length !== 11) return false
  if (/^(\d)\1{10}$/.test(cpf)) return false // ex.: 00000000000, 11111111111...

  const digits = cpf.split('').map((c) => Number(c))
  if (digits.some((n) => Number.isNaN(n))) return false

  const calcCheckDigit = (baseLen: number) => {
    let sum = 0
    let weight = baseLen + 1
    for (let i = 0; i < baseLen; i++) {
      sum += digits[i] * weight
      weight--
    }
    const mod = sum % 11
    const d = 11 - mod
    return d >= 10 ? 0 : d
  }

  const d1 = calcCheckDigit(9)
  const d2 = calcCheckDigit(10)
  return digits[9] === d1 && digits[10] === d2
}

