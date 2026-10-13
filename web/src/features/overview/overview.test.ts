import { describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { bounceLabel, clockIn, complaintLabel, costCell, dateIn, delta, minutes, netBadge, quotaRow, rateCell, readinessBadge, refreshSentences, routeLine } from './words'

const limit = (o: Partial<Schemas['ProviderLimitRate']>) => ({ limit: 5, stale: false, ...o }) as Schemas['ProviderLimitRate']

describe('overview words', () => {
  it('labels limits like the board', () => {
    expect(bounceLabel(limit({ limit: 5, review: 5, pause: 10, label: 'SES' }))).toBe('Bounce rate · limit 5% (SES review)')
    expect(bounceLabel(limit({ limit: 10, label: 'Postmark' }))).toBe('Bounce rate · limit 10%')
    expect(bounceLabel(limit({ limit: 2, label: 'warm-up guard' }))).toBe('Bounce rate · warm-up limit 2%')
    expect(complaintLabel(limit({ limit: 0.1, review: 0.1, label: 'SES' }))).toBe('Complaint rate · limit 0.1%')
  })

  it('reads the next campaign', () => {
    expect(readinessBadge({ verdict: 'warnings', issues: [{ status: 'warn', title: 'a' }, { status: 'warn', title: 'b' }], ranAt: '' })).toEqual({ tone: 'warn', text: 'Ready with 2 warnings' })
    expect(readinessBadge({ verdict: 'blocked', issues: [{ status: 'block', title: 'a' }], ranAt: '' })).toEqual({ tone: 'bad', text: 'Blocked by 1 check' })
    expect(readinessBadge(undefined).text).toBe('Not checked yet')
    expect(routeLine([{ short: 'SES', adapter: 'ses', color: 'ses', percent: 90, planned: 1 }, { short: 'Postal', adapter: 'postal', color: 'postal', percent: 10, planned: 1 }])).toBe('SES 90 / Postal 10')
    expect(minutes(34)).toBe('34 min')
    expect(minutes(72)).toBe('1 h 12 min')
    expect(minutes(undefined)).toBe('—')
    expect(dateIn('2026-10-14T13:00:00Z', 'America/New_York')).toBe('Wednesday 14 Oct 2026')
    expect(clockIn('2026-10-14T13:00:00Z', 'America/New_York')).toBe('09:00')
  })

  it('writes provider rows', () => {
    const card = (o: Partial<Schemas['ProviderCard']>) =>
      ({ quota: { kind: 'none', max: 0, used: 0, fromFailover: 0 }, rate: { max: 0, now: 0, peakToday: 0, throttled: false }, cost: { basis: 'metered', toDate: 0 }, ...o }) as Schemas['ProviderCard']
    const now = new Date('2026-10-13T15:00:00Z')
    expect(quotaRow(card({ quota: { kind: 'rolling_24h', max: 50000, used: 2140, fromFailover: 0 } }), now)).toEqual({ label: '24 h quota · 50,000', value: '2,140 used (4%)', percent: 4 })
    expect(quotaRow(card({ quota: { kind: 'monthly_plan', max: 10000, used: 3412, fromFailover: 0 } }), now)?.label).toBe('Plan · 10,000 / month')
    expect(quotaRow(card({ quota: { kind: 'warmup_daily', max: 6000, used: 0, fromFailover: 0 } }), now)?.value).toMatch(/^0 sent \(\w+day\)$/)
    expect(rateCell(card({ rate: { max: 14, now: 0, peakToday: 0, throttled: false } }))).toEqual({ label: 'Max rate', value: '14 /s' })
    expect(rateCell(card({ rate: { max: 2, now: 0, peakToday: 0, throttled: false, label: 'ramp limit' } }))).toEqual({ label: 'Rate (ramp)', value: '2 /s' })
    expect(rateCell(card({})).value).toBe('No limit set')
    expect(costCell(card({ cost: { basis: 'metered', toDate: 14.62 } }))).toBe('$14.62')
    expect(costCell(card({ cost: { basis: 'plan', toDate: 15, monthly: 15 } }))).toBe('$15.00 plan')
    expect(costCell(card({ cost: { basis: 'self_hosted', toDate: 0, monthly: 24 } }))).toBe('$0 metered')
  })

  it('writes growth', () => {
    const a = { total: 48213, subscribers: 48213, days: 30, net: 1284, netPercent: 2.7, series: [], lists: [], tiers: [], policy: '', policyEnabled: true }
    expect(netBadge(a)).toEqual({ tone: 'ok', text: '+1,284 net · +2.7%' })
    expect(netBadge({ ...a, net: -312, netPercent: -0.6 })).toEqual({ tone: 'bad', text: '−312 net · −0.6%' })
    expect(delta(812)).toBe('+812')
    expect(delta(0)).toBe('±0')
  })

  it('explains a failing health refresh like the states board', () => {
    const p = {
      displayName: 'Amazon SES',
      detail: '',
      card: { short: 'SES', webhook: { kind: 'SNS' } },
      refresh: { op: 'GetAccount', status: 503, code: 'ServiceUnavailable', attempts: ['a', 'b', 'c'], lastSuccessAt: 'd' },
    } as unknown as Schemas['OverviewProviderCard']
    const s = refreshSentences(p, (d) => ({ a: '09:41', b: '09:42', c: '09:44', d: '09:12' })[d] ?? d)
    expect(s?.title).toBe("Couldn't refresh Amazon SES account health")
    expect(s?.body).toBe(
      'SES GetAccount returned 503 ServiceUnavailable three times (09:41, 09:42, 09:44). Bounce and complaint rates below are from the last successful fetch at 09:12. Routing and sending continue; SES events still arrive through SNS.',
    )
  })
})
