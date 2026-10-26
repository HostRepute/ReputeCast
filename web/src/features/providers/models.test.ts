import type { Schemas } from '@/api/client'
import { bouncePct, complaintPct, limitLabel, limitPct, perSec, rateColor, statusBadge } from './common'
import {
  bulkIssues,
  conditionProviders,
  conditionWithout,
  costText,
  fallbackText,
  providerOption,
  providerUses,
  sameDraft,
  setPrimary,
  setShare,
  stepText,
  toDraft,
  toInput,
  withoutProvider,
  type Condition,
  type RoutingConfig,
} from './routing-model'

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
    removesWebhook: false,
    warmingUp: false,
    shareSteps: [],
    terms: [],
    ...extra,
  }
}

const cfg: RoutingConfig = {
  streams: [
    { stream: 'broadcast', version: 'b1', splits: [{ providerId: ses, weight: 90 }, { providerId: postal, weight: 10 }], fallback: '', failover: 'pause_and_ask', conditions: [] },
    { stream: 'automation', version: 'a1', splits: [{ providerId: ses, weight: 100 }], fallback: postmark, failover: 'automatic', conditions: [] },
    { stream: 'system', version: 's1', splits: [{ providerId: postmark, weight: 100 }], fallback: ses, failover: 'automatic', conditions: [] },
  ],
  providers: [
    provider(ses, 'Amazon SES', { label: 'us-east-1' }),
    provider(postmark, 'Postmark', { label: 'Transactional stream', bulk: 'broadcast_streams_only', bulkAllowed: false, broadcastStreamCreatable: true }),
    provider(postal, 'Postal', { label: 'self-hosted · mta.acme.io', warmingUp: true }),
  ],
  lists: [],
  senderDomains: [],
  categories: [],
  templates: [],
  savedBy: '',
  issues: [],
  shared: { suppressions: 0, pushed: 0 },
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

describe('removing a provider', () => {
  it('lists where a provider is used, primaries first', () => {
    expect(providerUses(toDraft(cfg), postmark)).toEqual([
      { kind: 'primary', stream: 'system' },
      { kind: 'failover', stream: 'automation' },
    ])
    expect(providerUses(toDraft(cfg), postal)).toEqual([{ kind: 'share', stream: 'broadcast', weight: 10 }])
  })

  it('gives a primary role to the replacement and drops a fallback that became the primary', () => {
    const d = toDraft(cfg)
    expect(withoutProvider(d.system, postmark, ses, '')).toMatchObject({ splits: [{ providerId: ses, weight: 100 }], fallback: '' })
    expect(withoutProvider(d.automation, postmark, '', postal)).toMatchObject({ splits: [{ providerId: ses, weight: 100 }], fallback: postal })
  })

  it('returns a share to the primary and removes conditions that pointed at the provider', () => {
    const d = toDraft(cfg)
    d.broadcast.conditions = [{ field: 'mailbox_provider', values: ['outlook'], providerId: postal, note: '' }]
    expect(withoutProvider(d.broadcast, postal, '', '')).toMatchObject({ splits: [{ providerId: ses, weight: 100 }], conditions: [] })
    // A replacement already in the split takes the removed primary's weight too.
    expect(withoutProvider(d.broadcast, ses, postal, '').splits).toEqual([{ providerId: postal, weight: 100 }])
  })

  // PR-12: a condition can split by weight.
  it('keeps a split condition when one of its providers goes', () => {
    const split: Condition = {
      field: 'sender_domain',
      values: ['updates.acme.io'],
      providerId: ses,
      splits: [
        { providerId: ses, weight: 60 },
        { providerId: postal, weight: 30 },
        { providerId: postmark, weight: 10 },
      ],
      note: 'product announcements',
    }
    expect(conditionWithout(split, postmark)).toEqual([{ ...split, splits: [{ providerId: ses, weight: 70 }, { providerId: postal, weight: 30 }] }])
    const two = conditionWithout(split, postmark)[0]!
    expect(conditionWithout(two, ses)).toEqual([{ ...split, providerId: postal, splits: undefined }])
    expect(conditionWithout({ field: 'list', values: ['x'], providerId: postal }, postal)).toEqual([])
    const d = toDraft(cfg)
    d.broadcast.conditions = [split]
    expect(providerUses(d, postmark)).toContainEqual({ kind: 'condition', stream: 'broadcast', values: 'updates.acme.io' })
    expect(toInput(d).streams[0]!.conditions).toEqual([{ field: 'sender_domain', values: ['updates.acme.io'], splits: split.splits, note: 'product announcements' }])
    expect(conditionProviders({ providerId: ses })).toEqual([ses])
  })
})
