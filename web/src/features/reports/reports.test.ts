import { describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { placeBubbles, type AnchorBox } from './ClickMap'
import { hourX, seriesPath } from './EngagementChart'
import { abBadge, abExplanation, bounceLine, clickNote, engagementNote, fewerNote, headerCaption, providerNote, splitCaption, variantLine } from './words'

const info = (over: Partial<Schemas['ReportCampaignInfo']>): Schemas['ReportCampaignInfo'] => ({
  id: 'c',
  name: 'October Product Update',
  status: 'sent',
  audience: 'All subscribers',
  final: true,
  notes: ['A/B subject, winner B'],
  ...over,
})

describe('report words', () => {
  it('captions the header like the board', () => {
    expect(headerCaption(info({}))).toBe('All subscribers · A/B subject, winner B · results final after 72 h')
    expect(headerCaption(info({ final: false, finalAt: '2026-10-12T16:41:00Z' }))).toMatch(/^All subscribers · A\/B subject, winner B · results still coming in · final \w{3} 12 Oct, \d\d:41$/)
    expect(headerCaption(info({ origin: 'brevo', notes: [] }))).toBe('All subscribers · imported from Brevo')
  })

  it('explains bounces that never resolved', () => {
    const h = { bounceRate: 0.6003, undelivered: 144 } as Schemas['ReportHeadline']
    expect(bounceLine(h, true)).toBe('0.60% · 144 expired deferrals')
    expect(bounceLine(h, false)).toBe('0.60% · 144 not delivered yet')
    expect(bounceLine({ ...h, undelivered: 0 }, true)).toBe('0.60%')
  })

  it('describes the split and a warming provider', () => {
    const col = (o: Partial<Schemas['ReportProviderColumn']>) =>
      ({ providerId: 'p', name: 'Amazon SES', short: 'SES', adapter: 'ses', color: 'ses', sent: 43182, delivered: 1, percent: 90, cost: 0, engagedFirst: false, ...o }) as Schemas['ReportProviderColumn']
    const t = { split: true, columns: [col({ humanOpenRate: 41.38 }), col({ name: 'Postal', short: 'Postal', percent: 10, sent: 4798, humanOpenRate: 39.56, warmupDay: 6, engagedFirst: true })] }
    expect(splitCaption(t)).toBe('side by side · the send was split 90 / 10')
    expect(providerNote(t)).toBe('Postal was on warm-up day 6 and sent to the Active tier only; a 1.8-point open gap on a young IP is normal.')
    expect(providerNote({ split: false, columns: [t.columns[0]!] })).toBe('All of it went through Amazon SES.')
  })

  it('writes the engagement and click notes', () => {
    const e = { available: true, hours: 72, delivered: 100, points: [], humanWithin6h: 62.1, rollout: { variant: 'B', at: '2026-10-09T16:00:00Z', hour: 2 } }
    expect(engagementNote(e, true)).toMatch(/^62% of human opens happened within 6 hours; A\/B winner B went out at \d\d:00, which is the second rise\.$/)
    expect(engagementNote({ ...e, rollout: undefined }, false)).toBe('So far, 62% of human opens happened within 6 hours.')
    const m = { html: '', available: true, total: 2996, links: [], threshold: 5, other: { share: 13, links: 9, clicks: 390, labels: ['Email preferences', 'View in browser'] } }
    expect(clickNote(m)).toBe('Links with under 5% of clicks are grouped: “Other” 13% covers 9 links including email preferences and view in browser.')
    expect(fewerNote(28)).toBe('28 people chose “fewer emails” instead of leaving and moved to the monthly frequency.')
    expect(fewerNote(0)).toBeNull()
  })

  it('says honestly when an A/B result is not significant', () => {
    const t = {
      kind: 'subject',
      label: 'subject line',
      metric: 'human_opens',
      windowHours: 2,
      state: 'decided',
      winner: 'B',
      variants: [],
      lift: 2.8,
      pValue: 0.0225,
      confidence: 97.8,
      significant: true,
      remaining: 38384,
      rolloutAt: '2026-10-09T16:00:00Z',
    } as Schemas['ReportABTest']
    expect(abBadge(t)).toEqual({ tone: 'ok', text: 'Winner B · 98% confidence' })
    expect(abExplanation(t)[0]).toBe('Human opens 2 hours after the sample went out, two-proportion z-test, lift +2.8 points, p = 0.02. Machine opens were excluded so Apple Mail prefetches couldn’t decide it.')
    expect(abExplanation(t)[1]).toMatch(/^The remaining 38,384 recipients got B at \d\d:00\.$/)
    const weak = { ...t, state: 'no_winner', significant: false, confidence: 81.2, pValue: 0.188, winner: undefined } as Schemas['ReportABTest']
    expect(abBadge(weak)).toEqual({ tone: 'warn', text: 'Not significant yet · 81% confidence' })
    expect(abExplanation(weak)[0]).toContain('not significant, so the difference could be chance')
    expect(abBadge({ ...t, confidence: 99.9 }).text).toBe('Winner B · 99% confidence')
    expect(variantLine({ key: 'B', label: '', recipients: 4798, samplePercent: 10, winner: true, remaining: 38384 })).toBe('4,798 recipients · 10% sample, then 38,384 remaining')
  })
})

describe('engagement chart', () => {
  it('bends the time axis at 6, 24 and 48 hours', () => {
    expect(hourX(0)).toBe(40)
    expect(hourX(6)).toBe(205)
    expect(hourX(24)).toBe(370)
    expect(hourX(48)).toBe(535)
    expect(hourX(72)).toBe(700)
    expect(hourX(3)).toBeCloseTo(122.5)
  })
  it('draws shares of delivered, the top series at 93% of the height', () => {
    const d = seriesPath(
      [
        { hour: 0, humanOpens: 0, machineOpens: 0, clicks: 0 },
        { hour: 72, humanOpens: 412, machineOpens: 186, clicks: 63 },
      ],
      'humanOpens',
      1000,
      41.2 / 0.93,
    )
    expect(d).toBe('M40.0 190.0 L700.0 22.6')
  })
})

describe('click map bubbles', () => {
  const box = (o: Partial<AnchorBox>): AnchorBox => ({ top: 0, bottom: 20, right: 100, rowRight: 100, blockTop: 0, blockRight: 600, button: false, ...o })
  const link = (o: Partial<Schemas['ReportLink']>) => ({ id: '', url: 'u', label: 'l', position: 0, clicks: 1, uniqueClicks: 1, share: 10, grouped: false, hot: false, ...o }) as Schemas['ReportLink']
  it('puts bubbles by buttons, after text links and their row, and Other on the footer', () => {
    const anchors = [box({ top: 10, bottom: 30, right: 200, rowRight: 280 }), box({ top: 300, bottom: 334, right: 120, button: true }), box({ top: 500, bottom: 520, blockTop: 480, blockRight: 600 })]
    const out = placeBubbles(
      [link({ position: 0, share: 6 }), link({ position: 1, share: 38, hot: true }), link({ position: 2, share: 1.5, grouped: true })],
      anchors,
      { share: 13, links: 9, clicks: 1, labels: [] },
      0.5,
      300,
    )
    expect(out.map((b) => [b.key, b.left, b.top])).toEqual([
      ['0', 144, 0 + 2],
      ['1', 52, 159],
      ['other', 250, 232],
    ])
    expect(out[1]!.hot).toBe(true)
  })
})
