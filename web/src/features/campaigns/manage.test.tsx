import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { components } from '@/api/schema'
import { setDisplayTimeZone } from '@/lib/format'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { CalendarPage, viewDays } from './CalendarPage'
import { CampaignsPage, whenRange } from './CampaignsPage'
import { FollowUpDialog } from './followup'

type Item = components['schemas']['CampaignListItem']

const base = {
  channel: 'email' as const,
  audience: 'All subscribers',
  blocked: false,
  notes: [],
  paidOnly: false,
  routes: [],
  tags: [] as string[],
  subject: 'Subject',
  createdAt: '2026-10-01T09:00:00Z',
  updatedAt: '2026-10-02T09:00:00Z',
  updatedBy: 'Aisha',
}
const draft: Item = { ...base, id: '0192f0a0-0000-7000-8000-00000000000a', name: 'Flash sale', status: 'draft' }
const sent: Item = { ...base, id: '0192f0a0-0000-7000-8000-00000000000b', name: 'October update', status: 'sent', startedAt: '2026-10-02T09:00:00Z', tags: ['q4'], delivered: 0.99 }
const editor = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit', 'campaigns:send', 'reports:view'] })
const counts = { all: 2, drafts: 1, paidOnly: 0, scheduled: 0, sending: 0, sent: 1, archived: 3, deleted: 1 }

function mock(extra: Record<string, (r: Request) => Response | Promise<Response>> = {}, items: Item[] = [draft, sent]) {
  return mockApi({
    'GET /api/v1/workspaces/acme-newsletter/campaigns': () => json({ items, total: items.length, page: 1, pageSize: 25, counts }),
    'GET /api/v1/workspaces/acme-newsletter/campaigns/summary': () => json({ days: 30, campaigns: 1, sent: 1200, cost: 0, costBy: [] }),
    'GET /api/v1/workspaces/acme-newsletter/senders': () => json({ senders: [], providers: [], helpers: [], canManage: false }),
    'GET /api/v1/workspaces/acme-newsletter/lists': () => json({ items: [] }),
    'GET /api/v1/workspaces/acme-newsletter/campaigns/tags': () => json({ tags: ['q4'] }),
    ...extra,
  })
}

/** Opens a Radix dropdown from its trigger, as a keyboard does. */
function openMenu(trigger: HTMLElement) {
  fireEvent.keyDown(trigger, { key: 'Enter' })
}

describe('Campaigns › delete, archive, tags and bulk (review 2026-11 CA-1, CA-2)', () => {
  afterEach(() => mockApi({})())

  it('shows the Archived and Deleted tabs with their counts, and a row’s tags', async () => {
    mock()
    await renderInWorkspace(<CampaignsPage search={{}} />, { me: editor })
    expect(await screen.findByRole('button', { name: 'Archived · 3' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Deleted · 1' })).toBeInTheDocument()
    expect(screen.getByText('q4', { selector: '.badge' })).toBeInTheDocument()
  })

  it('deletes a draft from its menu, with Undo', async () => {
    const deleted: string[] = []
    const restored: string[] = []
    mock({
      [`DELETE /api/v1/workspaces/acme-newsletter/campaigns/${draft.id}`]: () => {
        deleted.push(draft.id)
        return new Response(null, { status: 204 })
      },
      [`POST /api/v1/workspaces/acme-newsletter/campaigns/${draft.id}/restore`]: () => {
        restored.push(draft.id)
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<CampaignsPage search={{}} />, { me: editor })
    openMenu(await screen.findByRole('button', { name: 'More actions for Flash sale' }))
    // A draft is deleted, not archived.
    expect(screen.queryByRole('menuitem', { name: 'Archive' })).not.toBeInTheDocument()
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Delete…' }))
    const dialog = await screen.findByRole('dialog', { name: 'Delete "Flash sale"?' })
    expect(within(dialog).getByText(/restored for 30 days/)).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete campaign' }))
    await waitFor(() => expect(deleted).toEqual([draft.id]))
    fireEvent.click(await screen.findByRole('button', { name: 'Undo' }))
    await waitFor(() => expect(restored).toEqual([draft.id]))
  })

  it('archives a sent campaign from its menu', async () => {
    let archived = 0
    mock({
      [`POST /api/v1/workspaces/acme-newsletter/campaigns/${sent.id}/archive`]: () => {
        archived++
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<CampaignsPage search={{}} />, { me: editor })
    openMenu(await screen.findByRole('button', { name: 'More actions for October update' }))
    expect(screen.queryByRole('menuitem', { name: 'Delete…' })).not.toBeInTheDocument()
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Archive' }))
    const dialog = await screen.findByRole('dialog', { name: 'Archive "October update"?' })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Archive' }))
    await waitFor(() => expect(archived).toBe(1))
  })

  it('acts on the selected rows and says what it skipped', async () => {
    const bodies: unknown[] = []
    mock({
      'POST /api/v1/workspaces/acme-newsletter/campaigns/bulk': async (r) => {
        bodies.push(await r.json())
        return json({ done: 1, skipped: [{ id: draft.id, name: 'Flash sale', reason: 'Drafts are deleted, not archived.' }] })
      },
    })
    await renderInWorkspace(<CampaignsPage search={{}} />, { me: editor })
    fireEvent.click(await screen.findByRole('checkbox', { name: 'Select every campaign on this page' }))
    const bar = screen.getByRole('region', { name: 'Act on the selected campaigns' })
    expect(within(bar).getByText('2 selected')).toBeInTheDocument()
    fireEvent.click(within(bar).getByRole('button', { name: 'Archive' }))
    fireEvent.click(within(await screen.findByRole('dialog', { name: 'Archive 2 campaigns?' })).getByRole('button', { name: 'Archive' }))
    await waitFor(() => expect(bodies).toEqual([{ ids: [draft.id, sent.id], action: 'archive' }]))
    const result = await screen.findByRole('dialog', { name: 'Archive: done in part' })
    expect(within(result).getByText(/Flash sale: Drafts are deleted, not archived./)).toBeInTheDocument()
  })

  it('tags one campaign, checking the tag first', async () => {
    const bodies: unknown[] = []
    mock({
      'POST /api/v1/workspaces/acme-newsletter/campaigns/bulk': async (r) => {
        bodies.push(await r.json())
        return json({ done: 1, skipped: [] })
      },
    })
    await renderInWorkspace(<CampaignsPage search={{}} />, { me: editor })
    openMenu(await screen.findByRole('button', { name: 'More actions for Flash sale' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Add tag…' }))
    const dialog = await screen.findByRole('dialog', { name: 'Add a tag to 1 campaign' })
    const input = within(dialog).getByRole('combobox', { name: 'Tag' })
    fireEvent.change(input, { target: { value: '!!' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Add tag' }))
    expect(await within(dialog).findByText(/Use lower-case letters/)).toBeInTheDocument()
    fireEvent.change(input, { target: { value: '  Black   Friday ' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Add tag' }))
    await waitFor(() => expect(bodies).toEqual([{ ids: [draft.id], action: 'tag', tag: 'black friday' }]))
  })

  it('restores from the Deleted tab, without checkboxes', async () => {
    let restored = 0
    const gone: Item = { ...draft, deletedAt: '2026-10-20T09:00:00Z' }
    mock(
      {
        [`POST /api/v1/workspaces/acme-newsletter/campaigns/${draft.id}/restore`]: () => {
          restored++
          return new Response(null, { status: 204 })
        },
      },
      [gone],
    )
    await renderInWorkspace(<CampaignsPage search={{ status: 'deleted' }} />, { me: editor })
    expect(await screen.findByText(/Deleted · restorable until/)).toBeInTheDocument()
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Restore' }))
    await waitFor(() => expect(restored).toBe(1))
  })

  it('turns a "when" filter into the days it covers', () => {
    const now = new Date(2026, 9, 31, 15)
    expect(whenRange({ when: '7d' }, now)).toEqual({ from: '2026-10-25', to: '2026-11-01' })
    expect(whenRange({ when: 'quarter' }, now)).toEqual({ from: '2026-10-01', to: '2027-01-01' })
    expect(whenRange({ when: 'custom', from: '2026-10-01', to: '2026-10-15' }, now)).toEqual({ from: '2026-10-01', to: '2026-10-16' })
    expect(whenRange({}, now)).toEqual({})
  })
})

describe('Campaigns › Calendar (review 2026-11 CA-3)', () => {
  beforeEach(() => setDisplayTimeZone('UTC'))
  afterEach(() => mockApi({})())

  it('lays out whole weeks around the month', () => {
    const days = viewDays('month', '2026-10-14')
    expect(days.length % 7).toBe(0)
    expect(days).toContain('2026-10-01')
    expect(days).toContain('2026-10-31')
    expect(viewDays('week', '2026-10-14')).toHaveLength(7)
  })

  it('moves a scheduled email from the week view', async () => {
    const moved: unknown[] = []
    let asked = ''
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/campaigns/calendar': (r) => {
        asked = new URL(r.url).search
        return json({
          from: '2030-01-07T00:00:00Z',
          to: '2030-01-14T00:00:00Z',
          items: [
            { kind: 'scheduled', id: draft.id, name: 'Flash sale', channel: 'email', status: 'scheduled', at: '2030-01-08T09:00:00Z', recipients: 47842, sent: 0 },
            { kind: 'scheduled', id: 'sms-1', name: 'Summit', channel: 'sms', status: 'scheduled', at: '2030-01-09T10:00:00Z', sent: 0, local: true, localTime: '10:00' },
            { kind: 'digest_issue', id: 'di-1', name: 'Weekly Digest', channel: 'email', status: 'collecting', at: '2030-01-10T09:00:00Z', sent: 0, digestId: 'd-1' },
          ],
        })
      },
      [`POST /api/v1/workspaces/acme-newsletter/campaigns/${draft.id}/reschedule`]: async (r) => {
        moved.push(await r.json())
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<CalendarPage search={{ view: 'week', date: '2030-01-08' }} />, { me: editor })
    expect(await screen.findByText('Flash sale')).toBeInTheDocument()
    expect(asked).toBe('?from=2030-01-07&to=2030-01-14')
    expect(screen.getByText('10:00 local · scheduled')).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: /^Reschedule/ })).toHaveLength(2)
    fireEvent.click(screen.getByRole('button', { name: 'Reschedule Flash sale' }))
    const dialog = await screen.findByRole('dialog', { name: 'Move "Flash sale"?' })
    fireEvent.change(within(dialog).getByLabelText('Day'), { target: { value: '2030-01-11' } })
    fireEvent.click(within(dialog).getByRole('button', { name: /^Move to/ }))
    await waitFor(() => expect(moved).toEqual([{ mode: 'at', date: '2030-01-11', time: '09:00', timezone: 'UTC' }]))
  })

  // Review 2026-11 CA-9: texts, push and WhatsApp move on the same calendar.
  it('moves a text at its local time, and says quiet hours still apply', async () => {
    const moved: unknown[] = []
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/campaigns/calendar': () =>
        json({
          from: '2030-01-07T00:00:00Z',
          to: '2030-01-14T00:00:00Z',
          items: [{ kind: 'scheduled', id: 'sms-1', name: 'Summit', channel: 'sms', status: 'scheduled', at: '2030-01-09T10:00:00Z', sent: 0, local: true, localTime: '10:00' }],
        }),
      'POST /api/v1/workspaces/acme-newsletter/campaigns/sms-1/reschedule': async (r) => {
        moved.push(await r.json())
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(<CalendarPage search={{ view: 'week', date: '2030-01-08' }} />, { me: editor })
    fireEvent.click(await screen.findByRole('button', { name: 'Reschedule Summit' }))
    const dialog = await screen.findByRole('dialog', { name: 'Move "Summit"?' })
    expect(within(dialog).getByText(/inside quiet hours/)).toBeInTheDocument()
    expect(within(dialog).getByRole('link', { name: 'Open Review & send' })).toHaveAttribute('href', '/w/acme-newsletter/campaigns/sms-1/sms-review')
    fireEvent.change(within(dialog).getByLabelText('Day'), { target: { value: '2030-01-12' } })
    fireEvent.click(within(dialog).getByRole('button', { name: /^Move to/ }))
    await waitFor(() => expect(moved).toEqual([{ mode: 'local', date: '2030-01-12', time: '10:00', timezone: 'UTC' }]))
  })

  it('unschedules an approved push campaign on its own channel', async () => {
    let unscheduled = 0
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/campaigns/calendar': () =>
        json({
          from: '2030-01-07T00:00:00Z',
          to: '2030-01-14T00:00:00Z',
          items: [{ kind: 'scheduled', id: 'push-1', name: 'Flags', channel: 'push', status: 'scheduled', at: '2030-01-08T15:00:00Z', sent: 0, approved: true }],
        }),
      'POST /api/v1/workspaces/acme-newsletter/push-campaigns/push-1/unschedule': () => {
        unscheduled++
        return new Response(null, { status: 204 })
      },
    })
    const { router } = await renderInWorkspace(<CalendarPage search={{ view: 'week', date: '2030-01-08' }} />, { me: editor })
    fireEvent.click(await screen.findByRole('button', { name: 'Reschedule Flags' }))
    const dialog = await screen.findByRole('dialog', { name: '"Flags" is approved for this time' })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Unschedule and ask again' }))
    await waitFor(() => expect(unscheduled).toBe(1))
    await waitFor(() => expect(router.state.location.pathname).toBe('/w/acme-newsletter/campaigns/push-1/push-review'))
  })

  // Review 2026-11 CA-10: an approval covers the time, so the calendar says
  // so before anyone picks a new one, and offers to unschedule and ask again.
  it('offers an approved campaign "Unschedule and ask again" instead of a new time', async () => {
    let unscheduled = 0
    mockApi({
      'GET /api/v1/workspaces/acme-newsletter/campaigns/calendar': () =>
        json({
          from: '2030-01-07T00:00:00Z',
          to: '2030-01-14T00:00:00Z',
          items: [{ kind: 'scheduled', id: draft.id, name: 'Flash sale', channel: 'email', status: 'scheduled', at: '2030-01-08T09:00:00Z', sent: 0, approved: true }],
        }),
      [`POST /api/v1/workspaces/acme-newsletter/campaigns/${draft.id}/unschedule`]: () => {
        unscheduled++
        return new Response(null, { status: 204 })
      },
    })
    const { router } = await renderInWorkspace(<CalendarPage search={{ view: 'week', date: '2030-01-08' }} />, { me: editor })
    fireEvent.click(await screen.findByRole('button', { name: 'Reschedule Flash sale' }))
    const dialog = await screen.findByRole('dialog', { name: '"Flash sale" is approved for this time' })
    expect(within(dialog).queryByLabelText('Day')).not.toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Unschedule and ask again' }))
    await waitFor(() => expect(unscheduled).toBe(1))
    await waitFor(() => expect(router.state.location.pathname).toBe(`/w/acme-newsletter/campaigns/${draft.id}/review`))
  })
})

describe('Follow-ups from a report (review 2026-11 CA-4)', () => {
  afterEach(() => mockApi({})())

  it('counts who it reaches and makes the draft', async () => {
    const made: unknown[] = []
    mockApi({
      [`GET /api/v1/workspaces/acme-newsletter/campaigns/${sent.id}/follow-up`]: () =>
        json({ received: 30000, nonOpeners: 26012, machineOnly: 3104, clickers: 900, sentAt: '2020-01-01T10:00:00Z', subject: "What's new in October" }),
      [`POST /api/v1/workspaces/acme-newsletter/campaigns/${sent.id}/follow-up`]: async (r) => {
        made.push(await r.json())
        return json({ id: 'new-1', name: 'October update · follow-up' }, 201)
      },
    })
    const closed = vi.fn()
    await renderInWorkspace(<FollowUpDialog campaign={{ id: sent.id, name: sent.name }} kind="non_openers" onClose={closed} />, { me: editor })
    const dialog = await screen.findByRole('dialog', { name: "Resend to people who didn't open" })
    expect(await within(dialog).findByText('26,012 contacts')).toBeInTheDocument()
    expect(within(dialog).getByText(/3,104 opened only through a privacy proxy/)).toBeInTheDocument()
    // The original went out long ago: Review & send picks the time.
    expect(within(dialog).getByRole('combobox', { name: 'Send' })).toHaveValue('0')
    expect(within(dialog).getByRole('textbox', { name: 'New subject' })).toHaveValue("In case you missed it: What's new in October")
    fireEvent.click(within(dialog).getByRole('radio', { name: 'People who clicked' }))
    expect(await screen.findByRole('dialog', { name: 'Send to people who clicked' })).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('radio', { name: "People who didn't open it" }))
    fireEvent.click(within(dialog).getByRole('button', { name: 'Create the follow-up' }))
    await waitFor(() => expect(made).toEqual([{ kind: 'non_openers', subject: "In case you missed it: What's new in October", delayHours: 0 }]))
    expect(closed).toHaveBeenCalled()
  })
})
