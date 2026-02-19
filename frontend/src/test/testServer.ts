import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'

export const server = setupServer(
  http.get(/\/api\/patients(\?|$)/, () => {
    return HttpResponse.json({
      patients: [{ id: 'p1', full_name: 'Paciente 1', birth_date: '2020-01-01' }],
      limit: 20,
      offset: 0,
      total: 1,
    })
  }),
  http.get(/\/api\/me\/available-slots/, ({ request }) => {
    const url = new URL(request.url)
    const from = url.searchParams.get('from')
    const to = url.searchParams.get('to')
    if (!from || !to) {
      return HttpResponse.json({ error: 'from and to required' }, { status: 400 })
    }
    return HttpResponse.json({
      slots: [
        { date: '2025-02-04', start_time: '09:00' },
        { date: '2025-02-04', start_time: '10:00' },
        { date: '2025-02-11', start_time: '09:00' },
      ],
    })
  }),
)

