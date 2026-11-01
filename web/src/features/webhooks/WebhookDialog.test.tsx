import { fireEvent, screen } from '@testing-library/react'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { WebhookDialog } from './WebhookDialog'

const events = [
  { name: 'email.delivered', group: 'Email', description: '' },
  { name: 'email.bounced', group: 'Email', description: '' },
  { name: 'campaign.started', group: 'Campaigns', description: '' },
]

describe('An endpoint that takes email events in batches (review 2026-11 EN-8)', () => {
  let unmock = () => {}
  afterEach(() => unmock())

  it('offers batching once an email event is picked, and sends it with the endpoint', async () => {
    let body: Record<string, unknown> | undefined
    unmock = mockApi({
      'POST /api/v1/workspaces/acme-newsletter/webhooks': async (r) => {
        body = (await r.json()) as Record<string, unknown>
        return json({ webhook: { id: 'w1', name: 'h', description: '', url: 'https://h.example/x', events: ['email.*'], enabled: true, batchEmail: true, status: 'new', createdAt: '2026-11-01T00:00:00Z', total24h: 0, queued: 0 }, secret: 'whsec_x' }, 201)
      },
      'GET /api/v1/workspaces/acme-newsletter/webhooks': () => json({ items: [], events, signatureHeader: '', timestampHeader: '', maxAttempts: 8 }),
      'GET /api/v1/workspaces/acme-newsletter/webhooks/deliveries': () => json({ items: [] }),
    })
    const onSecret = vi.fn()
    await renderInWorkspace(<WebhookDialog events={events} maxAttempts={8} onClose={() => {}} onSecret={onSecret} onShowLog={() => {}} />)
    fireEvent.change(screen.getByRole('textbox', { name: /Endpoint URL/ }), { target: { value: 'https://h.example/x' } })
    fireEvent.click(screen.getByRole('checkbox', { name: /campaign\.started/ }))
    expect(screen.queryByRole('switch', { name: /Send email events in batches/ })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('checkbox', { name: /All email events/ }))
    const sw = screen.getByRole('switch', { name: /Send email events in batches/ })
    expect(sw).not.toBeChecked()
    fireEvent.click(sw)
    fireEvent.click(screen.getByRole('button', { name: 'Add endpoint' }))
    await vi.waitFor(() => expect(onSecret).toHaveBeenCalled())
    expect(body).toMatchObject({ events: ['campaign.started', 'email.*'], batchEmail: true })
  })
})
