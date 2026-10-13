import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import { clock, count, longDate, shortDate } from '@/lib/format'

export type CampaignReport = Schemas['CampaignReport']
export type ReportInfo = Schemas['ReportCampaignInfo']

/** "41.2%": rates of delivered or sent keep one decimal. */
export function pct1(v: number | null | undefined): string {
  return v === null || v === undefined ? '—' : `${v.toFixed(1)}%`
}

/** "0.04%": small rates (complaints, unsubscribes) keep two decimals. */
export function pct2(v: number | null | undefined): string {
  return v === null || v === undefined ? '—' : `${v.toFixed(2)}%`
}

/** "Thu 9 Oct, 10:00 to 12:41", or with both dates when it ran past midnight. */
export function sendWindow(start?: string, end?: string): string {
  if (!start) return ''
  const s = `${shortDate(start)}, ${clock(start)}`
  if (!end) return s
  return shortDate(start) === shortDate(end) ? `${s} to ${clock(end)}` : `${s} to ${shortDate(end)}, ${clock(end)}`
}

/** The status badge of the report header. */
export function statusBadge(c: ReportInfo): { tone: Tone; text: string } {
  switch (c.status) {
    case 'sent':
      return { tone: 'ok', text: c.startedAt ? `Sent · ${sendWindow(c.startedAt, c.completedAt)}` : 'Sent' }
    case 'sending':
      return { tone: 'info', text: c.startedAt ? `Sending · started ${clock(c.startedAt)}` : 'Sending' }
    case 'paused':
      return { tone: 'warn', text: 'Paused' }
    case 'cancelled':
      return { tone: 'neutral', text: c.startedAt ? `Cancelled · ${sendWindow(c.startedAt, c.completedAt)}` : 'Cancelled' }
    case 'failed':
      return { tone: 'bad', text: 'Failed' }
    case 'scheduled':
      return { tone: 'info', text: c.scheduledAt ? `Scheduled · ${shortDate(c.scheduledAt)}, ${clock(c.scheduledAt)}` : 'Scheduled' }
    case 'approval':
      return { tone: 'warn', text: 'Waiting for approval' }
    case 'approved':
      return { tone: 'info', text: 'Approved' }
  }
  return { tone: 'neutral', text: 'Draft' }
}

/** "All subscribers · A/B subject, winner B · results final after 72 h". */
export function headerCaption(c: ReportInfo): string {
  const parts: string[] = []
  if (c.audience) parts.push(c.audience)
  parts.push(...c.notes)
  if (c.origin) {
    parts.push(`imported from ${c.origin[0]!.toUpperCase()}${c.origin.slice(1)}`)
  } else if (c.status === 'sent' || c.status === 'cancelled') {
    if (c.final) parts.push('results final after 72 h')
    else if (c.finalAt) parts.push(`results still coming in · final ${shortDate(c.finalAt)}, ${clock(c.finalAt)}`)
  } else if (c.status === 'sending' || c.status === 'paused') {
    parts.push('results coming in live')
  }
  return parts.join(' · ')
}

/** "SES 43,182 · Postal 4,798" */
export function sentByLine(h: Schemas['ReportHeadline']): string {
  return h.sentBy.map((p) => `${p.short} ${count(p.count)}`).join(' · ')
}

/** "0.60% · 144 expired deferrals" */
export function bounceLine(h: Schemas['ReportHeadline'], final: boolean): string {
  const rate = pct2(h.bounceRate)
  if (!h.undelivered) return rate
  return `${rate} · ${count(h.undelivered)} ${final ? (h.undelivered === 1 ? 'expired deferral' : 'expired deferrals') : 'not delivered yet'}`
}

/** "1.8 s" */
export function seconds(ms: number | null | undefined): string {
  if (ms === null || ms === undefined) return '—'
  return `${(ms / 1000).toFixed(1)} s`
}

export const mailboxColor: Record<string, string> = {
  gmail: 'var(--chart-gmail)',
  outlook: 'var(--chart-outlook)',
  yahoo: 'var(--chart-yahoo)',
  icloud: 'var(--chart-icloud)',
  other: 'var(--chart-other)',
}

/** "side by side · the send was split 90 / 10" */
export function splitCaption(t: Schemas['ReportProviderTable']): string {
  if (!t.split) return t.columns.length === 1 ? 'one sending provider' : 'where it was sent from'
  return `side by side · the send was split ${t.columns.map((c) => Math.round(c.percent)).join(' / ')}`
}

/**
 * The note under the per-provider table: a warming provider's open gap
 * explained ("Postal was on warm-up day 6 and sent to the Active tier only;
 * a 1.8-point open gap on a young IP is normal.").
 */
export function providerNote(t: Schemas['ReportProviderTable']): string | null {
  if (!t.split) {
    const only = t.columns[0]
    return only ? `All of it went through ${only.name}.` : null
  }
  const warm = t.columns.find((c) => c.warmupDay)
  if (!warm) return null
  const lead = t.columns.find((c) => c !== warm && c.sent > 0)
  const engaged = warm.engagedFirst ? ' and sent to the Active tier only' : ''
  if (!lead || lead.humanOpenRate === undefined || warm.humanOpenRate === undefined) {
    return `${warm.short} was on warm-up day ${warm.warmupDay}${engaged}.`
  }
  const gap = Math.abs(Number(lead.humanOpenRate.toFixed(1)) - Number(warm.humanOpenRate.toFixed(1)))
  const verdict = gap <= 3 ? 'on a young IP is normal' : 'is more than a young IP explains; check its reputation in Deliverability'
  return `${warm.short} was on warm-up day ${warm.warmupDay}${engaged}; a ${gap.toFixed(1)}-point open gap ${verdict}.`
}

/** "62% of human opens happened within 6 hours; A/B winner B went out at 12:00, which is the second rise." */
export function engagementNote(e: Schemas['ReportEngagement'], final: boolean): string | null {
  if (!e.available || e.humanWithin6h === undefined) return null
  if (e.hours < 6) return 'Opens and clicks are coming in; the curve fills in over 72 hours.'
  let s = `${Math.round(e.humanWithin6h)}% of human opens happened within 6 hours`
  if (!final) s = `So far, ${s}`
  if (e.rollout) return `${s}; A/B winner ${e.rollout.variant} went out at ${clock(e.rollout.at)}, which is the second rise.`
  return `${s}.`
}

function lowerFirst(s: string) {
  return s ? s[0]!.toLowerCase() + s.slice(1) : s
}

/** "Links with under 5% of clicks are grouped: “Other” 13% covers 9 links including email preferences and view in browser." */
export function clickNote(m: Schemas['ReportClickMap']): string | null {
  if (!m.other) return null
  const labels = m.other.labels.map(lowerFirst)
  const incl = labels.length ? ` including ${labels.join(' and ')}` : ''
  return `Links with under ${m.threshold}% of clicks are grouped: “Other” ${Math.round(m.other.share)}% covers ${m.other.links} ${m.other.links === 1 ? 'link' : 'links'}${incl}.`
}

/** "28 people chose “fewer emails” instead of leaving and moved to the monthly frequency." */
export function fewerNote(n: number): string | null {
  if (n <= 0) return null
  return `${count(n)} ${n === 1 ? 'person' : 'people'} chose “fewer emails” instead of leaving and moved to the monthly frequency.`
}

const metricName: Record<string, string> = { human_opens: 'Human opens', clicks: 'Clicks' }

/** The A/B badge: "Winner B · 98% confidence", "Testing · decides at 12:00", "Not significant yet · 81% confidence". */
export function abBadge(t: Schemas['ReportABTest']): { tone: Tone; text: string } {
  const conf = t.confidence !== undefined ? `${Math.min(99, Math.round(t.confidence))}% confidence` : ''
  if (t.state === 'testing') return { tone: 'info', text: t.decidedAt ? `Testing · decides at ${clock(t.decidedAt)}` : 'Testing' }
  if (t.state === 'decided' && t.significant) return { tone: 'ok', text: [`Winner ${t.winner}`, conf].filter(Boolean).join(' · ') }
  if (t.state === 'decided') return { tone: 'warn', text: [`${t.winner} chosen · not significant`, conf].filter(Boolean).join(' · ') }
  return { tone: 'warn', text: ['Not significant yet', conf].filter(Boolean).join(' · ') }
}

/** "How the winner was picked", as sentences. */
export function abExplanation(t: Schemas['ReportABTest']): string[] {
  const metric = metricName[t.metric] ?? 'Human opens'
  const window = t.windowHours === 1 ? '1 hour' : `${t.windowHours} hours`
  const stats = t.pValue !== undefined && t.lift !== undefined ? `, two-proportion z-test, lift ${t.lift >= 0 ? '+' : ''}${t.lift.toFixed(1)} points, p = ${t.pValue < 0.01 ? t.pValue.toFixed(3) : t.pValue.toFixed(2)}` : ''
  const machine = t.metric === 'human_opens' ? ' Machine opens were excluded so Apple Mail prefetches couldn’t decide it.' : ''
  const out: string[] = []
  if (t.state === 'testing') {
    out.push(`${metric} ${window} after the sample went out decide it${stats ? `; so far${stats.slice(1)}` : ''}.${machine}`)
    out.push(t.decidedAt ? `Everyone else gets the winner at ${clock(t.decidedAt)}, unless the difference isn't significant.` : 'Everyone else gets the winner when the test ends.')
    return out
  }
  if (!t.significant) {
    out.push(`${metric} ${window} after the sample went out${stats}: not significant, so the difference could be chance.${machine}`)
  } else {
    out.push(`${metric} ${window} after the sample went out${stats}.${machine}`)
  }
  if (t.state === 'decided' && t.remaining > 0) {
    out.push(`The remaining ${count(t.remaining)} recipients got ${t.winner}${t.rolloutAt ? ` at ${clock(t.rolloutAt)}` : ''}.`)
  }
  return out
}

/** "4,798 recipients · 10% sample, then 38,384 remaining" */
export function variantLine(v: Schemas['ReportABVariant']): string {
  const base = `${count(v.recipients)} recipients · ${Math.round(v.samplePercent)}% sample`
  return v.remaining > 0 ? `${base}, then ${count(v.remaining)} remaining` : base
}

/** "It's scheduled for Tuesday 14 Oct at 09:00." */
export function notSentLead(n: Schemas['ReportNotSent']): string {
  switch (n.status) {
    case 'scheduled':
      if (n.scheduledAt) {
        const day = longDate(n.scheduledAt).replace(/ \d{4}$/, '')
        return `It's scheduled for ${day} at ${clock(n.scheduledAt)}.`
      }
      return "It's scheduled."
    case 'approval':
      return "It's waiting for approval."
    case 'approved':
      return "It's approved and waiting to be scheduled."
  }
  return "It's a draft."
}
