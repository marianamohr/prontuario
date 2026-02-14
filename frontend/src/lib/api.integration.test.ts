import { describe, expect, it } from 'vitest'
import * as api from './api'

describe('api (integration via msw)', () => {
  it('listPatients retorna lista do backend', async () => {
    const res = await api.listPatients()
    expect(res.patients).toHaveLength(1)
    expect(res.patients[0].id).toBe('p1')
  })
})

