import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { setDisplayTimeZone } from '@/lib/format'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ScheduledCard, SuppressionActions, zonedInstant } from './schedule'

type Detail = Schemas['TransactionalMessageDetail']

const sender = workspaceMe({ permissions: ['transactional:view', 'transactional:send', 'subscribers:edit'] })

function detail(patch: Partial<Detail>): Detail {
  return { id: '0192f0a0-0000-7000-8000-0000000000aa', email: 'maya@northwind.io', status: 'scheduled', sendAt: '2030-01-08T14:00:00Z', ...patch } as Detail
}

describe('Transactional › a scheduled message (review 2026-11 TX-1, TX-2)', () => {
  beforeEach(() => setDisplayTimeZone('America/New_York'))
  afterEach(() => mockApi({})())

  it('reads a wall time in a zone as its moment', () => {
    expect(zonedInstant('2030-01-08', '09:00', 'America/New_York').toISOString()).toBe('2030-01-08T14:00:00.000Z')
    expect(zonedInstant('2030-07-08', '09:00', 'America/New_York').toISOString()).toBe('2030-07-08T13:00:00.000Z')
    expect(zonedInstant('2030-01-08', '09:00', 'Asia/Ho_Chi_Minh').toISOString()).toBe('2030-01-08T02:00:00.000Z')
  })

  it('cancels and moves a scheduled message', async () => {
    const calls: { path: string; body?: unknown }[] = []
    const d = detail({})
    mockApi({
      [`POST /api/v1/workspaces/acme-newsletter/transactional/messages/${d.id}/cancel`]: () => {
        calls.push({ path: 'cancel' })
        return json({ id: d.id, status: 'cancelled' })
      },
      [`POST /api/v1/workspaces/acme-newsletter/transactional/messages/${d.id}/reschedule`]: async (r) => {
        calls.push({ path: 'reschedule', body: await r.json() })
        return json({ id: d.id, status: 'scheduled' })
      },
    })
    await renderInWorkspace(<ScheduledCard d={d} />, { me: sender })
    expect(screen.getByText(/Goes out .* \(America\/New_York\)/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Reschedule…' }))
    const move = await screen.findByRole('dialog', { name: 'Move the send' })
    expect(within(move).getByLabelText('Time')).toHaveValue('09:00')
    fireEvent.change(within(move).getByLabelText('Time'), { target: { value: '10:30' } })
    fireEvent.click(within(move).getByRole('button', { name: /^Move to/ }))
    await waitFor(() => expect(calls).toEqual([{ path: 'reschedule', body: { sendAt: '2030-01-08T15:30:00.000Z' } }]))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel send…' }))
    const cancel = await screen.findByRole('dialog', { name: 'Cancel this message?' })
    fireEvent.click(within(cancel).getByRole('button', { name: 'Cancel send' }))
    await waitFor(() => expect(calls.map((c) => c.path)).toEqual(['reschedule', 'cancel']))
  })

  it('shows nothing to change without transactional:send', async () => {
    await renderInWorkspace(<ScheduledCard d={detail({})} />, { me: workspaceMe({ permissions: ['transactional:view'] }) })
    expect(screen.getByText(/Goes out/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Cancel send…' })).not.toBeInTheDocument()
  })

  it('removes a suppressed recipient with a reason', async () => {
    const bodies: unknown[] = []
    mockApi({
      'POST /api/v1/workspaces/acme-newsletter/suppressions/sup-1/remove': async (r) => {
        bodies.push(await r.json())
        return new Response(null, { status: 204 })
      },
    })
    const d = detail({ status: 'rejected', rejection: { reason: 'suppressed', suppression: { id: 'sup-1', reason: 'hard_bounce', since: '2026-10-13T09:00:00Z', source: 'Gmail' } } })
    await renderInWorkspace(<SuppressionActions d={d} />, { me: sender })
    fireEvent.click(screen.getByRole('button', { name: 'Remove from suppression list…' }))
    const dialog = await screen.findByRole('dialog', { name: 'Remove maya@northwind.io from the suppression list?' })
    expect(within(dialog).getByText(/marketing waits for them to confirm/)).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Remove from list' }))
    expect(await within(dialog).findAllByText('Say why, in a few words.')).not.toHaveLength(0)
    fireEvent.change(within(dialog).getByLabelText('Reason (kept in the audit log)'), { target: { value: 'Fixed their mailbox' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Remove from list' }))
    await waitFor(() => expect(bodies).toEqual([{ reason: 'Fixed their mailbox' }]))
  })
})
