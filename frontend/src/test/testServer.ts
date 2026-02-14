import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'

export const server = setupServer(
  http.get('/api/patients', () => {
    return HttpResponse.json({ patients: [{ id: 'p1', full_name: 'Paciente 1', birth_date: '2020-01-01' }] })
  }),
)

