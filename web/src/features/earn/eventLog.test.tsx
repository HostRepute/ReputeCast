import { fireEvent, screen, waitFor } from '@testing-library/react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { EventLog } from './PaymentsTab'

describe('Payments › event log with nothing found (review 2026-11 FE-3b)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('says a search found nothing and offers the way back, instead of an empty range', async () => {
    const asked: string[] = []
    unmock = mockApi({
      'GET /api/v1/workspaces/acme-newsletter/earn/payments/events': (r) => {
        asked.push(new URL(r.url).searchParams.get('q') ?? '')
        return json({ items: [], page: 1, pageSize: 8, range: '7d', total: 0, types: [] })
      },
    })
    await renderInWorkspace(<EventLog test={false} />)
    expect(await screen.findByText('No events in this range.')).toBeInTheDocument()
    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'in_9' } })
    expect(await screen.findByText('No events match "in_9" in this range.')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Clear search and filters' }))
    await waitFor(() => expect(screen.getByRole('searchbox')).toHaveValue(''))
    expect(await screen.findByText('No events in this range.')).toBeInTheDocument()
    expect(asked).toContain('in_9')
  })
})
