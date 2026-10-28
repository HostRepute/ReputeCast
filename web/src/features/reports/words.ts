import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import { t } from '@/i18n'
import { andList, clock, count, decimal, longDate, pct, shortDate } from '@/lib/format'
import { m } from './messages'

export type CampaignReport = Schemas['CampaignReport']
export type ReportInfo = Schemas['ReportCampaignInfo']

/** "41.2%": rates of delivered or sent keep one decimal. */
export function pct1(v: number | null | undefined): string {
  return pct(v, 1)
}

/** "0.04%": small rates (complaints, unsubscribes) keep two decimals. */
export function pct2(v: number | null | undefined): string {
  return pct(v, 2)
}

/** "Thu 9 Oct, 10:00 to 12:41", or with both dates when it ran past midnight. */
export function sendWindow(start?: string, end?: string): string {
  if (!start) return ''
  const day = shortDate(start)
  if (!end) return t(m.windowStart, { day, time: clock(start) })
  return day === shortDate(end)
    ? t(m.windowSameDay, { day, start: clock(start), end: clock(end) })
    : t(m.windowTwoDays, { day, start: clock(start), endDay: shortDate(end), end: clock(end) })
}

/** The status badge of the report header. */
export function statusBadge(c: ReportInfo): { tone: Tone; text: string } {
  switch (c.status) {
    case 'sent':
      return { tone: 'ok', text: c.startedAt ? t(m.statusSentWindow, { window: sendWindow(c.startedAt, c.completedAt) }) : t(m.statusSent) }
    case 'sending':
      return { tone: 'info', text: c.startedAt ? t(m.statusSendingSince, { time: clock(c.startedAt) }) : t(m.statusSending) }
    case 'paused':
      return { tone: 'warn', text: t(m.statusPaused) }
    case 'cancelled':
      return { tone: 'neutral', text: c.startedAt ? t(m.statusCancelledWindow, { window: sendWindow(c.startedAt, c.completedAt) }) : t(m.statusCancelled) }
    case 'failed':
      return { tone: 'bad', text: t(m.statusFailed) }
    case 'scheduled':
      return { tone: 'info', text: c.scheduledAt ? t(m.statusScheduledAt, { day: shortDate(c.scheduledAt), time: clock(c.scheduledAt) }) : t(m.statusScheduled) }
    case 'approval':
      return { tone: 'warn', text: t(m.statusApproval) }
    case 'approved':
      return { tone: 'info', text: t(m.statusApproved) }
  }
  return { tone: 'neutral', text: t(m.statusDraft) }
}

/** "All subscribers · A/B subject, winner B · results final since Sun 12 Oct, 12:41". */
export function headerCaption(c: ReportInfo): string {
  const parts: string[] = []
  if (c.audience) parts.push(c.audience)
  parts.push(...c.notes)
  if (c.origin) {
    parts.push(t(m.importedFrom, { origin: `${c.origin[0]!.toUpperCase()}${c.origin.slice(1)}` }))
  } else if (c.status === 'sent' || c.status === 'cancelled') {
    if (c.final && c.finalAt) parts.push(t(m.resultsFinalSince, { day: shortDate(c.finalAt), time: clock(c.finalAt) }))
    else if (c.final) parts.push(t(m.resultsFinal))
    else if (c.finalAt) parts.push(t(m.resultsComing, { day: shortDate(c.finalAt), time: clock(c.finalAt) }))
  } else if (c.status === 'sending' || c.status === 'paused') {
    parts.push(t(m.resultsLive))
  }
  return parts.join(' · ')
}

/** "SES 43,182 · Postal 4,798" */
export function sentByLine(h: Schemas['ReportHeadline']): string {
  return h.sentBy.map((p) => `${p.short} ${count(p.count)}`).join(' · ')
}

/** "1.8 s" */
export function seconds(ms: number | null | undefined): string {
  if (ms === null || ms === undefined) return '—'
  return t(m.seconds, { n: decimal(ms / 1000, 1) })
}

export const mailboxColor: Record<string, string> = {
  gmail: 'var(--chart-gmail)',
  outlook: 'var(--chart-outlook)',
  yahoo: 'var(--chart-yahoo)',
  icloud: 'var(--chart-icloud)',
  other: 'var(--chart-other)',
}

/** "side by side · the send was split 90 / 10" */
export function splitCaption(x: Schemas['ReportProviderTable']): string {
  if (!x.split) return x.columns.length === 1 ? t(m.oneProvider) : t(m.whereSent)
  return t(m.splitCaption, { split: x.columns.map((c) => Math.round(c.percent)).join(' / ') })
}

/**
 * The note under the per-provider table: a warming provider's open gap
 * explained ("Postal was on warm-up day 6 and sent to the Active tier only;
 * a 1.8-point open gap on a young IP is normal.").
 */
export function providerNote(x: Schemas['ReportProviderTable']): string | null {
  if (!x.split) {
    const only = x.columns[0]
    return only ? t(m.allThrough, { name: only.name }) : null
  }
  const warm = x.columns.find((c) => c.warmupDay)
  if (!warm) return null
  const lead = x.columns.find((c) => c !== warm && c.sent > 0)
  const engaged = warm.engagedFirst ? 'yes' : 'no'
  // Human opens of measurable recipients: Apple Mail's share doesn't move the gap.
  const opens = (c: Schemas['ReportProviderColumn']) => c.humanOpenRateMeasurable ?? c.humanOpenRate
  const a = lead ? opens(lead) : undefined
  const b = opens(warm)
  if (a === undefined || b === undefined) {
    return t(m.warmupDay, { engaged, provider: warm.short, day: warm.warmupDay ?? 0 })
  }
  const gap = Math.abs(Number(a.toFixed(1)) - Number(b.toFixed(1)))
  return t(m.warmupGapOpens, { engaged, normal: gap <= 3 ? 'yes' : 'no', provider: warm.short, day: warm.warmupDay ?? 0, gap: decimal(gap, 1) })
}

/**
 * "62% of human opens came within 6 hours. The samples went out at 10:00;
 * the steep rise starts at 12:00, when B went to the other 38,384."
 */
export function engagementNote(e: Schemas['ReportEngagement'], final: boolean, startedAt?: string, remaining?: number): string | null {
  if (!e.available || e.humanWithin6h === undefined) return null
  if (e.hours < 6) return t(m.coming)
  const share = pct(Math.round(e.humanWithin6h), 0)
  const done = final ? 'yes' : 'no'
  if (e.rollout && startedAt && remaining) {
    return t(m.within6hRolloutNew, { final: done, share, start: clock(startedAt), time: clock(e.rollout.at), variant: e.rollout.variant, count: count(remaining) })
  }
  return t(m.within6hNew, { final: done, share })
}

/** Metrics whose rates keep two decimals (0.24%, 0.04%, 0.60%). */
const smallRates = new Set(['unsubscribes', 'complaints', 'bounces'])

export interface TileComparison {
  /** ▲ or ▼ (aria-hidden), and "Up" or "Down" for screen readers. */
  arrow: string
  direction: string
  text: string
  tone: 'pos' | 'neg' | 'neu'
}

/** "▲ 2.4% vs your last 6 issues": a tile's change against the median of the last issues. */
export function comparison(c: Schemas['ReportComparisons'], metric: Schemas['ReportComparison']['metric']): TileComparison | undefined {
  const x = c.tiles.find((tile) => tile.metric === metric)
  if (!x || c.issues <= 0) return undefined
  const digits = x.unit === 'percent' ? 1 : smallRates.has(metric) ? 2 : 1
  const size = Math.abs(x.change)
  const tone = x.tone === 'good' ? 'pos' : x.tone === 'bad' ? 'neg' : 'neu'
  if (Number(size.toFixed(digits)) === 0) return { arrow: '=', direction: '', text: t(m.sameAsLast, { n: c.issues }), tone: 'neu' }
  const change = x.unit === 'percent' ? pct(size, digits) : t(m.points, { n: decimal(size, digits) })
  const up = x.change > 0
  return { arrow: up ? '▲' : '▼', direction: t(up ? m.up : m.down), text: t(m.vsLast, { change, n: c.issues }), tone }
}

/**
 * A tile's trend line, read out: "Up over your last 6 issues, from 49.9% to
 * 52.8%" (review AN-11). It runs from the oldest of the earlier issues to
 * this one, and says the same as the numbers the tile shows: the change is
 * up or down when they differ at the tile's precision.
 */
export function trendLabel(x: Schemas['ReportComparison']): string | undefined {
  const v = x.trend
  if (!v || v.length < 2) return undefined
  const fmt = (n: number) => (x.metric === 'sent' ? count(n) : pct(n, smallRates.has(x.metric) ? 2 : 1))
  const first = v[0]!
  const now = v[v.length - 1]!
  const from = fmt(first)
  const to = fmt(now)
  const dir = from === to ? 'same' : now > first ? 'up' : 'down'
  return t(m.trendLine, { dir, n: v.length - 1, from, to })
}

/** "193 hard, 79 soft, 16 blocked or other", after the rate of sent. */
export function bounceSub(h: Schemas['ReportHeadline'], final: boolean): string {
  const rate = pct2(h.bounceRate)
  if (!final) return t(m.bouncesSoFar, { rate })
  const other = Math.max(0, h.bounces - h.hardBounces - h.softBounces)
  const values = { rate, hard: count(h.hardBounces), soft: count(h.softBounces), other: count(other) }
  return other > 0 ? t(m.bouncesSub, values) : t(m.bouncesHardSoft, values)
}

/** "Gmail and iCloud don't report complaints". */
export function notReportedLine(h: Schemas['ReportHeadline']): string | null {
  const by = h.notReportedBy ?? []
  if (!by.length) return null
  return t(m.notReportedBy, { n: by.length, providers: andList(by) })
}

/** A scale's top for the bars and the chart: 10%, 20%, 50%… above the largest value. */
export function niceMax(v: number): { max: number; step: number } {
  for (const step of [1, 2, 5, 10, 20, 25]) {
    if (v <= step * 5) return { max: Math.max(step, Math.ceil(v / step) * step), step }
  }
  return { max: 100, step: 20 }
}

/** "28 people chose “fewer emails” instead of leaving and moved to the monthly frequency." */
export function fewerNote(n: number): string | null {
  if (n <= 0) return null
  return t(m.fewer, { n, count: count(n) })
}

/** "How the winner was picked", as sentences. */
export function abExplanation(x: Schemas['ReportABTest']): string[] {
  const hasStats = x.pValue !== undefined && x.lift !== undefined
  const values = {
    metric: x.metric === 'clicks' ? 'clicks' : 'human_opens',
    hours: x.windowHours,
    stats: hasStats ? 'yes' : 'no',
    lift: hasStats ? `${x.lift! >= 0 ? '+' : ''}${decimal(x.lift!, 1)}` : '',
    p: hasStats ? decimal(x.pValue!, x.pValue! < 0.01 ? 3 : 2) : '',
  }
  // Machine opens are left out of a human-opens test, so Apple Mail can't decide it.
  const machine = (s: string) => (x.metric === 'human_opens' ? `${s} ${t(m.abMachine)}` : s)
  const out: string[] = []
  if (x.state === 'testing') {
    out.push(machine(t(m.abDecides, values)))
    out.push(x.decidedAt ? t(m.abEveryoneAt, { time: clock(x.decidedAt) }) : t(m.abEveryoneEnd))
    return out
  }
  out.push(machine(t(x.significant ? m.abResult : m.abNotSignificant, values)))
  if (x.state === 'decided' && x.remaining > 0) {
    const remaining = { n: x.remaining, count: count(x.remaining), winner: x.winner ?? '' }
    out.push(x.rolloutAt ? t(m.abRemainingAt, { ...remaining, time: clock(x.rolloutAt) }) : t(m.abRemaining, remaining))
  }
  return out
}

/** "4,798 recipients · 10% sample, then 38,384 remaining" */
export function variantLine(v: Schemas['ReportABVariant']): string {
  const values = { count: count(v.recipients), sample: pct(Math.round(v.samplePercent), 0) }
  return v.remaining > 0 ? t(m.variantLineRemaining, { ...values, remaining: count(v.remaining) }) : t(m.variantLine, values)
}

/** "It's scheduled for Tuesday 14 Oct at 09:00." */
export function notSentLead(n: Schemas['ReportNotSent']): string {
  switch (n.status) {
    case 'scheduled':
      if (n.scheduledAt) {
        // The day without its year: "Tuesday 14 Oct", "Thứ Ba, 14 tháng 10".
        const day = longDate(n.scheduledAt).replace(/(,|\s+de)?\s*\d{4}$/, '')
        return t(m.leadScheduledFor, { day, time: clock(n.scheduledAt) })
      }
      return t(m.leadScheduled)
    case 'approval':
      return t(m.leadApproval)
    case 'approved':
      return t(m.leadApproved)
  }
  return t(m.leadDraft)
}
