import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import type { RoutingConfig } from './routing-model'
import { RoutingPage } from './RoutingPage'

const ses = '0199e000-0000-7000-8000-000000000001'
const postal = '0199e000-0000-7000-8000-000000000002'
const postmark = '0199e000-0000-7000-8000-000000000003'

const provider = (patch: Partial<Schemas['RoutingProvider']> & Pick<Schemas['RoutingProvider'], 'id' | 'name' | 'short' | 'adapter'>): Schemas['RoutingProvider'] => ({
  broadcastStreamCreatable: false,
  bulk: 'allowed',
  bulkAllowed: true,
  color: 'other',
  label: '',
  removesWebhook: false,
  shareSteps: [],
  status: 'healthy',
  terms: [],
  warmingUp: false,
  ...patch,
})

/** The demo routing: Broadcasts SES 90 / Postal 10, with fallback as given. */
const config = (broadcastFallback = ''): RoutingConfig => ({
  streams: [
    { stream: 'broadcast', version: 'b1', splits: [{ providerId: ses, weight: 90 }, { providerId: postal, weight: 10 }], fallback: broadcastFallback, failover: 'pause_and_ask', conditions: [] },
    { stream: 'automation', version: 'a1', splits: [{ providerId: ses, weight: 100 }], fallback: postmark, failover: 'automatic', conditions: [] },
    { stream: 'system', version: 's1', splits: [{ providerId: postmark, weight: 100 }], fallback: ses, failover: 'automatic', conditions: [] },
    { stream: 'transactional', version: '0', splits: [], fallback: '', failover: 'automatic', conditions: [] },
  ],
  providers: [
    provider({ id: ses, name: 'Amazon SES', short: 'SES', adapter: 'ses', color: 'ses', label: 'us-east-1' }),
    provider({ id: postal, name: 'Postal', short: 'Postal', adapter: 'postal', color: 'postal', label: 'self-hosted · mta.acme.io' }),
    provider({ id: postmark, name: 'Postmark', short: 'Postmark', adapter: 'postmark', color: 'postmark', label: 'Transactional stream', bulk: 'broadcast_streams_only', bulkAllowed: false }),
  ],
  lists: [],
  senderDomains: ['news.acme.io'],
  categories: [],
  templates: [],
  savedBy: 'Linh Nguyen',
  issues: [],
  shared: { suppressions: 1874, pushed: 1498 },
})

const owner = workspaceMe({
  permissions: ['providers:view', 'providers:manage', 'campaigns:view'],
  workspace: { ...workspaceMe().workspace, roleKey: 'owner', roleName: 'Owner' },
})
const path = '/w/acme-newsletter/providers/routing'
const ws = 'GET /api/v1/workspaces/acme-newsletter'
const problem = (status: number, title: string, extra: Record<string, unknown> = {}) => json({ title, status, detail: title, ...extra }, status)

/** The routing API: GET answers with the latest config, or fails while `down` says so. */
function api(cfg: RoutingConfig, opts: { put?: () => Response; campaign?: Schemas['RouteCampaignResult'] } = {}) {
  const state = { cfg, down: false, puts: 0 }
  mockApi({
    [`${ws}/routing`]: () => (state.down ? problem(503, 'Service Unavailable') : json(state.cfg)),
    [`${ws}/channels`]: () => json({ switchShown: false, channels: [{ channel: 'email', available: true, setUp: true }] }),
    [`${ws}/providers`]: () => json({ providers: [] }),
    'POST /api/v1/workspaces/acme-newsletter/routing/test': () => problem(500, 'Internal Server Error'),
    'POST /api/v1/workspaces/acme-newsletter/routing/test-campaign': () => (opts.campaign ? json(opts.campaign) : problem(500, 'Internal Server Error')),
    'PUT /api/v1/workspaces/acme-newsletter/routing': () => {
      state.puts++
      return opts.put ? opts.put() : json(state.cfg)
    },
  })
  return state
}

const automatic = () => screen.getByRole('radio', { name: /^Fail over automatically to any fallback that passes its checks/ })

// Review R3.U4b: the routing editor keeps unsaved edits safe in both views.
describe('Routing › unsaved edits', () => {
  afterEach(() => mockApi({}))

  it('keeps the editor and its edits when a background refresh fails, and says so', async () => {
    const state = api(config())
    const { queryClient } = await renderInWorkspace(<RoutingPage />, { me: owner, path })
    await waitFor(() => expect(automatic()).not.toBeChecked())
    fireEvent.click(automatic())
    expect(automatic()).toBeChecked()
    expect(screen.getByRole('region', { name: 'Unsaved changes' })).toBeInTheDocument()

    state.down = true
    await act(() => queryClient.refetchQueries())
    expect(await screen.findByText("Couldn't refresh the routing")).toBeInTheDocument()
    expect(screen.getByText(/Your edits are still here, and mail keeps going out on the saved routing\./)).toBeInTheDocument()
    // The editor stayed mounted with the edit, instead of the load error.
    expect(automatic()).toBeChecked()
    expect(screen.queryByText("Couldn't load the routing")).not.toBeInTheDocument()

    state.down = false
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() => expect(screen.queryByText("Couldn't refresh the routing")).not.toBeInTheDocument())
    expect(automatic()).toBeChecked()
  })

  it('in the bulk-terms warning, asks before leaving and says when a save was refused', async () => {
    // Broadcasts fall back to Postmark's Transactional stream, which its
    // terms keep from bulk mail: any edit opens the warning view.
    api(config(postmark), {
      put: () => problem(409, 'Conflict', { code: 'routing_changed', meta: { streams: ['broadcast'] } }),
    })
    const { router } = await renderInWorkspace(<RoutingPage />, { me: owner, path })
    await waitFor(() => expect(automatic()).not.toBeChecked())
    fireEvent.click(automatic())
    expect(await screen.findByRole('radio', { name: /^Keep Postmark/ })).toBeInTheDocument()

    // Leaving asks first; the warning stands, so there's no "save and leave".
    act(() => {
      void router.navigate({ to: '/w/$workspace/campaigns', params: { workspace: 'acme-newsletter' } })
    })
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText('Leave without saving the routing?')).toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: 'Save routing and leave' })).not.toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Stay' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(router.state.location.pathname).toBe(path)

    // Keeping the provider anyway: the save is refused because Broadcasts
    // changed meanwhile, and the note says so above the warning.
    fireEvent.click(screen.getByRole('radio', { name: /^Keep Postmark/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Keep it and save' }))
    expect(await screen.findByText('Broadcasts changed while you were editing')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Show the latest routing' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Keep my edits and replace it' })).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /^Keep Postmark/ })).toBeChecked()
  })
})

const share = (patch: Partial<Schemas['RouteCampaignShare']> & Pick<Schemas['RouteCampaignShare'], 'providerId' | 'provider' | 'count'>): Schemas['RouteCampaignShare'] => ({
  short: patch.provider,
  color: 'other',
  policy: 'automatic',
  ...patch,
})

const mailboxes = (gmail: Partial<Schemas['RouteCampaignMailboxCap']> = {}): Schemas['RouteCampaignMailboxCap'][] => [
  { mailbox: 'gmail', cap: 3000, left: 1040, recipients: 1812, over: 772, ...gmail },
  ...['outlook', 'yahoo', 'icloud', 'other'].map((mailbox) => ({ mailbox, cap: 1125, left: 400, recipients: 300, over: 0 })),
]

/** Weekly Digest #142 through SES 90 / Postal 10, Postal on warm-up day 10. */
const campaign = (warmup: Partial<Schemas['RouteCampaignWarmup']>, postalShare: Partial<Schemas['RouteCampaignShare']> = {}): Schemas['RouteCampaignResult'] => ({
  none: false,
  campaignId: 'c142',
  name: 'Weekly Digest #142',
  timezone: 'UTC',
  lists: [],
  domain: 'news.acme.io',
  total: 48213,
  sunset: 412,
  shares: [
    share({ providerId: ses, provider: 'Amazon SES', count: 43392, color: 'ses' }),
    share({ providerId: postal, provider: 'Postal', count: 4821, color: 'postal', fallback: 'Amazon SES', ...postalShare }),
  ],
  conditions: [],
  warmup: [
    {
      providerId: postal,
      provider: 'Postal',
      day: 10,
      date: '2026-10-13T13:00:00Z',
      cap: 7500,
      left: 2320,
      spread: true,
      mailboxCaps: mailboxes(),
      paused: false,
      tier: 1180,
      tierTo: 'Amazon SES',
      over: 1321,
      resumes: '2026-10-14T04:00:00Z',
      ...warmup,
    },
  ],
  impact: [],
  estimated: false,
  simulated: 48213,
})

const campaignTab = async () => {
  fireEvent.click(await screen.findByRole('button', { name: 'Campaign' }))
  return screen.findByText('Weekly Digest #142')
}

// Review R3.U4c: the Campaign tab shows what the send would do on its day:
// the warm-up's tiers, cap and hold, and where a down provider's mail goes.
describe('Routing › route tester › Campaign', () => {
  afterEach(() => mockApi({}))

  it('shows the warm-up hold without overflow, per mailbox provider, and a down provider’s failover', async () => {
    api(config(), { campaign: campaign({}, { down: { status: 'down', moved: 4821, to: 'Amazon SES', held: 0, checks: [] } }) })
    await renderInWorkspace(<RoutingPage />, { me: owner, path })
    await campaignTab()
    const row = (label: string) => screen.getByText(label).closest('div')!
    expect(row("Outside day 10's engagement tiers, to Amazon SES")).toHaveTextContent('1,180')
    expect(row("Held by Postal's warm-up cap")).toHaveTextContent('1,321')
    expect(screen.queryByText(/^Warm-up overflow to/)).not.toBeInTheDocument()
    expect(
      screen.getByText(/^Postal's cap on .+ \(day 10\) is 7,500, split by mailbox provider\. Over its part at Gmail \(1,812 for 1,040\)\. They go out from .+, when the next day's cap starts\.$/),
    ).toBeInTheDocument()
    expect(screen.getByText("Postal can't send now: 4,821 recipients go through Amazon SES instead.")).toBeInTheDocument()
  })

  it('says a paused plan sends its share through the warm provider when it overflows, and holds it otherwise', async () => {
    api(config(), { campaign: campaign({ paused: true, tier: 0, tierTo: undefined, over: 4821, to: 'Amazon SES', resumes: undefined, mailboxCaps: mailboxes({ over: 0 }) }) })
    const first = await renderInWorkspace(<RoutingPage />, { me: owner, path })
    await campaignTab()
    expect(screen.getByText('Warm-up overflow to Amazon SES').closest('div')).toHaveTextContent('4,821')
    expect(screen.getByText("Postal's warm-up is paused, so its whole share goes through Amazon SES.")).toBeInTheDocument()
    first.unmount()

    api(config(), { campaign: campaign({ paused: true, tier: 0, tierTo: undefined, over: 4821, resumes: undefined, mailboxCaps: mailboxes({ over: 0 }) }) })
    await renderInWorkspace(<RoutingPage />, { me: owner, path })
    await campaignTab()
    expect(screen.getByText("Held while Postal's warm-up is paused").closest('div')).toHaveTextContent('4,821')
    expect(screen.getByText("Postal's warm-up is paused: its recipients wait until someone resumes it.")).toBeInTheDocument()
  })
})
