import type { Schemas } from '@/api/client'
import { ago, andList, clock, clockSeconds, count, duration, money, monthDay, rate, shortDate, trimPct } from '@/lib/format'

/**
 * The live sending monitor's words and numbers (spec 7.6; boards
 * LiveMonitor, LiveMonitorPaused, LiveMonitorOutage, LiveMonitorStates,
 * LiveMonitorMobile), computed from the engine's snapshot. Pure functions,
 * pinned by model.test.ts against the boards' copy.
 */

export type Snapshot = Schemas['LiveSnapshot']
export type Mailbox = Schemas['LiveMailbox']
export type Provider = Schemas['LiveProvider']
export type Incident = Schemas['LiveIncident']
export type LiveEvent = Schemas['LiveEvent']
export type Threshold = Schemas['LiveThreshold']
export type IncidentOption = Schemas['IncidentOption']
export type MailboxKey = Mailbox['key']
export type Tone = 'ok' | 'warn' | 'bad' | 'info' | 'neutral'

/** Mailbox-provider series colours (design system chart tokens). */
export const mailboxColor: Record<MailboxKey, string> = {
  gmail: 'var(--chart-gmail)',
  outlook: 'var(--chart-outlook)',
  yahoo: 'var(--chart-yahoo)',
  icloud: 'var(--chart-icloud)',
  other: 'var(--chart-other)',
}

export const mailboxNames: Record<MailboxKey, string> = { gmail: 'Gmail', outlook: 'Outlook', yahoo: 'Yahoo', icloud: 'iCloud', other: 'Other' }

export function mailboxName(key: string | undefined): string {
  return (key && mailboxNames[key as MailboxKey]) || key || ''
}

/** Sending-provider series colours. */
export const providerColor: Record<Provider['color'], string> = {
  ses: 'var(--chart-ses)',
  postal: 'var(--chart-postal)',
  postmark: 'var(--chart-postmark)',
  other: 'var(--chart-other)',
}

/** The progress bar's other segments and legend dots (boards). */
export const queuedColor = 'var(--line-hover)'
export const heldMailboxColor = 'var(--bad-line)'
export const heldRouteColor = '#FCD34D'
export const waitingColor = 'var(--purple-fg)'

/** Threshold bars: green under half the limit, amber from half, red at the limit. */
export function usageColor(value: number, limit: number): string {
  if (limit <= 0) return 'var(--ok-bar)'
  const r = value / limit
  if (r >= 1) return 'var(--bad-strong)'
  if (r >= 0.5) return 'var(--warn-bar)'
  return 'var(--ok-bar)'
}

export function usageWidth(value: number, limit: number): number {
  if (limit <= 0) return 0
  return Math.max(0, Math.min(100, (value / limit) * 100))
}

// --- Numbers in the boards' style ------------------------------------------

/** One decimal: 99.3%. */
export function pct1(v: number | null | undefined): string {
  return v === null || v === undefined || Number.isNaN(v) ? '—' : `${v.toFixed(1)}%`
}

/** Two decimals, for complaint rates: 0.02%. */
export function pct2(v: number | null | undefined): string {
  return v === null || v === undefined || Number.isNaN(v) ? '—' : `${v.toFixed(2)}%`
}

/** n of d as a percentage, or undefined without a base. */
export function share(n: number, d: number): number | undefined {
  return d > 0 ? (n / d) * 100 : undefined
}

/** A live rate: "14.2 /s", "2.0 /s", "0 /s". */
export function perSec(v: number): string {
  return v <= 0 ? '0 /s' : `${v.toFixed(1)} /s`
}

/** A throttle or a rounded rate: "5 /s", "14 /s". */
export function wholePerSec(v: number): string {
  return `${Number.isInteger(v) ? v : Math.round(v * 10) / 10} /s`
}

/** "62%": whole percent, never 100% before everything is sent. */
export function progressText(s: Pick<Snapshot, 'sent' | 'recipients' | 'percent'>): string {
  if (s.recipients <= 0) return '0%'
  if (s.sent >= s.recipients) return '100%'
  return `${Math.min(99, Math.round((s.sent / s.recipients) * 100))}%`
}

/** "1 min 26 s", "45 s", "3 min". */
export function secondsText(total: number): string {
  const s = Math.max(0, Math.round(total))
  if (s < 60) return `${s} s`
  const m = Math.floor(s / 60)
  const r = s % 60
  if (m >= 60) return duration(m)
  return r ? `${m} min ${r} s` : `${m} min`
}

/** "$0.12", and "$0" for nothing (self-hosted providers). */
export function cost(v: number): string {
  return v === 0 ? '$0' : money(v)
}

/** "1 s ago", "40 s ago", "3 min ago" for a live feed. */
export function liveAgo(at: string | undefined, now: Date): string {
  if (!at) return 'no events yet'
  const s = Math.round((now.getTime() - new Date(at).getTime()) / 1000)
  if (s < 60) return `${Math.max(1, s)} s ago`
  if (s < 3600) return `${Math.round(s / 60)} min ago`
  return ago(at, now)
}

/** "Oct 10 16:20", as the header writes approvals. */
export function shortStamp(d: string): string {
  return `${monthDay(d)} ${clock(d)}`
}

// --- Header ------------------------------------------------------------------

/** "Started by Sofia Rossi · approved by Marcus Chen, Oct 10 16:20". */
export function startedLine(s: Snapshot): string {
  const parts: string[] = []
  if (s.startedBy) parts.push(`Started by ${s.startedBy}`)
  else if (s.state === 'scheduled' && s.scheduledBy) parts.push(`Scheduled by ${s.scheduledBy}`)
  if (s.approvedBy) parts.push(`approved by ${s.approvedBy}${s.approvedAt ? `, ${shortStamp(s.approvedAt)}` : ''}`)
  const line = parts.join(' · ')
  return line ? line[0]!.toUpperCase() + line.slice(1) : ''
}

/** Who can help with a permission: "Ask Marcus Chen or Linh Nguyen." */
export function needs(action: string, perm: string, helpers: string[] | undefined): string {
  const ask = helpers?.length ? ` Ask ${helpers.length === 1 ? helpers[0] : `${helpers.slice(0, -1).join(', ')} or ${helpers[helpers.length - 1]}`}.` : ''
  return `${action} needs the ${perm} permission.${ask}`
}

// --- Progress card -----------------------------------------------------------

export type Segment = { key: string; value: number; color: string; title: string }

/** The providers' sent shares, then what's held (boards: red for a mailbox provider, amber for a route). */
export function progressSegments(s: Snapshot, incident?: Incident): Segment[] {
  const segs: Segment[] = s.providers.filter((p) => p.sent > 0).map((p) => ({ key: p.id, value: p.sent, color: providerColor[p.color], title: p.short }))
  if (s.held > 0) {
    const route = incident?.kind === 'provider_outage'
    segs.push({ key: 'held', value: s.held, color: route ? heldRouteColor : heldMailboxColor, title: 'Held' })
  }
  return segs
}

/** "19,594 of 31,604 sent", with what's held in the decision views. */
export function sentLine(s: Snapshot, incident?: Incident): string {
  const base = `${count(s.sent)} of ${count(s.recipients)} sent`
  if (!incident || s.held <= 0) return base
  if (incident.kind === 'provider_outage') return `${base} · ${count(s.held)} held on ${incident.providerShort}`
  return `${base} · ${count(s.held)} held`
}

/** "Started 09:00:04 · ETA 13 min · finishes about 09:34". */
export function timingLine(s: Snapshot, incident?: Incident): string {
  const parts: string[] = []
  if (s.startedAt) parts.push(`Started ${clockSeconds(s.startedAt)}`)
  const eta = s.etaSeconds !== undefined ? duration(s.etaSeconds / 60) : ''
  if (incident?.kind === 'threshold' && incident.scope !== 'campaign') {
    parts.push(`paused route since ${clockSeconds(incident.trippedAt)}`)
    if (eta) parts.push(`ETA for the rest ${eta}`)
  } else if (incident?.kind === 'provider_outage') {
    const others = continuingProviders(s, incident)
    if (s.finishesAt && others.length) parts.push(`${andList(others.map((p) => p.short))} finishes about ${clock(s.finishesAt)}`)
  } else if (s.state === 'sending' || s.state === 'partial') {
    if (eta) parts.push(`ETA ${eta}`)
    if (s.finishesAt) parts.push(`finishes about ${clock(s.finishesAt)}`)
  } else if (s.state === 'paused' && s.pausedAt) {
    parts.push(`paused since ${clockSeconds(s.pausedAt)}`)
  }
  return parts.join(' · ')
}

/** "A/B subject test: variant B leading, 15.1% vs 13.2% opens on the 20% sample". */
export function abLine(ab: Snapshot['ab']): string {
  if (!ab || ab.variants.length < 2) return ''
  const kind = ab.kind === 'content' ? 'content' : ab.kind === 'sender' || ab.kind === 'from' ? 'sender' : 'subject'
  const metric = ab.metric === 'clicks' ? 'clicks' : 'opens'
  const lead = ab.phase === 'decided' && ab.winner ? ab.winner : ab.leader
  const first = ab.variants.find((v) => v.key === lead) ?? ab.variants[0]!
  const rest = ab.variants.filter((v) => v.key !== first.key)
  const rates = [first, ...rest].map((v) => `${v.rate.toFixed(1)}%`).join(' vs ')
  const verb = ab.phase === 'decided' ? 'won' : 'leading'
  return `A/B ${kind} test: variant ${first.key} ${verb}, ${rates} ${metric} on the ${ab.sample}% sample`
}

// --- Totals tiles ------------------------------------------------------------

export type TilesVariant = 'live' | 'threshold' | 'outage'
export type Tile = { key: string; label: string; value: string; sub: string; alert?: boolean }

export function tiles(s: Snapshot, variant: TilesVariant = 'live', incident?: Incident): Tile[] {
  const t = s.totals
  const delivered = { key: 'delivered', label: 'Delivered', value: count(t.delivered), sub: `${pct1(share(t.delivered, s.sent))} of sent` }
  const bounceRate = rate(share(t.bounced, s.sent) ?? 0)
  const bounced = {
    key: 'bounced',
    label: 'Bounced',
    value: count(t.bounced),
    sub: variant === 'outage' ? bounceRate : `${bounceRate} · ${count(t.hardBounced)} hard, ${count(t.softBounced)} soft`,
  }
  const complaintRate = pct2(share(t.complained, t.delivered) ?? 0)
  let complained: Tile = { key: 'complained', label: 'Complained', value: count(t.complained), sub: variant === 'live' ? `${complaintRate} of delivered` : complaintRate }
  if (variant === 'threshold' && incident?.metric === 'complaint' && incident.scope === 'mailbox_provider') {
    const mb = s.mailboxes.find((m) => m.key === incident.mailbox)
    complained = { ...complained, sub: `${complaintRate} overall · ${incident.mailboxName} ${pct2(mb?.complaintRate ?? incident.value)}`, alert: true }
  }
  const opened = {
    key: 'opened',
    label: 'Opened',
    value: count(t.opened),
    sub: `${pct1(share(t.opened, t.delivered) ?? 0)} · ${count(t.openedHuman)} human, ${count(t.openedMachine)} machine`,
  }
  const clickRate = pct1(share(t.clicked, t.delivered) ?? 0)
  const clicked = {
    key: 'clicked',
    label: 'Clicked',
    value: count(t.clicked),
    sub: variant === 'live' ? `${clickRate} · ${pct1(share(t.clicked, t.opened) ?? 0)} click-to-open` : clickRate,
  }
  return [delivered, bounced, complained, opened, clicked]
}

/** The complete state's tiles (LiveMonitorStates: send complete). */
export function finalTiles(s: Snapshot): Tile[] {
  const t = s.totals
  const providers = s.providers.filter((p) => p.sent > 0)
  return [
    { key: 'delivered', label: 'Delivered', value: count(t.delivered), sub: pct1(share(t.delivered, s.sent)) },
    { key: 'bounced', label: 'Bounced', value: count(t.bounced), sub: `${rate(share(t.bounced, s.sent) ?? 0)}${t.hardBounced > 0 ? ' · added to suppression' : ''}` },
    { key: 'complained', label: 'Complained', value: count(t.complained), sub: pct2(share(t.complained, t.delivered) ?? 0) },
    { key: 'cost', label: 'Cost', value: cost(s.cost.total), sub: providers.map((p) => `${p.short} ${cost(p.cost)}`).join(' · ') },
  ]
}

/** "SES 28,444 · 99.2%" lines of the complete state's last tile. */
export function providerResultLines(s: Snapshot): string[] {
  return s.providers.filter((p) => p.sent > 0).map((p) => `${p.short} ${count(p.sent)} · ${pct1(p.deliveredRate)}`)
}

// --- Thresholds --------------------------------------------------------------

export type ThresholdRow = { key: string; label: string; short: string; value: string; width: number; color: string }

export function thresholdRows(s: Snapshot): ThresholdRow[] {
  return s.thresholds.map((t) => {
    const mb = mailboxName(t.mailbox)
    switch (t.key) {
      case 'bounce':
        return row(t, 'Bounce rate', 'Bounce', `${t.value.toFixed(2)}% of ${t.limit.toFixed(1)}%`)
      case 'complaint':
        return row(t, 'Complaint rate', 'Complaint', `${t.value.toFixed(2)}% of ${t.limit.toFixed(2)}%`)
      case 'mailbox_complaint':
        return row(t, `${mb} complaints (highest mailbox provider)`, `${mb} complaint`, `${t.value.toFixed(2)}% of ${t.limit.toFixed(2)}%`)
      default:
        return row(t, 'Provider API errors, 3 min', 'Provider errors', `${errorRate(t.value)} of ${trimPct(t.limit)}`)
    }
  })
}

function row(t: Threshold, label: string, short: string, value: string): ThresholdRow {
  return { key: t.key + (t.mailbox ?? ''), label, short, value, width: usageWidth(t.value, t.limit), color: usageColor(t.value, t.limit) }
}

export function scopeLine(s: Snapshot): string {
  return s.thresholdScope === 'campaign' ? 'Scope: pause the whole campaign.' : 'Scope: pause only the affected route or mailbox provider.'
}

// --- Rows --------------------------------------------------------------------

/** A mailbox provider's state word: the live view shows health, the decision view activity. */
export function mailboxStatus(m: Mailbox, s: Snapshot, decision: boolean): { tone: Tone; text: string } {
  if (m.held) return { tone: 'bad', text: m.heldSince ? `Paused ${clock(m.heldSince)}` : 'Paused' }
  if (decision) {
    if (m.remaining > 0) return { tone: 'ok', text: 'Sending' }
    return { tone: 'neutral', text: 'Done' }
  }
  const limit = s.thresholds.find((t) => t.key === 'mailbox_complaint')?.limit ?? 0.1
  // Over the limit without a pause yet: the engine pauses once there are
  // enough complaints to be sure it isn't chance.
  if (m.complaintRate !== undefined && m.complaintRate > limit && m.delivered >= 500) return { tone: 'warn', text: 'Over limit' }
  if (m.complaintRate !== undefined && m.complaintRate >= limit * 0.8 && m.delivered >= 500) return { tone: 'warn', text: 'Near limit' }
  return { tone: 'ok', text: 'OK' }
}

/** "full", "5 /s" or "held". */
export function mailboxRate(m: Mailbox): string {
  if (m.held) return 'held'
  return m.throttle ? wholePerSec(m.throttle) : 'full'
}

/** A sending provider's state word. */
export function providerStatus(p: Provider, mobile = false): { tone: Tone; text: string } {
  if (p.downSince && (p.routeStatus === 'held' || p.status === 'down')) return { tone: 'bad', text: `Down since ${clock(p.downSince)}` }
  if (p.routeStatus === 'held' || p.routeStatus === 'paused') return { tone: 'warn', text: 'Held' }
  if (p.status === 'down') return { tone: 'bad', text: 'Down' }
  if (p.status === 'degraded') return { tone: 'warn', text: 'Degraded' }
  if (p.warmupDay !== undefined) return { tone: 'info', text: mobile ? 'Warm-up' : 'Warming up' }
  return { tone: 'ok', text: 'Healthy' }
}

/** The line under a provider's name: quota, warm-up, or where it runs when it's down. */
export function providerNote(p: Provider, decision: boolean): string {
  if (decision && (p.routeStatus === 'held' || p.downSince)) return [p.host, p.ip].filter(Boolean).join(' · ')
  if (decision) return ''
  if (p.warmupDay !== undefined && p.warmupCap !== undefined)
    return `warm-up day ${p.warmupDay} · ${count(p.warmupSent ?? 0)} of today's ${count(p.warmupCap)} cap`
  if (p.quotaFree !== undefined) return `quota ${count(p.quotaFree)} free after this send`
  return ''
}

/** Providers still sending while another one's share is held. */
export function continuingProviders(s: Snapshot, incident?: Incident): Provider[] {
  return s.providers.filter((p) => p.id !== incident?.providerId && (p.routeStatus === 'sending' || p.routeStatus === 'pending') && p.remaining > 0)
}

/** The decision view orders the held mailbox provider first. */
export function orderedMailboxes(s: Snapshot, decision: boolean): Mailbox[] {
  if (!decision) return s.mailboxes
  return [...s.mailboxes].sort((a, b) => Number(b.held) - Number(a.held))
}

// --- Incidents ---------------------------------------------------------------

const metricWords: Record<string, { one: string; many: string; base: string }> = {
  complaint: { one: 'complaint', many: 'complaints', base: 'delivered' },
  bounce: { one: 'bounce', many: 'bounces', base: 'sent' },
  deferral: { one: 'deferral', many: 'deferrals', base: 'sent' },
}

function words(metric: string) {
  return metricWords[metric] ?? { one: metric, many: metric, base: 'sent' }
}

/** A rate the way thresholds show it: complaints with two decimals, others with one below 1%. */
export function rateText(metric: string, v: number | undefined): string {
  if (v === undefined) return '—'
  return metric === 'complaint' ? pct2(v) : rate(v)
}

/** A limit: complaints "0.10%", bounces "2.0%". */
export function limitText(metric: string, v: number | undefined): string {
  if (v === undefined) return '—'
  return metric === 'complaint' ? pct2(v) : pct1(v)
}

/** Whose share tripped: "Outlook", "Postal" or "" for the whole campaign. */
export function scopeName(i: Incident): string {
  if (i.scope === 'mailbox_provider') return i.mailboxName || mailboxName(i.mailbox)
  if (i.scope === 'route') return i.providerShort || i.provider
  return ''
}

/** "Outlook complaint rate reached 0.18% (limit 0.10%)". */
export function incidentTitle(i: Incident): string {
  if (i.kind === 'provider_outage') return outageTitle(i)
  const w = words(i.metric)
  const who = scopeName(i)
  const subject = who ? `${who} ${w.one} rate` : `${w.one[0]!.toUpperCase()}${w.one.slice(1)} rate`
  return `${subject} reached ${rateText(i.metric, i.value)} (limit ${limitText(i.metric, i.limit)})`
}

/** "Postal returned connection errors for 3 minutes". */
export function outageTitle(i: Incident): string {
  const d = i.detail
  const name = i.provider || i.providerShort
  if (d.reason === 'auth') return `${name} rejected the credentials`
  if (!d.from || !d.to) return d.errorKind ? `${name} returned ${d.errorKind}` : `${name} is down`
  const mins = Math.max(1, Math.round((new Date(d.to).getTime() - new Date(d.from).getTime()) / 60000))
  return `${name} returned ${d.errorKind || 'errors'} for ${mins} ${mins === 1 ? 'minute' : 'minutes'}`
}

/**
 * "Tripped at 09:21:14, after 4,115 of 6,637 Outlook recipients were sent.
 * 7 complaints out of 3,980 delivered. ReputeCast held the remaining 2,522
 * Outlook recipients. Gmail, Yahoo, iCloud and Other continue, now 71%
 * complete overall."
 */
export function thresholdSummary(i: Incident, s: Snapshot): string {
  const d = i.detail
  const w = words(i.metric)
  const who = scopeName(i)
  const recipients = who ? `${who} recipients` : 'recipients'
  const events = d.events ?? 0
  const parts = [
    `Tripped at ${clockSeconds(i.trippedAt)}, after ${count(d.sent ?? 0)} of ${count(d.total ?? 0)} ${recipients} were sent.`,
    `${count(events)} ${events === 1 ? w.one : w.many} out of ${count(d.base ?? 0)} ${w.base}.`,
  ]
  const pctText = progressText(s)
  if (i.scope === 'campaign') {
    parts.push(`ReputeCast paused the campaign with ${count(i.held)} recipients left, ${pctText} complete.`)
    return parts.join(' ')
  }
  parts.push(`ReputeCast held the remaining ${count(i.held)} ${recipients}.`)
  const continuing =
    i.scope === 'route' ? continuingProviders(s, i).map((p) => p.name) : s.continuing.length ? s.continuing : (d.continuing ?? [])
  if (continuing.length) parts.push(`${andList(continuing)} ${continuing.length === 1 ? 'continues' : 'continue'}, now ${pctText} complete overall.`)
  else parts.push(`Now ${pctText} complete overall.`)
  return parts.join(' ')
}

/**
 * "From 09:14:02 to 09:17:10, 37 send attempts to mta.acme.io failed and 0
 * succeeded, so the Postal share of this campaign is on hold: 1,956 sent,
 * 1,204 remaining. Amazon SES keeps sending at 14 /s, 73% complete overall.
 * Nothing moved automatically because Broadcast failover is set to pause and
 * ask."
 */
export function outageSummary(i: Incident, s: Snapshot): string {
  const d = i.detail
  const p = s.providers.find((x) => x.id === i.providerId)
  const short = i.providerShort || i.provider
  const sent = p?.sent ?? d.sentOnRoute ?? 0
  const remaining = p?.remaining ?? d.remaining ?? i.held
  const hold = `so the ${short} share of this campaign is on hold: ${count(sent)} sent, ${count(remaining)} remaining.`
  const parts: string[] = []
  if (d.reason === 'auth') {
    parts.push(`${d.from ? `Since ${clockSeconds(d.from)}, ` : ''}${i.provider} refused the credentials on every send, ${hold}`)
  } else if (d.from && d.to) {
    const attempts = d.attempts ?? 0
    parts.push(
      `From ${clockSeconds(d.from)} to ${clockSeconds(d.to)}, ${count(attempts)} send ${attempts === 1 ? 'attempt' : 'attempts'} to ${d.host || i.provider} failed and ${count(d.succeeded ?? 0)} succeeded, ${hold}`,
    )
  } else {
    parts.push(`ReputeCast's health checks found ${i.provider} down at ${clockSeconds(i.trippedAt)}, ${hold}`)
  }
  const others = continuingProviders(s, i)
  const pctText = progressText(s)
  if (others.length) {
    const r = Math.round(others.reduce((a, x) => a + x.rate, 0))
    parts.push(`${andList(others.map((x) => x.name))} ${others.length === 1 ? 'keeps' : 'keep'} sending at ${r} /s, ${pctText} complete overall.`)
  } else {
    parts.push(`No other provider is sending, ${pctText} complete overall.`)
  }
  if (d.policy === 'pause_and_ask') parts.push('Nothing moved automatically because Broadcast failover is set to pause and ask.')
  else if (d.policy === 'warm_only') parts.push('Nothing moved automatically because no warmed-up provider can take the share.')
  else parts.push('Nothing moved automatically because no other provider can take the share.')
  return parts.join(' ')
}

export type Fact = { label: string; value: string; bad?: boolean }

/** "What tripped": the scope's rate, the limit, the campaign-wide rate and the other metric. */
export function trippedFacts(i: Incident, s: Snapshot): { head: Fact; bar: { width: number }; rows: Fact[] } {
  const d = i.detail
  const w = words(i.metric)
  const who = scopeName(i)
  const t = s.totals
  const head: Fact = {
    label: who ? `${who} ${w.many}` : `${w.many[0]!.toUpperCase()}${w.many.slice(1)}`,
    value: `${count(d.events ?? 0)} of ${count(d.base ?? 0)} · ${rateText(i.metric, i.value)}`,
    bad: true,
  }
  const rows: Fact[] = [
    { label: `Limit, ${i.scope === 'mailbox_provider' ? 'per mailbox provider' : i.scope === 'route' ? 'per sending provider' : 'whole campaign'}`, value: limitText(i.metric, i.limit) },
  ]
  if (i.scope !== 'campaign') {
    const [n, base] = i.metric === 'complaint' ? [t.complained, t.delivered] : i.metric === 'bounce' ? [t.bounced, s.sent] : [t.deferred, s.sent]
    rows.push({ label: `Campaign-wide ${w.many}`, value: `${count(n)} of ${count(base)} · ${rateText(i.metric, share(n, base) ?? 0)}` })
  }
  // The other metric, now, for the same scope.
  const mb = i.scope === 'mailbox_provider' ? s.mailboxes.find((m) => m.key === i.mailbox) : undefined
  const pv = i.scope === 'route' ? s.providers.find((p) => p.id === i.providerId) : undefined
  const bounceLimit = s.thresholds.find((x) => x.key === 'bounce')?.limit ?? 2
  const complaintLimit = s.thresholds.find((x) => x.key === (i.scope === 'mailbox_provider' ? 'mailbox_complaint' : 'complaint'))?.limit ?? 0.1
  if (i.metric === 'bounce') {
    const v = mb?.complaintRate ?? pv?.complaintRate ?? share(t.complained, t.delivered)
    rows.push({ label: who ? `${who} complaints` : 'Complaints', value: `${pct2(v ?? 0)} of ${pct2(complaintLimit)}` })
  } else {
    const v = mb?.bounceRate ?? pv?.bounceRate ?? share(t.bounced, s.sent)
    rows.push({ label: who ? `${who} bounces` : 'Bounces', value: `${pct1(v ?? 0)} of ${pct1(bounceLimit)}` })
  }
  return { head, bar: { width: usageWidth(i.value ?? 0, i.limit ?? 0) || (i.value ? 100 : 0) }, rows }
}

/** "09:14:02 ECONNREFUSED mta.acme.io:443" lines of an outage. */
export function logLines(i: Incident): { key: string; text: string; bad: boolean }[] {
  return (i.detail.log ?? []).map((l, n) => ({ key: `${n}`, text: `${clockSeconds(l.at)} ${l.text}`, bad: l.bad }))
}

/** "If you move the share to SES": cost, time, quota and reputation risk. */
export function moveFacts(i: Incident): { title: string; rows: Fact[]; risk?: { tone: Tone; text: string }; note?: string } | null {
  const m = i.detail.move
  if (!m) return null
  const rows: Fact[] = [
    { label: 'Extra cost', value: money(m.extraCost) },
    { label: m.rate ? `Extra time at ${wholePerSec(Math.round(m.rate))}` : 'Extra time', value: secondsText(m.extraSeconds) },
  ]
  if (m.quotaFree !== undefined && m.quotaMax !== undefined) rows.push({ label: `${m.targetShort} quota after`, value: `${count(m.quotaFree)} free of ${count(m.quotaMax)}` })
  return {
    title: `If you move the share to ${m.targetShort}`,
    rows,
    risk: { tone: m.risk === 'low' ? 'ok' : 'warn', text: m.riskText },
    note: i.detail.warmupNote || undefined,
  }
}

/** The option the panel preselects: the recommended one, else the first. */
export function defaultOption(i: Incident): string {
  const opts = i.detail.options ?? []
  return (opts.find((o) => o.recommended) ?? opts[0])?.key ?? ''
}

export type LiveAction =
  | { kind: 'resume'; body: { mailbox?: MailboxKey; providerId?: string; withChanges?: boolean } }
  | { kind: 'cancel'; body: { mailbox?: MailboxKey; providerId?: string }; whole?: boolean }
  | { kind: 'move'; body: { fromProviderId: string; toProviderId: string } }
  | { kind: 'retry'; body: { providerId: string } }

/** What an incident option does on the API. */
export function optionAction(i: Incident, key: string): LiveAction | null {
  const scope = i.scope === 'mailbox_provider' ? { mailbox: i.mailbox as MailboxKey } : i.scope === 'route' ? { providerId: i.providerId } : {}
  switch (key) {
    case 'resume':
      return { kind: 'resume', body: scope }
    case 'resume_with_changes':
      return { kind: 'resume', body: { ...scope, withChanges: true } }
    case 'cancel_share':
      return { kind: 'cancel', body: i.kind === 'provider_outage' ? { providerId: i.providerId } : scope }
    case 'cancel':
      return { kind: 'cancel', body: {}, whole: true }
    case 'retry':
      return { kind: 'retry', body: { providerId: i.providerId } }
    case 'move':
      return i.detail.move ? { kind: 'move', body: { fromProviderId: i.providerId, toProviderId: i.detail.move.targetId } } : null
  }
  return null
}

/** The header badges of a send. */
export function headerBadges(s: Snapshot, incident: Incident | undefined): { tone: Tone; text: string; pulse?: boolean; live?: boolean }[] {
  if (incident?.kind === 'threshold') {
    const who = scopeName(incident)
    const out: { tone: Tone; text: string; pulse?: boolean; live?: boolean }[] = [{ tone: 'bad', text: who ? `Auto-paused · ${who}` : 'Auto-paused', pulse: true }]
    if (incident.scope === 'mailbox_provider' && s.continuing.length) out.push({ tone: 'ok', text: 'Other mailbox providers still sending' })
    if (incident.scope === 'route') {
      const others = continuingProviders(s, incident)
      if (others.length) out.push({ tone: 'ok', text: `${andList(others.map((p) => p.short))} still sending` })
    }
    return out
  }
  if (incident?.kind === 'provider_outage') {
    const p = s.providers.find((x) => x.id === incident.providerId)
    const out: { tone: Tone; text: string; pulse?: boolean; live?: boolean }[] = [
      { tone: 'warn', text: `${incident.providerShort} route held · ${p?.retrying ? 'retrying every 60 s' : 'waiting for your decision'}` },
    ]
    const others = continuingProviders(s, incident)
    if (others.length) out.push({ tone: 'ok', text: `${andList(others.map((x) => x.short))} still sending` })
    return out
  }
  switch (s.state) {
    case 'preparing':
      return [{ tone: 'info', text: 'Starting' }]
    case 'sending':
    case 'partial':
      return [{ tone: 'ok', text: 'Sending · live', live: true }]
    case 'paused':
      return [{ tone: 'warn', text: s.pausedBy ? `Paused by ${s.pausedBy}` : 'Paused' }]
    case 'scheduled':
      return [{ tone: 'info', text: 'Scheduled' }]
    case 'finished':
      return [{ tone: 'ok', text: 'Sent' }]
    case 'cancelled':
      return [{ tone: 'neutral', text: s.sent > 0 ? 'Cancelled · partially sent' : 'Cancelled' }]
    case 'failed':
      return [{ tone: 'bad', text: 'Failed' }]
  }
  return [{ tone: 'neutral', text: 'Draft' }]
}

/** The open incident that needs a decision (an auto-pause or an outage). */
export function openIncident(s: Snapshot): Incident | undefined {
  if (s.state !== 'partial' && s.state !== 'paused') return undefined
  return s.incident
}

/** Whether the first events haven't arrived yet (the connecting state). */
export function isStarting(s: Snapshot, now: Date): boolean {
  if (s.state === 'preparing') return true
  if (s.state !== 'sending') return false
  if (s.events.length > 0 || s.totals.delivered > 0) return false
  const started = s.startedAt ? new Date(s.startedAt).getTime() : now.getTime()
  return now.getTime() - started < 60_000
}

/** "Starting send · 09:00:04" and what's happening. */
export function startingText(s: Snapshot): { title: string; body: string } {
  const at = s.startedAt ? ` · ${clockSeconds(s.startedAt)}` : ''
  const via = andList(s.providers.map((p) => p.short))
  if (s.state === 'preparing') {
    return {
      title: `Starting send${at}`,
      body: `Building the queue${s.recipients ? ` for ${count(s.recipients)} recipients` : ''}: suppressed, unsubscribed and paused contacts are left out.${via ? ` The first batches go to ${via} as soon as it's ready.` : ''}`,
    }
  }
  return {
    title: `Starting send${at}`,
    body: `Queue built for ${count(s.recipients)} recipients. Handing the first batches to ${via || 'your providers'}; delivery events usually arrive within 10 seconds.`,
  }
}

/** "34 minutes", "1 minute", "1 h 12 min": how long a send took. */
export function tookText(seconds: number): string {
  const m = Math.max(1, Math.round(seconds / 60))
  if (m < 60) return `${m} ${m === 1 ? 'minute' : 'minutes'}`
  return duration(m)
}

/** "Weekly Digest #142 finished at 09:34:12" and the summary under it. */
export function finishedText(s: Snapshot): { title: string; body: string } {
  const t = s.totals
  const at = s.completedAt ? ` at ${clockSeconds(s.completedAt)}` : ''
  const took = s.durationSeconds !== undefined ? ` in ${tookText(s.durationSeconds)}` : ''
  const trips = s.incidents.filter((i) => i.kind === 'threshold').length
  const outages = s.incidents.filter((i) => i.kind === 'provider_outage').length
  let tripped = trips === 0 ? 'No threshold tripped.' : trips === 1 ? 'One threshold paused part of the send.' : `${trips} thresholds paused parts of the send.`
  if (outages) tripped += ` ${outages === 1 ? 'A provider outage' : `${outages} provider outages`} held part of it.`
  return {
    title: `${s.name} finished${at}`,
    body: `${count(s.sent)} sent${took}: ${count(t.delivered)} delivered (${pct1(share(t.delivered, s.sent))}), ${count(t.bounced)} bounced, ${count(t.complained)} ${t.complained === 1 ? 'complaint' : 'complaints'}. ${tripped} Opens and clicks keep updating in the report for 72 hours.`,
  }
}

/** The cancelled state's sentence. */
export function cancelledText(s: Snapshot): { title: string; body: string } {
  const at = s.completedAt ? ` at ${clockSeconds(s.completedAt)}` : ''
  if (s.sent === 0) return { title: `${s.name} was cancelled${at}`, body: 'Nothing was sent. The campaign can be scheduled again from Review & send.' }
  return {
    title: `${s.name} was cancelled${at}`,
    body: `${count(s.sent)} of ${count(s.recipients)} were sent before it stopped; ${count(s.unsent)} weren't sent and are marked "not sent" in the report. Opens and clicks keep updating for 72 hours.`,
  }
}

/** The manual pause banner. */
export function pausedText(s: Snapshot): { title: string; body: string } {
  const by = s.pausedBy ? ` by ${s.pausedBy}` : ''
  const at = s.pausedAt ? ` at ${clockSeconds(s.pausedAt)}` : ''
  const reason = s.pauseReason ? `"${s.pauseReason}" ` : ''
  const waiting = s.held + s.queued + s.waiting
  return {
    title: `Paused${by}${at}`,
    body: `${reason}${count(waiting)} recipients are waiting. Nothing more is sent until someone with campaigns:send resumes it; what was already sent keeps reporting opens and clicks.`.trim(),
  }
}

/** "Lost the live event stream at 09:18:51" and what that means. */
export function lostText(at: Date, lastUpdate: string | undefined, attempt: number, max: number, stopped: boolean, now: Date): { title: string; body: string } {
  const since = lastUpdate ? new Date(lastUpdate) : at
  const behind = Math.max(1, Math.ceil((now.getTime() - since.getTime()) / 60000))
  const asOf = lastUpdate ? clockSeconds(lastUpdate) : clockSeconds(at)
  const retry = stopped ? `Stopped reconnecting after ${max} attempts.` : `Reconnecting automatically (attempt ${Math.max(1, attempt)} of ${max}).`
  return {
    title: `Lost the live event stream at ${clockSeconds(at)}`,
    body: `The browser's connection to ReputeCast dropped. Sending continues on the server; the counts below are from ${asOf} and may be up to ${behind} ${behind === 1 ? 'minute' : 'minutes'} behind. ${retry}`,
  }
}

/** Mobile progress line: "ETA 13 min · about 09:34". */
export function mobileEta(s: Snapshot): string {
  if (s.etaSeconds === undefined) return s.state === 'paused' ? 'Paused' : ''
  return `ETA ${duration(s.etaSeconds / 60)}${s.finishesAt ? ` · about ${clock(s.finishesAt)}` : ''}`
}

/** The delivered-share legend: whole percents that add up to 100 (largest remainder). */
export function deliveredMix(s: Snapshot): { key: MailboxKey; name: string; percent: number }[] {
  const rows = s.deliveredMix.map((m) => ({ key: m.key as MailboxKey, name: m.name, exact: m.percent, percent: Math.floor(m.percent) }))
  const total = rows.reduce((a, r) => a + r.exact, 0)
  if (total > 0) {
    let left = Math.round(total) - rows.reduce((a, r) => a + r.percent, 0)
    for (const r of [...rows].sort((a, b) => b.exact - Math.floor(b.exact) - (a.exact - Math.floor(a.exact)))) {
      if (left <= 0) break
      r.percent += 1
      left -= 1
    }
  }
  return rows.map(({ key, name, percent }) => ({ key, name, percent }))
}

// --- Progress card legend and bar -------------------------------------------

/** Segment widths as a share of the audience; the bar's track is the rest. */
export function barSegments(s: Snapshot, incident?: Incident): (Segment & { width: number })[] {
  const base = Math.max(s.recipients, 1)
  return progressSegments(s, incident).map((seg) => ({ ...seg, width: Math.min(100, (seg.value / base) * 100) }))
}

export type LegendItem = { key: string; color: string; text: string }

/** What is still to send now (queued, and waiting for the A/B winner), outside any hold. */
export function queuedCount(s: Snapshot, incident?: Incident): number {
  return s.queued + s.waiting + (incident ? 0 : s.held)
}

/**
 * The legend under the bar: "Amazon SES 17,635 · 14.2 /s", "Held, Outlook
 * 2,522", "Queued 12,010" (the phone layout: "SES 17,635").
 */
export function progressLegend(s: Snapshot, incident?: Incident, mobile = false): LegendItem[] {
  const items: LegendItem[] = s.providers
    .filter((p) => p.sent > 0 || p.planned > 0)
    .map((p) => ({ key: p.id, color: providerColor[p.color], text: mobile ? `${p.short} ${count(p.sent)}` : `${p.name} ${count(p.sent)} · ${perSec(p.rate)}` }))
  if (mobile) return items
  if (incident && s.held > 0) {
    if (incident.kind === 'provider_outage') {
      items.push({ key: 'held', color: heldRouteColor, text: `Held on ${incident.providerShort} ${count(s.held)}` })
    } else {
      const who = scopeName(incident)
      items.push({ key: 'held', color: heldMailboxColor, text: who ? `Held, ${who} ${count(s.held)}` : `Held ${count(s.held)}` })
    }
  }
  const queued = queuedCount(s, incident)
  if (queued > 0) {
    const others = incident?.kind === 'provider_outage' ? continuingProviders(s, incident) : []
    items.push({ key: 'queued', color: queuedColor, text: others.length === 1 ? `Queued on ${others[0]!.short} ${count(queued)}` : `Queued ${count(queued)}` })
  }
  return items
}

// --- Phone layout (LiveMonitorMobile) ----------------------------------------

/** The phone layout's tiles: shorter captions, human opens first. */
export function mobileTiles(s: Snapshot): Tile[] {
  const t = s.totals
  return [
    { key: 'delivered', label: 'Delivered', value: count(t.delivered), sub: pct1(share(t.delivered, s.sent) ?? 0) },
    { key: 'bounced', label: 'Bounced', value: count(t.bounced), sub: rate(share(t.bounced, s.sent) ?? 0) },
    { key: 'complained', label: 'Complained', value: count(t.complained), sub: pct2(share(t.complained, t.delivered) ?? 0) },
    { key: 'opened', label: 'Opened · human', value: count(t.openedHuman), sub: `+${count(t.openedMachine)} machine` },
    {
      key: 'clicked',
      label: 'Clicked',
      value: count(t.clicked),
      sub: `${pct1(share(t.clicked, t.delivered) ?? 0)} · ${pct1(share(t.clicked, t.opened) ?? 0)} click-to-open`,
    },
  ]
}

/** "14.2 /s · 99.1% delivered" under a provider's name on phones. */
export function providerLine(p: Provider): string {
  return `${perSec(p.rate)} · ${pct1(p.deliveredRate ?? 0)} delivered`
}

// --- Captions ------------------------------------------------------------------

export type CaptionKind = 'live' | 'threshold' | 'outage'

/**
 * The top bar's line about events: "Events via SNS and Postal webhooks ·
 * last 1 s ago · pause needs campaigns:pause, resume needs campaigns:send".
 */
export function eventsCaption(s: Snapshot, now: Date, kind: CaptionKind, streaming = true): string {
  const last = s.lastEventAt ? `last ${liveAgo(s.lastEventAt, now)}` : 'waiting for the first events'
  const head = !streaming ? 'Reconnecting to the event stream' : kind === 'live' && s.eventsVia ? `Events via ${s.eventsVia}` : 'Events live'
  switch (kind) {
    case 'threshold':
      return `${head} · ${last} · anyone with campaigns:pause can stop a send; resume needs campaigns:send`
    case 'outage':
      return `${head} · ${last}`
  }
  return `${head} · ${last} · pause needs campaigns:pause, resume needs campaigns:send`
}

/** The phone header's line: "Live · events 1 s ago". */
export function mobileCaption(s: Snapshot, now: Date, streaming = true): string {
  if (!streaming) return 'Reconnecting…'
  return s.lastEventAt ? `Live · events ${liveAgo(s.lastEventAt, now)}` : 'Live · waiting for the first events'
}

/** API error rates: "0.1%", "0.4%", "100%". */
export function errorRate(v: number | undefined): string {
  if (v === undefined) return '—'
  return v >= 10 ? `${Math.round(v)}%` : `${v.toFixed(1)}%`
}

/** "17,635 / 28,444": sent of the provider's planned share. */
export function sentOfPlanned(p: Provider): string {
  return `${count(p.sent)} / ${count(p.planned)}`
}

/** The complaint column's heading in the decision view: "Complaint · limit 0.10%". */
export function complaintHeading(s: Snapshot): string {
  const limit = s.thresholds.find((t) => t.key === 'mailbox_complaint')?.limit ?? s.thresholds.find((t) => t.key === 'complaint')?.limit
  return limit === undefined ? 'Complaint' : `Complaint · limit ${pct2(limit)}`
}

/** "Automations and System already failed over to Postmark and SES on their own". */
export function failoverCaption(s: Snapshot): string {
  if (s.failovers.length) return `${andList(s.failovers.map((f) => f.stream))} already failed over to ${andList(s.failovers.map((f) => f.to))} on their own`
  const held = s.providers.filter((p) => p.routeStatus === 'held').map((p) => p.short)
  if (!held.length) return 'who delivers the mail'
  if (!s.sharedStreams.length) return `Automations and System don't use ${andList(held)}`
  return `${andList(s.sharedStreams)} also ${s.sharedStreams.length === 1 ? 'uses' : 'use'} ${andList(held)}`
}

/** "Delivered share: Gmail 46%, Outlook 21%, …" for the stacked bar. */
export function mixLabel(mix: { name: string; percent: number }[]): string {
  return `Delivered share: ${mix.map((m) => `${m.name} ${m.percent}%`).join(', ')}`
}

/** "Scheduled for Tue 14 Oct · 09:00". */
export function scheduledLine(at: string | undefined): string {
  return at ? `Scheduled for ${shortDate(at)} · ${clock(at)}` : 'Scheduled'
}
