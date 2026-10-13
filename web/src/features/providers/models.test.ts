import type { Schemas } from '@/api/client'
import { setDisplayTimeZone } from '@/lib/format'
import { bouncePct, complaintPct, limitLabel, limitPct, perSec, rateColor, statusBadge } from './common'
import { bulkIssues, costText, fallbackText, providerOption, sameDraft, setPrimary, setShare, stepText, toDraft, toInput, type RoutingConfig } from './routing-model'
import { audienceText, rampHeight, tiersText, visibleDays, type WarmupDay } from './warmup-model'

const ses = 'ses-0000-0000-0000-000000000001'
const postmark = 'pm-00-0000-0000-0000-000000000002'
const postal = 'po-00-0000-0000-0000-000000000003'

function provider(id: string, name: string, extra: Partial<Schemas['RoutingProvider']> = {}): Schemas['RoutingProvider'] {
  return {
    id,
    name,
    short: name === 'Amazon SES' ? 'SES' : name,
    label: '',
    adapter: name.toLowerCase(),
    color: 'other',
    status: 'healthy',
    bulk: 'allowed',
    bulkAllowed: true,
    broadcastStreamCreatable: false,
    warmingUp: false,
    shareSteps: [],
    terms: [],
    ...extra,
  }
}

const cfg: RoutingConfig = {
  streams: [
    { stream: 'broadcast', splits: [{ providerId: ses, weight: 90 }, { providerId: postal, weight: 10 }], fallback: '', failover: 'pause_and_ask', conditions: [] },
    { stream: 'automation', splits: [{ providerId: ses, weight: 100 }], fallback: postmark, failover: 'automatic', conditions: [] },
    { stream: 'system', splits: [{ providerId: postmark, weight: 100 }], fallback: ses, failover: 'automatic', conditions: [] },
  ],
  providers: [
    provider(ses, 'Amazon SES', { label: 'us-east-1' }),
    provider(postmark, 'Postmark', { label: 'Transactional stream', bulk: 'broadcast_streams_only', bulkAllowed: false, broadcastStreamCreatable: true }),
    provider(postal, 'Postal', { label: 'self-hosted · mta.acme.io', warmingUp: true }),
  ],
  lists: [],
  senderDomains: [],
  savedBy: '',
  issues: [],
  shared: { suppressions: 0 },
}

describe('routing model', () => {
  it('round-trips the saved routing', () => {
    const d = toDraft(cfg)
    expect(sameDraft(d, toDraft(cfg))).toBe(true)
    expect(toInput(d).streams[0]).toEqual({ stream: 'broadcast', splits: cfg.streams[0]!.splits, fallback: undefined, failover: 'pause_and_ask', conditions: [] })
  })

  it('keeps the split when the primary changes', () => {
    // A provider already in the split swaps places with the old primary.
    expect(setPrimary(toDraft(cfg).broadcast, postal).splits).toEqual([
      { providerId: postal, weight: 90 },
      { providerId: ses, weight: 10 },
    ])
    // Another provider takes the primary's place and weight.
    expect(setPrimary(toDraft(cfg).broadcast, postmark).splits).toEqual([
      { providerId: postmark, weight: 90 },
      { providerId: postal, weight: 10 },
    ])
    // The new primary can't stay the fallback.
    expect(setPrimary(toDraft(cfg).automation, postmark)).toMatchObject({ splits: [{ providerId: postmark, weight: 100 }], fallback: '' })
  })

  it('gives the primary what the other shares leave', () => {
    const b = setShare(toDraft(cfg).broadcast, postal, 15)
    expect(b.splits).toEqual([
      { providerId: ses, weight: 85 },
      { providerId: postal, weight: 15 },
    ])
    expect(setShare(toDraft(cfg).broadcast, postal, 140).splits[0]!.weight).toBe(0)
  })

  it('flags a Broadcasts fallback whose terms forbid bulk mail', () => {
    const d = toDraft(cfg)
    expect(bulkIssues(d, cfg.providers)).toEqual([])
    d.broadcast = { ...d.broadcast, fallback: postmark }
    expect(bulkIssues(d, cfg.providers)).toEqual([{ stream: 'broadcast', providerId: postmark, where: 'fallback' }])
    // Automations may use Postmark's transactional stream.
    expect(bulkIssues({ ...d, broadcast: toDraft(cfg).broadcast }, cfg.providers)).toEqual([])
  })

  it('names providers like the selects on the board', () => {
    expect(providerOption(cfg.providers[0]!)).toBe('Amazon SES · us-east-1')
    expect(providerOption(cfg.providers[1]!)).toBe('Postmark · Transactional')
    expect(providerOption(cfg.providers[1]!, { stream: true })).toBe('Postmark · Transactional stream')
    expect(providerOption(cfg.providers[2]!)).toBe('Postal · mta.acme.io')
    expect(providerOption(cfg.providers[2]!, { warming: true })).toBe('Postal · warming up')
  })

  it('explains each step of the route tester', () => {
    expect(stepText({ kind: 'stream', stream: 'broadcast', conditions: 2, bucket: 0, suppressed: false }, 'broadcast', true)).toBe(
      'Stream is Broadcasts, so the Broadcasts rule set applies.',
    )
    expect(stepText({ kind: 'stream', stream: 'automation', conditions: 0, provider: 'Amazon SES', bucket: 0, suppressed: false }, 'automation', false)).toBe(
      'Stream is Automations: primary Amazon SES, no conditions.',
    )
    expect(
      stepText({ kind: 'condition', field: 'mailbox_provider', value: 'Outlook', provider: 'Amazon SES', warming: 'Postal', conditions: 0, bucket: 0, suppressed: false }, 'broadcast', true),
    ).toBe('Condition matched: recipient mailbox provider is Outlook, so Amazon SES only. Postal is still warming up.')
    expect(
      stepText(
        { kind: 'split', bucket: 37, splits: [{ short: 'SES', weight: 90 }, { short: 'Postal', weight: 10 }], conditions: 0, suppressed: false },
        'broadcast',
        true,
      ),
    ).toBe('No condition matched. Weighted split SES 90 / Postal 10: this address hashes to bucket 37 of 100.')
    expect(stepText({ kind: 'suppression', email: 'a@b.co', suppressed: false, conditions: 0, bucket: 0 }, 'broadcast', true)).toBe(
      'Suppression list checked: a@b.co is not suppressed.',
    )
  })

  it('says what happens if the provider fails', () => {
    const r = { fallback: { policy: 'pause_and_ask' } } as Schemas['RouteTestResult']
    expect(fallbackText(r, 'broadcast')).toBe('pause and ask. Broadcast failover to a cold provider is never automatic.')
    const a = { fallback: { policy: 'automatic', provider: 'Postmark' } } as Schemas['RouteTestResult']
    expect(fallbackText(a, 'automation')).toBe('automatic failover to Postmark after 3 failed sends or 2 failed health checks.')
    expect(fallbackText({ fallback: { policy: 'automatic', provider: 'Amazon SES' } } as Schemas['RouteTestResult'], 'system')).toBe('automatic failover to Amazon SES.')
  })

  it('prices one message', () => {
    expect(costText({ basis: 'metered', per1000: 0.1 })).toBe('$0.0001 · $0.10 per 1,000')
    expect(costText({ basis: 'plan', perMessage: 15 / 3412 })).toBe('plan · about $0.0044')
    expect(costText({ basis: 'self_hosted', perMessage: 0 })).toBe('$0 metered · self-hosted')
  })
})

describe('provider card helpers', () => {
  it('formats limits and rates like the board', () => {
    expect(limitPct(5)).toBe('5%')
    expect(limitPct(0.1)).toBe('0.1%')
    expect(bouncePct(0.6)).toBe('0.6%')
    expect(complaintPct(0.04)).toBe('0.04%')
    expect(perSec(14)).toBe('14')
    expect(perSec(0)).toBe('0')
    expect(perSec(2, true)).toBe('2.0')
  })

  it('labels the bounce and complaint limits', () => {
    expect(limitLabel('Bounce', { limit: 5, review: 5, pause: 10, label: 'SES', stale: false })).toBe('Bounce rate · limit 5% (SES review at 5%, pause at 10%)')
    expect(limitLabel('Bounce', { limit: 10, label: 'Postmark', stale: false })).toBe('Bounce rate · limit 10% (Postmark)')
    expect(limitLabel('Bounce', { limit: 2, label: 'warm-up guard', stale: false })).toBe('Bounce rate · warm-up guard 2%')
    expect(limitLabel('Bounce', { limit: 5, review: 5, pause: 10, label: 'SES', stale: false }, true)).toBe('Bounce rate · limit 5%')
  })

  it('colours a rate by its share of the limit', () => {
    expect(rateColor(0.9, 2)).toBe('var(--ok-bar)')
    expect(rateColor(0.05, 0.1)).toBe('var(--warn-bar)')
    expect(rateColor(3, 2)).toBe('var(--bad-bar)')
    expect(rateColor(0.9, 2, true)).toBe('var(--color-muted-line)')
  })

  it('words the status badge', () => {
    const now = new Date('2026-10-13T09:16:00Z')
    expect(statusBadge({ status: 'healthy', webhook: { kind: 'SNS' } }).text).toBe('Healthy')
    expect(statusBadge({ status: 'warming_up', webhook: { kind: '' }, warmup: { day: 9, days: 21 } as Schemas['ProviderWarmupBrief'] }).text).toBe('Warming up · day 9 of 21')
    expect(statusBadge({ status: 'down', webhook: { kind: '' }, down: { since: '2026-10-13T09:02:00Z', reason: 'outage', errors: [] } }, now).text).toBe('Down · 14 min')
    expect(statusBadge({ status: 'down', webhook: { kind: '' }, down: { since: '2026-10-13T09:02:00Z', reason: 'auth', errors: [] } }, now).text).toBe('Down · auth failed')
    expect(statusBadge({ status: 'degraded', webhook: { kind: '', silentSince: '2026-10-13T03:14:00Z' } }, now).text).toBe('Degraded · webhooks silent 6 h 2 min')
  })
})

describe('warm-up model', () => {
  beforeAll(() => setDisplayTimeZone('UTC'))
  const day = (n: number, status: WarmupDay['status'], extra: Partial<WarmupDay> = {}): WarmupDay => ({
    day: n,
    date: `2026-10-${String(n + 3).padStart(2, '0')}T00:00:00Z`,
    cap: 100 * n,
    plannedCap: 100 * n,
    held: false,
    tiers: ['active'],
    share: 10,
    shareChange: false,
    uses: [],
    status,
    eligible: false,
    ...extra,
  })

  it('compresses the ramp but keeps it monotonic', () => {
    expect(rampHeight(48000, 48000)).toBe(100)
    expect(rampHeight(6000, 48000)).toBe(22)
    expect(rampHeight(200, 48000)).toBe(2)
    expect(rampHeight(0, 48000)).toBe(0)
    expect(rampHeight(7500, 48000)).toBeGreaterThan(rampHeight(6000, 48000))
  })

  it('shows the days around today and the last day', () => {
    const days = Array.from({ length: 21 }, (_, i) => day(i + 1, i + 1 < 9 ? 'done' : i + 1 === 9 ? 'today' : 'planned'))
    expect(visibleDays(days).map((d) => d.day)).toEqual([5, 6, 7, 8, 9, 10, 11, 12, 21])
  })

  it('describes the audience of a day', () => {
    expect(tiersText(['active'])).toBe('Active tier')
    expect(tiersText(['active', 'cooling'])).toBe('Active + Cooling')
    expect(tiersText(['active', 'cooling', 'at_risk'])).toBe('All tiers except Sunset')
    expect(tiersText(['active', 'cooling', 'at_risk', 'sunset'])).toBe('All')
    expect(audienceText(day(11, 'planned', { tiers: ['active', 'cooling'], uses: [{ name: 'Weekly Digest #142', count: 3160 }] }))).toBe(
      'Active + Cooling · Weekly Digest #142 uses 3,160',
    )
    expect(audienceText(day(12, 'planned', { tiers: ['active', 'cooling', 'at_risk'], share: 15, shareChange: true }))).toBe('All tiers except Sunset · share rises to 15%')
    expect(audienceText(day(21, 'planned', { tiers: ['active', 'cooling', 'at_risk', 'sunset'], share: 40, eligible: true }))).toBe('All · share 40%, eligible as failover')
  })
})
