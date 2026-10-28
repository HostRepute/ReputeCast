import { describe, expect, it } from 'vitest'
import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { abHeadline, abHow, abLead } from './ABTestCard'
import { anchorsOf, bubbleWidth, entries, placeBubbles, screenRulesForPrint, shownVariant, withFallbacks, type AnchorBox, type Spot } from './ClickMap'
import { hourX, seriesPath, tableHours } from './EngagementChart'
import { linkMeta } from './ShareCard'
import { abExplanation, bounceSub, comparison, engagementNote, fewerNote, headerCaption, niceMax, notReportedLine, providerNote, splitCaption, trendLabel, variantLine } from './words'

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
    expect(headerCaption(info({ finalAt: '2026-10-12T16:41:00Z' }))).toMatch(/^All subscribers · A\/B subject, winner B · results final since \w{3} 12 Oct, \d\d:41$/)
    expect(headerCaption(info({}))).toBe('All subscribers · A/B subject, winner B · results final after 72 h')
    expect(headerCaption(info({ final: false, finalAt: '2026-10-12T16:41:00Z' }))).toMatch(/^All subscribers · A\/B subject, winner B · results still coming in · final \w{3} 12 Oct, \d\d:41$/)
    expect(headerCaption(info({ origin: 'brevo', notes: [] }))).toBe('All subscribers · imported from Brevo')
  })

  it('splits bounces into hard, soft and blocked or other', () => {
    const h = { bounceRate: 0.6003, bounces: 288, hardBounces: 193, softBounces: 79 } as Schemas['ReportHeadline']
    expect(bounceSub(h, true)).toBe('0.60% of sent · 193 hard, 79 soft, 16 blocked or other')
    expect(bounceSub({ ...h, bounces: 272 }, true)).toBe('0.60% of sent · 193 hard, 79 soft')
    expect(bounceSub(h, false)).toBe('0.60% of sent so far')
  })

  it('says which mailbox providers do not report complaints', () => {
    expect(notReportedLine({ notReportedBy: ['Gmail', 'iCloud'] } as unknown as Schemas['ReportHeadline'])).toBe("Gmail and iCloud don't report complaints")
    expect(notReportedLine({ notReportedBy: ['Gmail'] } as unknown as Schemas['ReportHeadline'])).toBe("Gmail doesn't report complaints")
    expect(notReportedLine({ notReportedBy: [] } as unknown as Schemas['ReportHeadline'])).toBeNull()
  })

  it('compares each tile with the median of the last issues', () => {
    const c: Schemas['ReportComparisons'] = {
      issues: 6,
      tiles: [
        { metric: 'sent', change: 2.4, unit: 'percent', median: 46856, tone: 'neutral', trend: [] },
        { metric: 'delivered', change: -0.1, unit: 'points', median: 99.2, tone: 'bad', trend: [] },
        { metric: 'humanOpens', change: 2.1, unit: 'points', median: 50.7, tone: 'good', trend: [] },
        { metric: 'unsubscribes', change: 0.03, unit: 'points', median: 0.21, tone: 'bad', trend: [] },
        { metric: 'complaints', change: -0.01, unit: 'points', median: 0.1, tone: 'good', trend: [] },
        { metric: 'bounces', change: 0, unit: 'points', median: 0.6, tone: 'neutral', trend: [] },
      ],
    }
    expect(comparison(c, 'sent')).toEqual({ arrow: '▲', direction: 'Up', text: '2.4% vs your last 6 issues', tone: 'neu' })
    expect(comparison(c, 'delivered')).toEqual({ arrow: '▼', direction: 'Down', text: '0.1 pts vs your last 6 issues', tone: 'neg' })
    expect(comparison(c, 'humanOpens')?.tone).toBe('pos')
    expect(comparison(c, 'unsubscribes')?.text).toBe('0.03 pts vs your last 6 issues')
    expect(comparison(c, 'complaints')).toEqual({ arrow: '▼', direction: 'Down', text: '0.01 pts vs your last 6 issues', tone: 'pos' })
    expect(comparison(c, 'bounces')?.text).toBe('Same as your last 6 issues')
    expect(comparison(c, 'clicks')).toBeUndefined()
    expect(comparison({ issues: 1, tiles: [c.tiles[0]!] }, 'sent')?.text).toBe('2.4% vs your last issue')
  })

  it('describes the split and a warming provider by its gap in human opens of measurable', () => {
    const col = (o: Partial<Schemas['ReportProviderColumn']>) =>
      ({
        providerId: 'p',
        name: 'Amazon SES',
        short: 'SES',
        adapter: 'ses',
        color: 'ses',
        sent: 43182,
        delivered: 1,
        percent: 90,
        cost: 0,
        engagedFirst: false,
        trackedBy: 'reputecast',
        deliveryInferred: false,
        events: 'sns',
        measurable: 1,
        ...o,
      }) as Schemas['ReportProviderColumn']
    const x = {
      split: true,
      columns: [
        col({ humanOpenRate: 41.38, humanOpenRateMeasurable: 53.0 }),
        col({ name: 'Postal', short: 'Postal', percent: 10, sent: 4798, humanOpenRate: 39.56, humanOpenRateMeasurable: 50.6, warmupDay: 6, engagedFirst: true }),
      ],
    }
    expect(splitCaption(x)).toBe('side by side · the send was split 90 / 10')
    expect(providerNote(x)).toBe('Postal was on warm-up day 6 and sent to the Active tier only; a 2.4-point gap in human opens on a young IP is normal.')
    expect(providerNote({ split: false, columns: [x.columns[0]!] })).toBe('All of it went through Amazon SES.')
  })

  it('writes the engagement note with the rollout', () => {
    const e = { available: true, hours: 72, delivered: 100, points: [], humanWithin6h: 62.1, rollout: { variant: 'B', at: '2026-10-09T16:00:00Z', hour: 2 } }
    expect(engagementNote(e, true, '2026-10-09T14:00:00Z', 38384)).toMatch(
      /^62% of human opens came within 6 hours\. The samples went out at \d\d:00; the steep rise starts at \d\d:00, when B went to the other 38,384\.$/,
    )
    expect(engagementNote({ ...e, rollout: undefined }, false)).toBe('So far, 62% of human opens came within 6 hours.')
    expect(fewerNote(28)).toBe('28 people chose “fewer emails” instead of leaving and moved to the monthly frequency.')
    expect(fewerNote(0)).toBeNull()
  })

  it('picks a scale above the largest value', () => {
    expect(niceMax(41.2)).toEqual({ max: 50, step: 10 })
    expect(niceMax(6.7)).toEqual({ max: 8, step: 2 })
    expect(niceMax(0)).toEqual({ max: 1, step: 1 })
    expect(niceMax(73)).toEqual({ max: 80, step: 20 })
  })

  it('keeps the testing state of an A/B test', () => {
    const x = {
      kind: 'subject',
      label: 'subject line',
      metric: 'human_opens',
      windowHours: 2,
      state: 'testing',
      default: 'A',
      variants: [],
      significant: false,
      minSample: 500,
      threshold: 95,
      remaining: 0,
      barMax: 10,
      decidedAt: '2026-10-09T16:00:00Z',
    } as Schemas['ReportABTest']
    expect(abHeadline(t, x)).toEqual({ tone: 'info', text: expect.stringMatching(/^Testing · decides at \d\d:00$/) as string })
    expect(abExplanation(x)[0]).toBe('Human opens 2 hours after the sample went out decide it. Machine opens were excluded so Apple Mail prefetches couldn’t decide it.')
    expect(variantLine({ key: 'B', label: '', recipients: 4798, samplePercent: 10, winner: true, remaining: 38384, default: false, eligible: 0, conversions: 0 })).toBe(
      '4,798 recipients · 10% sample, then 38,384 remaining',
    )
  })
})

describe('A/B card', () => {
  const v = (o: Partial<Schemas['ReportABVariant']>) =>
    ({ key: 'A', label: '', default: false, recipients: 4798, samplePercent: 10, eligible: 4752, conversions: 124, winner: false, remaining: 0, ...o }) as Schemas['ReportABVariant']
  const won = {
    kind: 'subject',
    label: 'subject line',
    metric: 'clicks',
    windowHours: 2,
    decidedAfter: 2,
    state: 'decided',
    outcome: 'winner',
    winner: 'B',
    default: 'A',
    leader: 'B',
    runnerUp: 'A',
    lift: 0.8,
    confidence: 97.9,
    significant: true,
    minSample: 500,
    threshold: 95,
    remaining: 38384,
    rolloutAt: '2026-10-09T16:00:00Z',
    decidedAt: '2026-10-09T16:00:00Z',
    barMax: 10,
    variants: [v({ key: 'A', default: true, decisionRate: 2.6 }), v({ key: 'B', winner: true, remaining: 38384, decisionRate: 3.4 })],
  } as Schemas['ReportABTest']

  it('explains how the winner was picked, as the board does', () => {
    const how = abHow(t, won, true)
    expect(how[0]).toMatch(
      /^Two-proportion z-test on human clicks 2 h after the sample went out \(B 3\.4%, A 2\.6%\); at least 500 recipients per variant; the remaining 38,384 got B at \d\d:00\.$/,
    )
    expect(how[1]).toBe("Opens couldn't decide it, because Apple Mail opens messages automatically. The bars show human clicks after 72 h, on a 0 to 10% scale.")
    expect(abLead(t, won)).toBeNull()
  })

  it('claims a confidence only for a significant decision', () => {
    expect(abHeadline(t, won)).toEqual({ tone: 'ok', text: 'Winner B · 98% confidence · decided on clicks' })
    expect(abHeadline(t, { ...won, confidence: 99.97 }).text).toBe('Winner B · 99% confidence · decided on clicks')
    expect(abHeadline(t, { ...won, metric: 'human_opens' }).text).toBe('Winner B · 98% confidence · decided on human opens')
    expect(abHeadline(t, { ...won, confidence: undefined }).text).toBe('Winner B')
    // Numbers that don't reach significance never read as a confident win.
    expect(abHeadline(t, { ...won, significant: false, confidence: 61.3 })).toEqual({ tone: 'warn', text: 'B chosen · not significant' })
    expect(abHeadline(t, { ...won, outcome: 'manual' })).toEqual({ tone: 'ok', text: 'Winner B · picked by a person' })
    expect(abHeadline(t, { ...won, state: 'no_winner', outcome: 'not_significant', winner: 'A' })).toEqual({ tone: 'neutral', text: 'No clear winner' })
    expect(abHeadline(t, { ...won, state: 'testing', outcome: undefined, decidedAt: undefined })).toEqual({ tone: 'info', text: 'Testing' })
  })

  it('claims no confidence for a test decided before decisions were kept', () => {
    const legacy = {
      ...won,
      outcome: 'unrecorded',
      significant: false,
      confidence: undefined,
      pValue: undefined,
      lift: undefined,
      leader: undefined,
      runnerUp: undefined,
      decidedAfter: undefined,
      variants: [v({ key: 'A', default: true, metricRate: 5.6 }), v({ key: 'B', winner: true, remaining: 38384, metricRate: 6.7 })],
    } as Schemas['ReportABTest']
    expect(abHeadline(t, legacy)).toEqual({ tone: 'neutral', text: 'B chosen · confidence not recorded' })
    const how = abHow(t, legacy, true)
    expect(how[0]).toMatch(/^This test was decided before ReputeCast kept a record of each decision, so how sure it was isn't known\. The remaining 38,384 got B at \d\d:00\.$/)
    expect(how[1]).toBe('The bars show human clicks after 72 h, on a 0 to 10% scale.')
    expect(how.join(' ')).not.toMatch(/confidence|significant|z-test|couldn't decide/)
    expect(abHow(t, { ...legacy, remaining: 0 }, false)).toEqual(["This test was decided before ReputeCast kept a record of each decision, so how sure it was isn't known."])
  })

  it('says so when there is no clear winner, and that the default went out', () => {
    const none = {
      ...won,
      state: 'no_winner',
      outcome: 'not_significant',
      winner: 'A',
      lift: 0.4,
      confidence: 61.3,
      significant: false,
      remaining: 25284,
      decidedAfter: 3,
      variants: [v({ key: 'A', default: true, winner: true, remaining: 25284, decisionRate: 3.6 }), v({ key: 'B', decisionRate: 4.0 })],
    } as Schemas['ReportABTest']
    expect(abLead(t, none)).toBe('B was 0.4 pts ahead at 61% confidence, so everyone else got A (your default).')
    const how = abHow(t, none, false, '2026-10-24T16:41:00Z')
    expect(how[0]).toMatch(/A winner needs 95% confidence, so the remaining 25,284 got A, your default, at \d\d:00\.$/)
    expect(how[1]).toMatch(/^Both versions' final numbers arrive on \w{3} 24 Oct, \d\d:41\. The bars run from 0% to 10%\.$/)
    expect(abLead(t, { ...none, outcome: 'too_small' })).toBe('Fewer than 500 recipients per variant could be measured, so everyone else got A (your default).')
  })
})

describe('engagement chart', () => {
  it('runs the time axis evenly over 72 hours', () => {
    expect(hourX(0)).toBe(0)
    expect(hourX(2)).toBe(20)
    expect(hourX(36)).toBe(360)
    expect(hourX(72)).toBe(720)
    expect(hourX(80)).toBe(720)
  })

  it('draws shares of delivered from the send, on the scale given', () => {
    const d = seriesPath([{ hour: 72, humanOpens: 412, privacyOpens: 219, machineOpens: 12, clicks: 63 }], 'humanOpens', 1000, 50)
    expect(d).toBe('M0.0 200.0 L720.0 35.2')
  })

  it('lists the rollout hour first in the table', () => {
    const e = { available: true, hours: 72, delivered: 1, points: [], rollout: { variant: 'B', at: '', hour: 2 } }
    expect(tableHours(e)).toEqual([
      { hour: 2, rollout: true },
      { hour: 6, rollout: false },
      { hour: 24, rollout: false },
      { hour: 48, rollout: false },
      { hour: 72, rollout: false },
    ])
    expect(tableHours({ ...e, hours: 20, rollout: undefined }).map((x) => x.hour)).toEqual([6])
  })
})

describe('click map table', () => {
  const link = (o: Partial<Schemas['ReportLink']>) =>
    ({ id: '', url: 'u', label: 'l', position: 0, clicks: 1, uniqueClicks: 1, botClicks: 0, share: 10, grouped: false, hot: false, ...o }) as Schemas['ReportLink']
  it('orders links by share, then rows, keeping row links out of Other', () => {
    const s = {
      threshold: 5,
      html: '',
      available: true,
      total: 3412,
      botTotal: 1108,
      links: [
        link({ position: 0, label: 'Acme', share: 3.1, grouped: true }),
        link({ position: 1, label: 'Blog', share: 2.4 }),
        link({ position: 2, label: 'Docs', share: 2.0 }),
        link({ position: 6, label: 'Read the docs', share: 38 }),
        link({ position: 9, label: 'Read the guide', share: 19 }),
        link({ position: 8, label: 'See changelog', share: 24 }),
      ],
      rows: [{ place: 'header' as const, positions: [1, 2], labels: ['Blog', 'Docs'], clicks: 150, botClicks: 222, share: 4.4 }],
    }
    const e = entries(s)
    expect(e.links.map((l) => l.label)).toEqual(['Read the docs', 'See changelog', 'Read the guide'])
    expect(e.rows.length).toBe(1)
    expect(e.grouped.map((l) => l.label)).toEqual(['Acme'])
  })
})

describe('click map bubbles', () => {
  const box = (o: Partial<AnchorBox>): AnchorBox => ({ top: 0, bottom: 20, right: 100, rowRight: 100, endsLine: false, bandTop: 0, blockRight: 600, button: false, ...o })
  const spot = (o: Partial<Spot>): Spot => ({ key: '0', label: 'l', text: '10%', anchor: 0, hot: false, ...o })
  it('puts bubbles by buttons, after text links and their row, and Other on the footer', () => {
    const anchors = [box({ top: 10, bottom: 30, right: 200, rowRight: 280 }), box({ top: 300, bottom: 334, right: 120, button: true }), box({ top: 500, bottom: 520, bandTop: 480, blockRight: 600 })]
    const out = placeBubbles(
      [spot({ key: 'row-0', text: '6%', anchor: 0 }), spot({ key: '1', text: '38%', anchor: 1, hot: true })],
      anchors,
      { text: '13%', label: 'Other', anchors: [2] },
      0.5,
      300,
    )
    expect(out.map((b) => [b.key, b.left, b.top])).toEqual([
      ['row-0', 144, 0 + 2],
      ['1', 52, 159],
      ['other', 256, 232],
    ])
    expect(out[1]!.hot).toBe(true)
  })

  it('sizes bubbles by their text', () => {
    expect([bubbleWidth('6%'), bubbleWidth('38%'), bubbleWidth('100%'), bubbleWidth('74'), bubbleWidth('1,108')]).toEqual([34, 42, 49, 31, 49])
  })

  it('puts the bubble of a link that ends its line at the right edge of its block, as the board does', () => {
    const out = placeBubbles([spot({ text: '19%' })], [box({ top: 100, bottom: 120, right: 300, rowRight: 300, endsLine: true, blockRight: 560 })], undefined, 0.5, 300)
    expect(out.map((b) => [b.left, b.top])).toEqual([[280 - 42, 44]])
  })

  it('keeps a right-aligned navigation row clear: its bubble goes after the row', () => {
    const scale = 340 / 600
    const out = placeBubbles([spot({ text: '6%' })], [box({ top: 20, bottom: 38, right: 430, rowRight: 518, endsLine: true, blockRight: 518 })], undefined, scale, 340)
    expect(out[0]!.left).toBeCloseTo(518 * scale + 4)
  })

  it('puts the bubble of a link that ends its line under it, not over its words', () => {
    const out = placeBubbles([spot({ text: '73%', hot: true })], [box({ top: 400, bottom: 420, right: 590, rowRight: 590 })], undefined, 0.5, 300)
    expect(out.map((b) => [b.left, b.top])).toEqual([[295 - 42, 212]])
  })
})

describe('click map preview', () => {
  it('shows personalization fallbacks as a reader without the field sees them', () => {
    expect(withFallbacks('Hi {{ first_name | default: "there" }}, five things shipped.')).toBe('Hi there, five things shipped.')
    expect(withFallbacks("Hi {{first_name|default:'friend'}}!")).toBe('Hi friend!')
    expect(withFallbacks('Plan: {{ plan | upcase | default: "free" }}')).toBe('Plan: free')
    expect(withFallbacks('Hello {{ first_name }}')).toBe('Hello {{ first_name }}')
  })

  it('keeps the email’s desktop layout when the report is printed', () => {
    expect(screenRulesForPrint('@media only screen and (min-width:480px) { .mj-column-per-50 { width:50% } }')).toBe('@media (min-width:480px) { .mj-column-per-50 { width:50% } }')
    expect(screenRulesForPrint('@media screen and (max-width: 479px){a{b:c}}')).toBe('@media (max-width: 479px){a{b:c}}')
    expect(screenRulesForPrint('@media only screen { p { margin:0 } }')).toBe('@media all { p { margin:0 } }')
    expect(screenRulesForPrint('@media print { .x { display:none } }')).toBe('@media print { .x { display:none } }')
  })
})

describe('click map links', () => {
  it('follows the positions the report marks, not its own rule', () => {
    const doc = new DOMParser().parseFromString(
      `<a href="mailto:x@y">mail</a><a data-rc-link="1" href="https://acme.io/b">B</a><a href="{{ unsubscribe_url }}">unsub</a><a data-rc-link="0" href="https://acme.io/a">A</a><a data-rc-link="x" href="https://acme.io/c">C</a>`,
      'text/html',
    )
    const anchors = anchorsOf(doc)
    expect(anchors.length).toBe(2)
    expect(anchors[0]?.textContent).toBe('A')
    expect(anchors[1]?.textContent).toBe('B')
  })
})

describe('share links', () => {
  it('say who made them, when they expire and what they include', () => {
    const s = {
      id: '1',
      url: 'https://reputecast.acme.io/r/8f2c1a9d',
      createdAt: '2026-10-21T10:00:00Z',
      createdByName: 'Marcus Chen',
      expiresAt: '2026-11-20T10:00:00Z',
      views: 0,
      includes: { cost: true, warmup: false, email: true },
    }
    expect(linkMeta(s)).toBe('Created by Marcus Chen on 21 Oct · expires 20 Nov · with the email and cost per provider')
    expect(linkMeta({ ...s, includes: { cost: false, warmup: false, email: false } })).toBe('Created by Marcus Chen on 21 Oct · expires 20 Nov')
  })
})

describe('trend lines', () => {
  const c = (o: Partial<Schemas['ReportComparison']>) =>
    ({ metric: 'humanOpens', change: 2.1, unit: 'points', median: 50.7, tone: 'good', trend: [], ...o }) as Schemas['ReportComparison']
  it('read the line out the way the tile writes its numbers', () => {
    expect(trendLabel(c({ trend: [49.92, 50.4, 49.7, 50.71, 51.03, 50.6, 52.79] }))).toBe('Up over your last 6 issues, from 49.9% to 52.8%')
    expect(trendLabel(c({ metric: 'sent', trend: [46000, 47980] }))).toBe('Up since your last issue, from 46,000 to 47,980')
    // Small rates keep two decimals, as their tiles do.
    expect(trendLabel(c({ metric: 'complaints', trend: [0.06, 0.05, 0.041] }))).toBe('Down over your last 2 issues, from 0.06% to 0.04%')
    // The same at the tile's precision reads as the same, not as a change.
    expect(trendLabel(c({ metric: 'delivered', trend: [99.12, 98.7, 99.08] }))).toBe('Same as 2 issues ago, at 99.1%')
    expect(trendLabel(c({ trend: [52.79] }))).toBeUndefined()
  })
})

describe('click map variants', () => {
  const vm = (key: string) => ({ key, sameEmail: key === 'A', html: '', available: true, total: 0, botTotal: 0, links: [], rows: [] }) as Schemas['ReportVariantClickMap']
  it('starts a test on content on its first variant: no map holds everyone', () => {
    expect(shownVariant({ perVariant: true, variants: [vm('A'), vm('B')] }, '')).toBe('A')
    expect(shownVariant({ perVariant: true, variants: [vm('A'), vm('B')] }, 'B')).toBe('B')
    expect(shownVariant({ perVariant: true, variants: [vm('A'), vm('B')] }, 'C')).toBe('A')
    expect(shownVariant({ perVariant: false, variants: [vm('A'), vm('B')] }, '')).toBe('')
    expect(shownVariant({ perVariant: false, variants: [vm('A'), vm('B')] }, 'B')).toBe('B')
    expect(shownVariant({ perVariant: false, variants: [] }, 'B')).toBe('')
  })
})
