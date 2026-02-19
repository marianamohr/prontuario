import { describe, expect, it } from 'vitest'
import * as api from './api'

describe('api (integration via msw)', () => {
  it('listPatients retorna lista do backend', async () => {
    const res = await api.listPatients()
    expect(res.patients).toHaveLength(1)
    expect(res.patients[0].id).toBe('p1')
  })

  it('listAvailableSlots retorna slots do backend', async () => {
    const res = await api.listAvailableSlots('2025-02-01', '2025-02-28')
    expect(res.slots).toBeDefined()
    expect(Array.isArray(res.slots)).toBe(true)
    expect(res.slots.length).toBeGreaterThanOrEqual(0)
    if (res.slots.length > 0) {
      expect(res.slots[0]).toHaveProperty('date')
      expect(res.slots[0]).toHaveProperty('start_time')
    }
  })
})

