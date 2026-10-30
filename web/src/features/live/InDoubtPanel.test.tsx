import { fireEvent, screen } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { setDisplayTimeZone } from '@/lib/format'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { useLiveActions, useLivePermissions } from './actions'
import { InDoubtPanel } from './Decision'
import type { Incident, Snapshot } from './model'

/*
 * The decision on messages in doubt (LiveMonitorStates › in doubt; review
 * GS-1, R2.L3): what is in doubt where, the two ways on, and the call that
 * makes the choice.
 */

const T = (hms: string) => `2026-10-14T${hms}Z`

const doubt: Incident = {
  id: 'i9',
  kind: 'in_doubt',
  scope: 'campaign',
  mailbox: '',
  mailboxName: '',
  providerId: '',
  provider: '',
  providerShort: '',
  metric: 'in_doubt',
  held: 15,
  trippedAt: T('09:52:02'),
  resolution: '',
  detail: {
    succeeded: 0,
    remaining: 15,
    options: [
      { key: 'resend', title: 'Send them again', body: 'Anyone whose copy the provider did take gets the email twice.', action: 'Send 15 messages again', recommended: false },
      { key: 'leave_out', title: 'Leave them out (recommended)', body: 'They\'re marked "not sent, in doubt" in the report.', action: 'Leave 15 messages out', recommended: true },
    ],
  },
}

const waiting = {
  campaignId: 'c',
  name: 'Weekly Digest #142',
  status: 'paused',
  state: 'paused',
  at: T('09:52:10'),
  recipients: 31192,
  sent: 31177,
  queued: 0,
  held: 0,
  waiting: 0,
  options: { canPause: false, canResume: false, canCancel: true },
  helpers: { send: ['Linh Nguyen', 'Marcus Chen'], pause: [], approve: [] },
  inDoubt: {
    count: 15,
    since: T('09:47:31'),
    safe: false,
    waiting: true,
    incidentId: 'i9',
    providers: [{ providerId: 'postal', provider: 'Postal', short: 'Postal', count: 15, since: T('09:47:31'), idempotent: false, safe: false }],
  },
  incidents: [doubt],
} as unknown as Snapshot

function Panel({ s }: { s: Snapshot }) {
  const actions = useLiveActions('acme-newsletter', 'c', () => {})
  const can = useLivePermissions(s.helpers)
  return <InDoubtPanel s={s} actions={actions} can={can} />
}

const sender = workspaceMe({ permissions: ['campaigns:view', 'campaigns:send', 'campaigns:pause'] })

describe('the in-doubt decision', () => {
  let unmock = () => {}
  beforeAll(() => setDisplayTimeZone('UTC'))
  afterEach(() => unmock())

  it('says what is in doubt where, and recommends leaving them out when a copy could arrive twice', async () => {
    unmock = mockApi({})
    await renderInWorkspace(<Panel s={waiting} />, { me: sender })
    expect(screen.getByRole('alert')).toHaveTextContent('15 messages in doubt wait for your decision')
    expect(screen.getByText('Postal · 15 messages · handed over at 09:47:31')).toBeInTheDocument()
    expect(screen.getByText("Postal can't recognize a message it already took, so sending them again can deliver a copy twice.")).toBeInTheDocument()
    expect(screen.getByText('Could arrive twice')).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /Leave them out \(recommended\)/ })).toBeChecked()
    expect(screen.getByRole('radio', { name: /Send them again/ })).not.toBeChecked()
    expect(screen.getByText('Linh Nguyen and Marcus Chen can decide')).toBeInTheDocument()
  })

  it('leaves them out, or sends them again, through the in-doubt call', async () => {
    const bodies: unknown[] = []
    unmock = mockApi({
      'POST /api/v1/workspaces/acme-newsletter/campaigns/c/live/in-doubt': async (r) => {
        bodies.push(await r.json())
        return json({ moved: 0, cancelled: 0, excluded: 0, decided: 15, snapshot: waiting } satisfies Schemas['LiveActionResult'])
      },
    })
    await renderInWorkspace(<Panel s={waiting} />, { me: sender })
    fireEvent.click(screen.getByRole('button', { name: 'Leave 15 messages out' }))
    expect(await screen.findByText('15 messages in doubt left out')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('radio', { name: /Send them again/ }))
    fireEvent.click(await screen.findByRole('button', { name: 'Send 15 messages again' }))
    expect(await screen.findByText('15 messages in doubt go out again')).toBeInTheDocument()
    expect(bodies).toEqual([{ action: 'leave_out' }, { action: 'resend' }])
  })

  it('names who can decide to someone who cannot', async () => {
    unmock = mockApi({})
    await renderInWorkspace(<Panel s={waiting} />)
    expect(screen.getByRole('button', { name: 'Leave 15 messages out' })).toHaveAttribute('aria-disabled', 'true')
  })

  it('shows nothing once nobody needs to decide', async () => {
    unmock = mockApi({})
    await renderInWorkspace(<Panel s={{ ...waiting, incidents: [{ ...doubt, resolvedAt: T('09:53:00') }] }} />, { me: sender })
    expect(screen.queryByRole('radiogroup')).not.toBeInTheDocument()
    expect(screen.queryByText('15 messages in doubt wait for your decision')).not.toBeInTheDocument()
  })
})
