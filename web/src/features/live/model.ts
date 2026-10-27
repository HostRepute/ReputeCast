import type { Schemas } from '@/api/client'
import { m as pm } from '@/features/providers/messages'
import { streamName, type Stream } from '@/features/providers/common'
import { t as tt, type MessageDescriptor } from '@/i18n'
import { ago, andList, clock, clockSeconds, count, decimal, duration, longDate, money, monthDay, orList, pct, rate, shortDate, trimPct } from '@/lib/format'
import { mm } from './model.messages'

/**
 * The live sending monitor's words and numbers (spec 7.6; boards
 * LiveMonitor, LiveMonitorPaused, LiveMonitorOutage, LiveMonitorStates,
 * LiveMonitorMobile), computed from the engine's snapshot. Pure functions,
 * pinned by model.test.ts against the boards' copy. The words come from
 * model.messages.ts in the app language; numbers and times from lib/format.
 */

export type Snapshot = Schemas['LiveSnapshot']
export type Mailbox = Schemas['LiveMailbox']
export type Provider = Schemas['LiveProvider']
export type Incident = Schemas['LiveIncident']
export type LiveEvent = Schemas['LiveEvent']
export type Threshold = Schemas['LiveThreshold']
export type IncidentOption = Schemas['IncidentOption']
export type MoveEstimate = Schemas['MoveEstimate']
export type WarmupHold = Schemas['LiveWarmupHold']
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

/** Mailbox providers by key; "Other" is a word, the rest are names. */
export const mailboxNames: Record<MailboxKey, string> = {
  gmail: 'Gmail',
  outlook: 'Outlook',
  yahoo: 'Yahoo',
  icloud: 'iCloud',
  get other() {
    return tt(mm.mailboxOther)
  },
}

export function mailboxName(key: string | undefined): string {
  return (key && mailboxNames[key as MailboxKey]) || key || ''
}

/** A list of mailbox provider names from the server, with "Other" in the app language. */
function mailboxNamesOf(names: string[]): string[] {
  return names.map((n) => (n === 'Other' ? mailboxNames.other : n))
}

const streamKeys: Record<string, Stream> = { Broadcasts: 'broadcast', Automations: 'automation', System: 'system', Transactional: 'transactional' }

/** A stream's name from the server ("Automations"), in the app language. */
function streamLabel(name: string): string {
  const key = streamKeys[name]
  return key ? streamName[key] : name
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

/** One decimal: 99.3%. */
export function pct1(v: number | null | undefined): string {
  return pct(v, 1)
}

/** Two decimals, for complaint rates: 0.02%. */
export function pct2(v: number | null | undefined): string {
  return pct(v, 2)
}

/** A rate as the thresholds write it: complaints with two decimals, the rest with one (two below 1%). */
export function pctOf(metric: string, v: number): string {
  return metric === 'complaint' || v < 1 ? pct(v, 2) : pct(v, 1)
}

/**
 * What the resume dialog offers (review GS-2): the recommended slower rate
 * of a mailbox provider or route, and the limit raised to twice what it
 * was, within what the API accepts.
 */
export function resumeChoices(i: Incident): { rate?: number; rateText: string; raised: number } {
  const max = i.metric === 'complaint' ? 5 : i.metric === 'deferral' ? 100 : 20
  const raised = Math.min(max, Math.round((i.limit ?? 0) * 2 * 100) / 100)
  const rate = i.scope !== 'campaign' ? i.detail.changes?.rate : undefined
  return { rate: rate || undefined, rateText: rate ? wholePerSec(rate) : '', raised }
}

/** n of d as a percentage, or undefined without a base. */
export function share(n: number, d: number): number | undefined {
  return d > 0 ? (n / d) * 100 : undefined
}

/** A live rate: "14.2 /s", "2.0 /s", "0 /s". */
export function perSec(v: number): string {
  return tt(mm.perSecond, { rate: v <= 0 ? decimal(0, 0) : decimal(v, 1) })
}

/** A throttle or a rounded rate: "5 /s", "14 /s". */
export function wholePerSec(v: number): string {
  const r = Number.isInteger(v) ? v : Math.round(v * 10) / 10
  return tt(mm.perSecond, { rate: decimal(r, Number.isInteger(r) ? 0 : 1) })
}

/** "62%": whole percent, never 100% before everything is sent. */
export function progressText(s: Pick<Snapshot, 'sent' | 'recipients' | 'percent'>): string {
  if (s.recipients <= 0) return pct(0, 0)
  if (s.sent >= s.recipients) return pct(100, 0)
  return pct(Math.min(99, Math.round((s.sent / s.recipients) * 100)), 0)
}

/** "1 min 26 s", "45 s", "3 min". */
export function secondsText(total: number): string {
  const s = Math.max(0, Math.round(total))
  if (s < 60) return tt(mm.seconds, { s })
  const m = Math.floor(s / 60)
  const r = s % 60
  if (m >= 60) return duration(m)
  return r ? tt(mm.minutesSeconds, { m, s: r }) : tt(mm.minutes, { m })
}

/** "$0.12", and "$0" for nothing (self-hosted providers). */
export function cost(v: number): string {
  return v === 0 ? money(0, 0) : money(v)
}

/** "1 s ago", "40 s ago", "3 min ago" for a live feed. */
export function liveAgo(at: string | undefined, now: Date): string {
  if (!at) return tt(mm.noEventsYet)
  const s = Math.round((now.getTime() - new Date(at).getTime()) / 1000)
  if (s < 60) return tt(mm.secondsAgo, { n: Math.max(1, s) })
  if (s < 3600) return tt(mm.minutesAgo, { n: Math.round(s / 60) })
  return ago(at, now)
}

/** "Oct 10 16:20", as the header writes approvals. */
export function shortStamp(d: string): string {
  return `${monthDay(d)} ${clock(d)}`
}

/** "Started by Sofia Rossi · approved by Marcus Chen, Oct 10 16:20". */
export function startedLine(s: Snapshot): string {
  const lead = s.startedBy ? 'started' : s.state === 'scheduled' && s.scheduledBy ? 'scheduled' : ''
  const person = s.startedBy || s.scheduledBy
  const approver = s.approvedBy
  const when = s.approvedAt ? shortStamp(s.approvedAt) : ''
  if (!approver) {
    if (lead === 'started') return tt(mm.startedBy, { person })
    if (lead === 'scheduled') return tt(mm.scheduledBy, { person })
    return ''
  }
  const v = { person, approver, when }
  if (lead === 'started') return tt(when ? mm.startedApprovedAt : mm.startedApproved, v)
  if (lead === 'scheduled') return tt(when ? mm.scheduledApprovedAt : mm.scheduledApproved, v)
  return tt(when ? mm.approvedByAt : mm.approvedBy, v)
}

const needsMessages = {
  pause: mm.needsPause,
  resume: mm.needsResume,
  cancel: mm.needsCancel,
  decide: mm.needsDecide,
  rates: mm.needsRates,
  warmupCancel: mm.needsWarmupCancel,
}

/** Who can help with a permission: "Pausing a send needs the campaigns:pause permission. Ask Marcus Chen or Linh Nguyen." */
export function needs(action: keyof typeof needsMessages, perm: string, helpers: string[] | undefined): string {
  const line = tt(needsMessages[action], { perm })
  return helpers?.length ? `${line} ${tt(mm.askHelpers, { helpers: orList(helpers) })}` : line
}

export type Segment = { key: string; value: number; color: string; title: string }

/** The providers' sent shares, then what's held (boards: red for a mailbox provider, amber for a route). */
export function progressSegments(s: Snapshot, incident?: Incident): Segment[] {
  const segs: Segment[] = s.providers.filter((p) => p.sent > 0).map((p) => ({ key: p.id, value: p.sent, color: providerColor[p.color], title: p.short }))
  if (s.held > 0) {
    const route = incident?.kind === 'provider_outage' || (!incident && warmupHolds(s).length > 0)
    segs.push({ key: 'held', value: s.held, color: route ? heldRouteColor : heldMailboxColor, title: tt(mm.held) })
  }
  return segs
}

/** "19,594 of 31,604 sent", with what's held in the decision views. */
export function sentLine(s: Snapshot, incident?: Incident): string {
  const v = { sent: count(s.sent), total: count(s.recipients), held: count(s.held) }
  const warm = warmupHolds(s)
  if (!incident && warm.length === 1 && s.held > 0) return tt(mm.sentOfHeldOn, { ...v, provider: warm[0]!.short })
  if (!incident || s.held <= 0) return tt(mm.sentOf, v)
  if (incident.kind === 'provider_outage') return tt(mm.sentOfHeldOn, { ...v, provider: incident.providerShort })
  return tt(mm.sentOfHeld, v)
}

/** "Started 09:00:04 · ETA 13 min · finishes about 09:34". */
export function timingLine(s: Snapshot, incident?: Incident): string {
  const parts: string[] = []
  if (s.startedAt) parts.push(tt(mm.timingStarted, { time: clockSeconds(s.startedAt) }))
  const eta = s.etaSeconds !== undefined ? duration(s.etaSeconds / 60) : ''
  if (incident?.kind === 'threshold' && incident.scope !== 'campaign') {
    // "Outlook held since 09:21:14 · the rest: SES done about 09:28 · Postal about 09:29".
    parts.push(tt(mm.timingHeldSince, { who: scopeName(incident), time: clockSeconds(incident.trippedAt) }))
    const routes = routeDoneLine(s)
    if (routes) parts.push(tt(mm.timingTheRest, { routes }))
    else if (eta) parts.push(tt(mm.timingEtaRest, { eta }))
  } else if (incident?.kind === 'provider_outage') {
    // "SES done about 09:26 · Postal held since 09:17:10".
    const others = continuingProviders(s, incident)
    const routes = routeDoneLine({ ...s, providers: others })
    if (routes) parts.push(routes)
    else if (s.finishesAt && others.length) parts.push(tt(mm.timingProvidersFinish, { providers: andList(others.map((p) => p.short)), time: whenText(s.finishesAt, s.at) }))
    parts.push(tt(mm.timingHeldSince, { who: incident.providerShort, time: clockSeconds(incident.trippedAt) }))
  } else if (s.state === 'sending' || s.state === 'partial') {
    // When each route is done at current rates, with the per-mailbox limits (review GS-8).
    const routes = routeDoneLine(s)
    if (routes) return tt(mm.timingRoutes, { time: s.startedAt ? clockSeconds(s.startedAt) : '', routes })
    if (eta) parts.push(tt(mm.timingEta, { eta }))
    if (s.finishesAt) parts.push(tt(mm.timingFinishes, { time: whenText(s.finishesAt, s.at) }))
  } else if (s.state === 'paused' && s.pausedAt) {
    parts.push(tt(mm.timingPausedSince, { time: clockSeconds(s.pausedAt) }))
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
  const rates = [first, ...rest].map((v) => pct(v.rate, 1)).reduce((a, b) => tt(mm.abVs, { a, b }))
  return tt(mm.abLine, { phase: ab.phase === 'decided' ? 'decided' : 'testing', kind, variant: first.key, rates, metric, sample: pct(ab.sample, 0) })
}

export type TilesVariant = 'live' | 'threshold' | 'outage'
/** A counter: `num` is the count behind `value`, for counting up. */
export type Tile = { key: string; label: string; value: string; sub: string; alert?: boolean; num?: number }

function countTile(key: string, label: MessageDescriptor, n: number, sub: string): Tile {
  return { key, label: tt(label), value: count(n), sub, num: n }
}

export function tiles(s: Snapshot, variant: TilesVariant = 'live', incident?: Incident): Tile[] {
  const t = s.totals
  const delivered = countTile('delivered', mm.tileDelivered, t.delivered, tt(mm.deliveredRetrying, { rate: pct1(share(t.delivered, s.sent)), retrying: s.retrying }))
  const bounceRate = rate(share(t.bounced, s.sent) ?? 0)
  const bounced = countTile(
    'bounced',
    mm.tileBounced,
    t.bounced,
    variant === 'outage' ? bounceRate : tt(mm.bouncedSub, { rate: bounceRate, unknown: count(t.unknownBounced), block: count(t.blockBounced), soft: count(t.softBounced) }),
  )
  let complained = countTile('complained', mm.tileComplained, t.complained, complaintSub(s, variant))
  if (variant === 'threshold' && incident?.metric === 'complaint' && incident.scope === 'mailbox_provider') complained = { ...complained, alert: true }
  const opened = countTile(
    'opened',
    mm.tileOpened,
    t.opened,
    tt(mm.openedSub, { rate: pct1(share(t.opened, t.delivered) ?? 0), human: count(t.openedHuman), machine: count(t.openedMachine) }),
  )
  const clickRate = pct1(share(t.clicked, t.delivered) ?? 0)
  const clicked = countTile(
    'clicked',
    mm.tileClicked,
    t.clicked,
    variant === 'live' ? tt(mm.clickedSub, { rate: clickRate, ctor: pct1(share(t.clicked, t.opened) ?? 0) }) : clickRate,
  )
  return [delivered, bounced, complained, opened, clicked]
}

/**
 * The complaints tile's line (review F-2): the rate counts only mail
 * delivered where mailbox providers report complaints; Gmail and iCloud
 * read n/a. The decision view breaks the count down by mailbox provider.
 */
export function complaintSub(s: Snapshot, variant: TilesVariant | 'final'): string {
  const t = s.totals
  const names = andList(t.notReported.map((n) => mailboxName(n)))
  if (t.complaintBase <= 0) return t.notReported.length ? tt(mm.complainedNone, { names }) : tt(mm.notAvailable)
  const rate = pct2(share(t.complained, t.complaintBase))
  switch (variant) {
    case 'live':
      return tt(mm.complainedSubReported, { rate, base: count(t.complaintBase) })
    case 'final':
      return tt(mm.complainedFinal, { rate, base: count(t.complaintBase), counting: s.watch?.active ? 'yes' : 'no' })
    case 'threshold': {
      const parts = s.mailboxes.filter((m) => !m.complaintReason && m.complained > 0).map((m) => `${mailboxName(m.key)} ${count(m.complained)}`)
      return [...parts, t.notReported.length ? tt(mm.complainedNa, { names }) : ''].filter(Boolean).join(' · ')
    }
  }
  return t.notReported.length ? tt(mm.complainedWhereReportedNa, { rate, names }) : tt(mm.complainedWhereReported, { rate })
}

/** The complete state's tiles (LiveMonitorStates: send complete). */
export function finalTiles(s: Snapshot): Tile[] {
  const t = s.totals
  const providers = s.providers.filter((p) => p.sent > 0)
  const bounceRate = rate(share(t.bounced, s.sent) ?? 0)
  return [
    countTile('delivered', mm.tileDelivered, t.delivered, pct1(share(t.delivered, s.sent))),
    countTile('bounced', mm.tileBounced, t.bounced, t.hardBounced > 0 ? tt(mm.bouncedSuppressed, { rate: bounceRate }) : bounceRate),
    countTile('complained', mm.tileComplained, t.complained, complaintSub(s, 'final')),
    { key: 'cost', label: tt(mm.tileCost), value: cost(s.cost.total), sub: providers.map((p) => `${p.short} ${cost(p.cost)}`).join(' · ') },
  ]
}

/** "SES 28,444 · 99.2%" lines of the complete state's last tile. */
export function providerResultLines(s: Snapshot): string[] {
  return s.providers.filter((p) => p.sent > 0).map((p) => `${p.short} ${count(p.sent)} · ${pct1(p.deliveredRate)}`)
}

export type ThresholdRow = {
  key: string
  /** "Bounces", "Outlook complaints". */
  label: string
  /** "whole send", "highest mailbox provider". */
  sub: string
  /** The phone layout's short label. */
  short: string
  /** "0.60% of 2.0%", "Yahoo 3.1% of 15%". */
  value: string
  /** "118 of 19,594 sent", "needs 5 complaints and 500 delivered to pause". */
  evidence: string
  state: { tone: Tone; text: string }
  width: number
  color: string
}

const stateTone: Record<Threshold['state'], Tone> = { ok: 'ok', near: 'warn', collecting: 'neutral', tripped: 'bad', na: 'neutral' }

/**
 * The thresholds card's rows (LiveMonitor, review GS-6): each rate against
 * its limit with its state, and the evidence behind it. A rate without
 * the evidence to trip shows "Collecting evidence" with a grey bar.
 */
export function thresholdRows(s: Snapshot): ThresholdRow[] {
  return s.thresholds.map((t) => {
    const mailbox = mailboxName(t.mailbox)
    const v = t.value ?? undefined
    const ofLimit = (value: string, limit: string) => (t.mailbox && t.key !== 'mailbox_complaint' ? tt(mm.ofLimitWho, { who: mailbox, value, limit }) : tt(mm.ofLimit, { value, limit }))
    let label: string, sub: string, short: string, value: string
    switch (t.key) {
      case 'bounce':
        ;[label, sub, short] = [tt(mm.thresholdBounce), tt(mm.thresholdWholeSend), tt(mm.thresholdBounceShort)]
        value = ofLimit(pct(v, 2), pct(t.limit, 1))
        break
      case 'block_bounce':
        ;[label, sub, short] = [tt(mm.thresholdBlock), tt(mm.thresholdPerMailbox), tt(mm.thresholdBlockShort)]
        value = ofLimit(pct(v, 2), pct(t.limit, 1))
        break
      case 'deferral':
        ;[label, sub, short] = [tt(mm.thresholdDeferral), t.window ? tt(mm.thresholdPerMailboxWindow, { window: t.window }) : tt(mm.thresholdPerMailbox), tt(mm.thresholdDeferralShort)]
        value = ofLimit(pct(v, 1), trimPct(t.limit))
        break
      case 'complaint':
        ;[label, sub, short] = [tt(mm.thresholdComplaint), tt(mm.thresholdWhereReported), tt(mm.thresholdComplaintShort)]
        value = ofLimit(pct(v, 2), pct(t.limit, 2))
        break
      case 'mailbox_complaint':
        ;[label, sub, short] = [tt(mm.thresholdMailbox, { mailbox }), tt(mm.thresholdHighest), tt(mm.thresholdMailboxShort, { mailbox })]
        value = ofLimit(pct(v, 2), pct(t.limit, 2))
        break
      case 'reply_rules':
        ;[label, sub, short] = [tt(mm.thresholdReplies), tt(mm.thresholdRepliesSub, { count: t.minBase }), tt(mm.thresholdRepliesShort)]
        value = tt(mm.thresholdRepliesValue, { who: mailbox, code: t.parts?.[0]?.name ?? '', count: count(t.events), limit: count(t.minEvents) })
        break
      default:
        ;[label, sub, short] = [tt(mm.thresholdErrors), tt(mm.thresholdErrorsSub), tt(mm.thresholdErrorsShort)]
        value = ofLimit(errorRate(v), trimPct(t.limit))
    }
    const collecting = t.state === 'collecting'
    return {
      key: t.key + (t.mailbox ?? ''),
      label,
      sub,
      short,
      value: t.state === 'na' ? tt(mm.notAvailable) : value,
      evidence: thresholdEvidence(t),
      state: { tone: stateTone[t.state], text: tt(mm.thresholdState, { state: t.state }) },
      width: usageWidth(v ?? 0, t.limit),
      color: collecting || t.state === 'na' ? 'var(--ink-4)' : usageColor(v ?? 0, t.limit),
    }
  })
}

/** The line under a threshold's bar: its counts, what it still needs, or why there's no rate. */
export function thresholdEvidence(t: Threshold): string {
  const unit = t.key === 'complaint' || t.key === 'mailbox_complaint' ? 'delivered' : 'sent'
  if (t.state === 'na') return tt(mm.thresholdNa, { reason: t.reason ?? 'no_data' })
  if (t.state === 'collecting' && (t.events < t.minEvents || t.base < t.minBase))
    return tt(mm.thresholdNeeds, { key: t.key, events: count(t.events), base: count(t.base), unit, minEvents: count(t.minEvents), minBase: count(t.minBase) })
  switch (t.key) {
    case 'complaint': {
      const names = (t.parts ?? []).map((p) => mailboxName(p.key ?? p.name))
      return names.length ? tt(mm.thresholdComplaintsTo, { events: count(t.events), base: count(t.base), names: andList(names) }) : tt(mm.thresholdIn, { events: count(t.events), base: count(t.base), unit })
    }
    case 'api_errors':
      return (t.parts ?? []).map((p) => `${p.name} ${errorRate(p.value ?? undefined)}`).join(', ')
    case 'deferral':
    case 'block_bounce':
      return tt(mm.thresholdHighestAction, { action: t.action ?? 'pause_mailbox', limit: t.key === 'deferral' ? trimPct(t.limit) : pct(t.limit, 1) })
    case 'reply_rules':
      return t.parts?.[0]?.name
        ? tt(mm.thresholdReplyAction, {
            who: mailboxName(t.mailbox),
            limit: count(t.minEvents),
            action: t.action ?? 'pause',
            timed: t.duration ? 'yes' : 'no',
            duration: t.duration ? duration(t.duration) : '',
          })
        : ''
  }
  return tt(mm.thresholdIn, { events: count(t.events), base: count(t.base), unit })
}

export function scopeLine(s: Snapshot): string {
  return s.thresholdScope === 'campaign' ? tt(mm.scopeCampaign) : tt(mm.scopeRoute)
}

/** A mailbox provider's state word: the live view shows health, the decision view activity. */
export function mailboxStatus(m: Mailbox, s: Snapshot, decision: boolean): { tone: Tone; text: string } {
  if (m.held) return { tone: 'bad', text: m.heldSince ? tt(mm.mailboxPausedAt, { time: clock(m.heldSince) }) : tt(mm.mailboxPaused) }
  if (decision) {
    if (m.remaining > 0) return { tone: 'ok', text: tt(mm.mailboxSending) }
    return { tone: 'neutral', text: tt(mm.mailboxDone) }
  }
  const limit = s.thresholds.find((t) => t.key === 'mailbox_complaint')?.limit ?? 0.1
  // Over the limit without a pause yet: the engine pauses once there are
  // enough complaints to be sure it isn't chance.
  if (m.complaintRate !== undefined && m.complaintRate > limit && m.delivered >= 500) return { tone: 'warn', text: tt(mm.mailboxOverLimit) }
  if (m.complaintRate !== undefined && m.complaintRate >= limit * 0.8 && m.delivered >= 500) return { tone: 'warn', text: tt(mm.mailboxNearLimit) }
  // Still going out (LiveMonitor: "Sending").
  if (m.remaining > 0 && (s.state === 'sending' || s.state === 'partial')) return { tone: 'ok', text: tt(mm.mailboxSending) }
  return { tone: 'ok', text: tt(mm.mailboxOk) }
}

/** "full", "5 /s" or "held". */
export function mailboxRate(m: Mailbox): string {
  if (m.held) return tt(mm.rateHeld)
  return m.throttle ? wholePerSec(m.throttle) : tt(mm.rateFull)
}

/**
 * The progress card's summary line (LiveMonitor): the send's state in a
 * sentence, with a tone for its icon. It changes only when the send's or a
 * threshold's state changes, so it can be the screen's one polite live
 * region.
 */
export function summaryLine(s: Snapshot, incident?: Incident): { tone: Tone; text: string; held?: boolean } {
  if (incident && incident.kind === 'provider_outage') {
    const names = continuingProviders(s, incident).map((p) => p.name)
    return {
      tone: 'warn',
      held: true,
      text: tt(mm.sumRouteHeld, { who: incident.provider || incident.providerShort, time: clockSeconds(incident.trippedAt), count: names.length, names: andList(names) }),
    }
  }
  if (incident) {
    const time = clockSeconds(incident.trippedAt)
    if (incident.scope === 'campaign') return { tone: 'bad', held: true, text: tt(mm.sumCampaignPaused, { time }) }
    const names =
      incident.scope === 'route' ? continuingProviders(s, incident).map((p) => p.name) : mailboxNamesOf(s.continuing.length ? s.continuing : (incident.detail.continuing ?? []))
    const msg = incident.scope === 'route' ? mm.sumRouteHeld : mm.sumMailboxPaused
    return { tone: 'bad', held: true, text: tt(msg, { who: scopeName(incident), time, count: names.length, names: andList(names) }) }
  }
  if (s.state === 'paused') return { tone: 'warn', held: true, text: tt(mm.sumPaused, { time: clockSeconds(s.pausedAt ?? s.at) }) }
  if (s.state === 'preparing') return { tone: 'ok', text: tt(mm.sumPreparing) }
  // A share today's warm-up cap holds goes on with tomorrow's cap (review PR-8).
  const warm = warmupHolds(s)[0]
  if (warm?.warmupHold) {
    const names = s.providers.filter((p) => !p.warmupHold && (p.routeStatus === 'sending' || p.routeStatus === 'pending') && p.remaining > 0).map((p) => p.name)
    const h = warm.warmupHold
    return {
      tone: 'warn',
      held: true,
      text: tt(mm.sumWarmupHeld, { who: warm.name, paused: h.paused ? 'yes' : 'no', time: h.until ? clock(h.until) : '', count: names.length, names: andList(names) }),
    }
  }
  const near = s.thresholds.filter((x) => x.state === 'near')
  // "Canary passed at 09:20." first, once it did (LiveMonitor).
  const c = s.canary
  const lead = c && (c.phase === 'passed' || c.phase === 'continued') ? `${tt(mm.sumCanaryLead, { phase: c.phase, time: clock(c.decidedAt ?? s.at) })} ` : ''
  if (near.length) {
    const names = andList(thresholdRows({ ...s, thresholds: near }).map((r) => r.label))
    return { tone: 'warn', text: lead + tt(mm.sumNear, { count: near.length, names }) }
  }
  const collecting = s.thresholds.some((x) => x.state === 'collecting')
  return { tone: 'ok', text: lead + tt(collecting ? mm.sumCollecting : mm.sumSending) }
}

/** A sending provider's state word. */
export function providerStatus(p: Provider, mobile = false): { tone: Tone; text: string } {
  if (p.warmupHold) return { tone: 'warn', text: tt(p.warmupHold.paused ? mm.providerWarmupPaused : mm.providerWarmupHeld) }
  if (p.retrying && p.routeStatus === 'held') return { tone: 'warn', text: tt(mm.providerRetrying, { done: p.probes, needed: p.probesNeeded }) }
  if (p.downSince && (p.routeStatus === 'held' || p.status === 'down')) return { tone: 'bad', text: tt(mm.providerDownSince, { time: clock(p.downSince) }) }
  if (p.routeStatus === 'held' || p.routeStatus === 'paused') return { tone: 'warn', text: tt(mm.providerHeld) }
  if (p.status === 'down') return { tone: 'bad', text: tt(pm.statusDown) }
  if (p.status === 'degraded') return { tone: 'warn', text: tt(pm.statusDegraded) }
  if (p.warmupDay !== undefined) return { tone: 'info', text: mobile ? tt(mm.providerWarmup) : tt(pm.statusWarmingUp) }
  return { tone: 'ok', text: tt(pm.statusHealthy) }
}

/** The line under a provider's name: quota, warm-up, or where it runs when it's down. */
export function providerNote(p: Provider, decision: boolean): string {
  if (decision && (p.routeStatus === 'held' || p.downSince)) return [p.host, p.ip].filter(Boolean).join(' · ')
  if (decision) return ''
  if (p.warmupDay !== undefined && p.warmupCap !== undefined)
    return tt(mm.providerWarmupNote, { day: p.warmupDay, sent: count(p.warmupSent ?? 0), cap: count(p.warmupCap) })
  if (p.quotaFree !== undefined) return tt(mm.providerQuotaNote, { free: count(p.quotaFree) })
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

/** The metrics an incident's words know; another one keeps its own name. */
function metricKey(metric: string): string {
  return metric === 'complaint' || metric === 'bounce' || metric === 'deferral' ? metric : 'other'
}

/** The values a metric's messages take: its key, its name and the name capitalised. */
function metricValues(metric: string) {
  return { metric: metricKey(metric) === 'other' ? metric : metricKey(metric), Metric: metric ? metric[0]!.toUpperCase() + metric.slice(1) : metric }
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
  if (i.scope === 'mailbox_provider') return i.mailbox === 'other' ? mailboxNames.other : i.mailboxName || mailboxName(i.mailbox)
  if (i.scope === 'route') return i.providerShort || i.provider
  return ''
}

/** "Outlook complaint rate reached 0.18% (limit 0.10%)". */
export function incidentTitle(i: Incident): string {
  if (i.kind === 'provider_outage') return outageTitle(i)
  const who = scopeName(i)
  const v = { ...metricValues(i.metric), who, value: rateText(i.metric, i.value), limit: limitText(i.metric, i.limit) }
  return who ? tt(mm.incidentTitle, v) : tt(mm.incidentTitleAll, v)
}

/** How long an outage's failures lasted, in whole minutes (at least 1, 3 without a window). */
function outageMinutes(d: Incident['detail']): number {
  if (!d.from || !d.to) return 3
  return Math.max(1, Math.round((new Date(d.to).getTime() - new Date(d.from).getTime()) / 60000))
}

/** "Postal returned connection errors for 3 minutes". */
export function outageTitle(i: Incident): string {
  const d = i.detail
  const name = i.provider || i.providerShort
  if (d.retried) return tt(mm.outageStillDown, { name })
  if (d.reason === 'auth') return tt(mm.outageAuth, { name })
  if (d.reason === 'identity') return tt(mm.outageIdentity, { name })
  if (d.reason === 'quota') return tt(mm.outageQuota, { name })
  if (d.reason === 'api_errors') return tt(mm.outageApiErrors, { name, rate: rate(d.errorRate ?? i.value ?? 0) })
  if (d.reason === 'throttled') return tt(mm.outageThrottled, { name, mins: outageMinutes(d) })
  if (!d.from || !d.to) return d.errorKind ? tt(mm.outageReturned, { name, errorKind: d.errorKind }) : tt(mm.outageDown, { name })
  const mins = Math.max(1, Math.round((new Date(d.to).getTime() - new Date(d.from).getTime()) / 60000))
  return d.errorKind ? tt(mm.outageReturnedFor, { name, errorKind: d.errorKind, mins }) : tt(mm.outageErrorsFor, { name, mins })
}

/**
 * "Tripped at 09:21:14, after 4,115 of 6,637 Outlook recipients were sent.
 * 7 complaints out of 3,980 delivered. ReputeCast held the remaining 2,522
 * Outlook recipients. Gmail, Yahoo, iCloud and Other continue, now 71%
 * complete overall."
 */
export function thresholdSummary(i: Incident, s: Snapshot): string {
  const d = i.detail
  const who = scopeName(i)
  const events = d.events ?? 0
  const parts = [
    tt(who ? mm.summaryTrippedWho : mm.summaryTripped, { time: clockSeconds(i.trippedAt), sent: count(d.sent ?? 0), total: count(d.total ?? 0), who }),
    tt(mm.summaryEvents, { ...metricValues(i.metric), events, count: count(events), base: count(d.base ?? 0) }),
  ]
  const progress = progressText(s)
  if (i.scope === 'campaign') {
    parts.push(tt(mm.summaryCampaignPaused, { held: count(i.held), progress }))
    return parts.join(' ')
  }
  parts.push(tt(who ? mm.summaryHeldWho : mm.summaryHeld, { held: count(i.held), who }))
  const continuing =
    i.scope === 'route' ? continuingProviders(s, i).map((p) => p.name) : mailboxNamesOf(s.continuing.length ? s.continuing : (d.continuing ?? []))
  if (continuing.length) parts.push(tt(mm.summaryContinue, { count: continuing.length, names: andList(continuing), progress }))
  else parts.push(tt(mm.summaryNow, { progress }))
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
  const hold = { short, provider: i.provider, sent: count(p?.sent ?? d.sentOnRoute ?? 0), remaining: count(p?.remaining ?? d.remaining ?? i.held) }
  const parts: string[] = []
  if (d.reason === 'auth') {
    parts.push(d.from ? tt(mm.outageAuthSince, { ...hold, from: clockSeconds(d.from) }) : tt(mm.outageAuthAlways, hold))
  } else if (d.reason === 'identity') {
    parts.push(tt(mm.outageIdentityHold, hold))
  } else if (d.reason === 'quota') {
    parts.push(tt(mm.outageQuotaHold, hold))
  } else if (d.reason === 'throttled') {
    parts.push(tt(mm.outageThrottledHold, { ...hold, mins: outageMinutes(d) }))
  } else if (d.reason === 'api_errors') {
    parts.push(tt(mm.outageApiErrorsHold, { ...hold, rate: rate(d.errorRate ?? i.value ?? 0), limit: trimPct(i.limit ?? 5) }))
  } else if (d.from && d.to) {
    parts.push(
      tt(mm.outageWindow, {
        ...hold,
        from: clockSeconds(d.from),
        to: clockSeconds(d.to),
        attempts: d.attempts ?? 0,
        host: d.host || i.provider,
        succeeded: count(d.succeeded ?? 0),
      }),
    )
  } else {
    parts.push(tt(mm.outageHealth, { ...hold, time: clockSeconds(i.trippedAt) }))
  }
  const others = continuingProviders(s, i)
  const progress = progressText(s)
  if (others.length) {
    const r = Math.round(others.reduce((a, x) => a + x.rate, 0))
    parts.push(tt(mm.outageKeepSending, { count: others.length, names: andList(others.map((x) => x.name)), rate: tt(mm.perSecond, { rate: decimal(r, 0) }), progress }))
  } else {
    parts.push(tt(mm.outageNoOther, { progress }))
  }
  if (d.policy === 'pause_and_ask') parts.push(tt(mm.policyPauseAsk))
  else if (d.policy === 'warm_only') parts.push(tt(mm.policyWarmOnly))
  else parts.push(tt(mm.policyNone))
  return parts.join(' ')
}

export type Fact = { label: string; value: string; bad?: boolean }

/** "Complaints", or "Outlook complaints" for a mailbox or sending provider. */
function metricLabel(metric: string, who: string): string {
  const v = { ...metricValues(metric), who }
  return who ? tt(mm.factWho, v) : tt(mm.factAll, v)
}

/** "What tripped": the scope's rate, the limit, the campaign-wide rate and the other metric. */
export function trippedFacts(
  i: Incident,
  s: Snapshot,
): { head: Fact; bar: { width: number }; evidence?: { state: string; text: string }; rows: Fact[]; watch?: string } {
  const d = i.detail
  const who = scopeName(i)
  const t = s.totals
  const head: Fact = {
    label: metricLabel(i.metric, who),
    value: tt(mm.factValue, { events: count(d.events ?? 0), base: count(d.base ?? 0), rate: rateText(i.metric, i.value) }),
    bad: true,
  }
  const rows: Fact[] = [{ label: tt(mm.factLimit, { scope: i.scope }), value: limitText(i.metric, i.limit) }]
  if (i.scope === 'mailbox_provider' && i.metric === 'complaint') {
    // The other mailbox providers that report complaints (review F-2): Gmail and iCloud aren't counted.
    const others = s.mailboxes.filter((m) => m.key !== i.mailbox && !m.complaintReason && m.delivered > 0)
    const n = others.reduce((a, m) => a + m.complained, 0)
    const base = others.reduce((a, m) => a + m.delivered, 0)
    if (others.length)
      rows.push({
        label: tt(mm.factOthers, { names: andList(others.map((m) => mailboxName(m.key))) }),
        value: tt(mm.factValue, { events: count(n), base: count(base), rate: rateText(i.metric, share(n, base) ?? 0) }),
      })
  } else if (i.scope !== 'campaign') {
    const [n, base] = i.metric === 'complaint' ? [t.complained, t.complaintBase] : i.metric === 'bounce' ? [t.bounced, s.sent] : [t.deferred, s.sent]
    rows.push({
      label: tt(mm.factCampaignWide, metricValues(i.metric)),
      value: tt(mm.factValue, { events: count(n), base: count(base), rate: rateText(i.metric, share(n, base) ?? 0) }),
    })
  }
  // The other metric, now, for the same scope.
  const mb = i.scope === 'mailbox_provider' ? s.mailboxes.find((m) => m.key === i.mailbox) : undefined
  const pv = i.scope === 'route' ? s.providers.find((p) => p.id === i.providerId) : undefined
  const bounceLimit = s.thresholds.find((x) => x.key === 'bounce')?.limit ?? 2
  const complaintLimit = s.thresholds.find((x) => x.key === (i.scope === 'mailbox_provider' ? 'mailbox_complaint' : 'complaint'))?.limit ?? 0.1
  if (i.metric === 'bounce') {
    const v = mb?.complaintRate ?? pv?.complaintRate ?? share(t.complained, t.delivered)
    rows.push({ label: metricLabel('complaint', who), value: tt(mm.ofLimit, { value: pct2(v ?? 0), limit: pct2(complaintLimit) }) })
  } else {
    const v = mb?.bounceRate ?? pv?.bounceRate ?? share(t.bounced, s.sent)
    rows.push({ label: metricLabel('bounce', who), value: tt(mm.ofLimit, { value: pct1(v ?? 0), limit: pct1(bounceLimit) }) })
  }
  const evidence = trippedEvidence(i, s)
  return {
    head,
    bar: { width: usageWidth(i.value ?? 0, i.limit ?? 0) || (i.value ? 100 : 0) },
    evidence: evidence ? { state: tt(mm.thresholdState, { state: 'tripped' }), text: evidence } : undefined,
    rows,
    // Complaints keep arriving after the send (review GS-6): the watch after it.
    watch: i.metric === 'complaint' ? tt(mm.factWatch, { hours: s.watch?.hours ?? 72 }) : undefined,
  }
}

/**
 * Why a threshold could trip (review GS-6): "7 complaints in 3,980
 * delivered · enough evidence (needs 5 events and 500 delivered)", from
 * the guard's minimums for the scope.
 */
export function trippedEvidence(i: Incident, s: Snapshot): string | undefined {
  const d = i.detail
  if (i.kind !== 'threshold' || d.events === undefined || d.base === undefined) return undefined
  const key = i.metric === 'complaint' ? (i.scope === 'mailbox_provider' ? 'mailbox_complaint' : 'complaint') : i.metric
  const row = s.thresholds.find((x) => x.key === key && (!x.mailbox || x.mailbox === i.mailbox)) ?? s.thresholds.find((x) => x.key === key)
  const unit = i.metric === 'complaint' ? 'delivered' : 'sent'
  return tt(mm.factEvidence, {
    metric: i.metric,
    events: count(d.events),
    base: count(d.base),
    unit,
    minEvents: count(row?.minEvents ?? 5),
    minBase: count(row?.minBase ?? (i.scope === 'campaign' ? 1000 : 500)),
  })
}

/** "09:14:02 ECONNREFUSED mta.acme.io:443" lines of an outage. */
export function logLines(i: Incident): { key: string; text: string; bad: boolean }[] {
  return (i.detail.log ?? []).map((l, n) => ({ key: `${n}`, text: `${clockSeconds(l.at)} ${l.text}`, bad: l.bad }))
}

/** "If you move the share to SES": cost, time, quota and reputation risk. */
export function moveFacts(i: Incident): { title: string; rows: Fact[]; risk?: { tone: Tone; text: string }; note?: string } | null {
  const mv = i.detail.move
  if (!mv) return null
  const rows: Fact[] = [
    { label: tt(mm.moveCost), value: money(mv.extraCost) },
    { label: mv.rate ? tt(mm.moveTimeAt, { rate: wholePerSec(Math.round(mv.rate)) }) : tt(mm.moveTime), value: secondsText(mv.extraSeconds) },
  ]
  if (mv.quotaFree !== undefined && mv.quotaMax !== undefined)
    rows.push({ label: tt(mm.moveQuota, { provider: mv.targetShort }), value: tt(mm.moveQuotaValue, { free: count(mv.quotaFree), max: count(mv.quotaMax) }) })
  return {
    title: tt(mm.moveTitle, { provider: mv.targetShort }),
    rows,
    risk: { tone: mv.risk === 'low' ? 'ok' : 'warn', text: mv.riskText },
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

/** "Auto-paused · Outlook", or "Auto-paused" for the whole campaign. */
export function autoPausedText(i: Incident): string {
  const who = scopeName(i)
  return who ? tt(mm.badgeAutoPausedWho, { who }) : tt(mm.badgeAutoPaused)
}

export function headerBadges(s: Snapshot, incident: Incident | undefined): { tone: Tone; text: string; pulse?: boolean; live?: boolean }[] {
  if (incident?.kind === 'threshold') {
    const out: { tone: Tone; text: string; pulse?: boolean; live?: boolean }[] = [{ tone: 'bad', text: autoPausedText(incident), pulse: true }]
    if (incident.scope === 'mailbox_provider' && s.continuing.length) out.push({ tone: 'ok', text: tt(mm.badgeOthersSending) })
    if (incident.scope === 'route') {
      const others = continuingProviders(s, incident)
      if (others.length) out.push({ tone: 'ok', text: tt(mm.badgeStillSending, { providers: andList(others.map((p) => p.short)) }) })
    }
    return out
  }
  if (incident?.kind === 'provider_outage') {
    const p = s.providers.find((x) => x.id === incident.providerId)
    const out: { tone: Tone; text: string; pulse?: boolean; live?: boolean }[] = [
      { tone: 'warn', text: tt(p?.retrying ? mm.badgeRouteRetrying : mm.badgeRouteWaiting, { provider: incident.providerShort }) },
    ]
    const others = continuingProviders(s, incident)
    if (others.length) out.push({ tone: 'ok', text: tt(mm.badgeStillSending, { providers: andList(others.map((x) => x.short)) }) })
    return out
  }
  // A share the warm-up cap holds waits for tomorrow's cap while the rest keeps going (review PR-8).
  const warm = warmupHolds(s)[0]
  if (!incident && warm && (s.state === 'partial' || s.state === 'sending')) {
    const out: { tone: Tone; text: string; pulse?: boolean; live?: boolean }[] = [{ tone: 'warn', text: tt(mm.badgeWarmupHeld, { provider: warm.short }) }]
    const others = s.providers.filter((x) => !x.warmupHold && (x.routeStatus === 'sending' || x.routeStatus === 'pending') && x.remaining > 0)
    if (others.length) out.push({ tone: 'ok', text: tt(mm.badgeStillSending, { providers: andList(others.map((x) => x.short)) }) })
    return out
  }
  // A retried share sends test messages while the rest keeps going (review GS-3).
  const retrying = s.providers.find((x) => x.retrying && x.routeStatus === 'held')
  if (retrying && (s.state === 'partial' || s.state === 'sending')) {
    const out: { tone: Tone; text: string; pulse?: boolean; live?: boolean }[] = [{ tone: 'warn', text: tt(mm.badgeRouteRetrying, { provider: retrying.short }) }]
    const others = s.providers.filter((x) => x.id !== retrying.id && (x.routeStatus === 'sending' || x.routeStatus === 'pending') && x.remaining > 0)
    if (others.length) out.push({ tone: 'ok', text: tt(mm.badgeStillSending, { providers: andList(others.map((x) => x.short)) }) })
    return out
  }
  switch (s.state) {
    case 'preparing':
      return [{ tone: 'info', text: tt(mm.badgeStarting) }]
    case 'sending':
    case 'partial':
      return [{ tone: 'ok', text: tt(mm.badgeSending), live: true }]
    case 'paused':
      return [{ tone: 'warn', text: s.pausedBy ? tt(mm.badgePausedBy, { name: s.pausedBy }) : tt(mm.badgePaused) }]
    case 'scheduled':
      return [{ tone: 'info', text: tt(mm.badgeScheduled) }]
    case 'finished':
      return [{ tone: 'ok', text: tt(mm.badgeSent) }]
    case 'cancelled':
      return [{ tone: 'neutral', text: s.sent > 0 ? tt(mm.badgeCancelledPartial) : tt(mm.badgeCancelled) }]
    case 'failed':
      return [{ tone: 'bad', text: tt(mm.badgeFailed) }]
  }
  return [{ tone: 'neutral', text: tt(mm.badgeDraft) }]
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
  const c = s.canary
  if (c && (c.phase === 'sending' || c.phase === 'watching') && s.state !== 'preparing') {
    // "Starting the canary · 09:00:04", who goes first and who waits.
    return {
      title: s.startedAt ? tt(mm.startingCanaryTitle, { time: clockSeconds(s.startedAt) }) : tt(mm.startingCanaryTitleNoTime),
      body: tt(mm.startingCanary, { size: count(c.size), seeds: c.seeds, queued: count(c.queued) }),
    }
  }
  const title = s.startedAt ? tt(mm.startingTitle, { time: clockSeconds(s.startedAt) }) : tt(mm.startingTitleNoTime)
  const via = andList(s.providers.map((p) => p.short))
  if (s.state === 'preparing') {
    const building = s.recipients ? tt(mm.startingBuilding, { count: count(s.recipients) }) : tt(mm.startingBuildingNoCount)
    return { title, body: via ? `${building} ${tt(mm.startingFirstBatches, { providers: via })}` : building }
  }
  return {
    title,
    body: via ? tt(mm.startingBuilt, { count: count(s.recipients), providers: via }) : tt(mm.startingBuiltYours, { count: count(s.recipients) }),
  }
}

/** "34 minutes", "1 minute", "1 h 12 min": how long a send took. */
export function tookText(seconds: number): string {
  const n = Math.max(1, Math.round(seconds / 60))
  if (n < 60) return tt(mm.minutesLong, { n })
  return duration(n)
}

/** "Weekly Digest #142 finished at 09:34:12" and the summary under it. */
export function finishedText(s: Snapshot): { title: string; body: string } {
  const t = s.totals
  const trips = s.incidents.filter((i) => i.kind === 'threshold').length
  const outages = s.incidents.filter((i) => i.kind === 'provider_outage').length
  const v = { sent: count(s.sent), delivered: count(t.delivered), rate: pct1(share(t.delivered, s.sent)), bounced: count(t.bounced), complaints: t.complained, watching: s.watch?.active ? 'yes' : 'no' }
  const parts = [
    s.durationSeconds !== undefined ? tt(mm.finishedIn, { ...v, took: tookText(s.durationSeconds) }) : tt(mm.finishedNoTime, v),
    tt(mm.finishedTrips, { n: trips }),
  ]
  if (outages) parts.push(tt(mm.finishedOutages, { n: outages }))
  parts.push(tt(mm.finishedReport))
  return {
    title: s.completedAt ? tt(mm.finishedAt, { name: s.name, time: clockSeconds(s.completedAt) }) : tt(mm.finished, { name: s.name }),
    body: parts.join(' '),
  }
}

export function cancelledText(s: Snapshot): { title: string; body: string } {
  const title = s.completedAt ? tt(mm.cancelledAt, { name: s.name, time: clockSeconds(s.completedAt) }) : tt(mm.cancelled, { name: s.name })
  if (s.sent === 0) return { title, body: tt(mm.cancelledNothing) }
  return { title, body: tt(mm.cancelledSome, { sent: count(s.sent), total: count(s.recipients), unsent: count(s.unsent) }) }
}

/** The manual pause banner. */
export function pausedText(s: Snapshot): { title: string; body: string } {
  const name = s.pausedBy
  const time = s.pausedAt ? clockSeconds(s.pausedAt) : ''
  const title = name ? tt(time ? mm.pausedByAt : mm.pausedBy, { name, time }) : time ? tt(mm.pausedAt, { time }) : tt(mm.paused)
  const v = { count: count(s.held + s.queued + s.waiting), perm: 'campaigns:send', reason: s.pauseReason }
  return { title, body: s.pauseReason ? tt(mm.pausedBodyReason, v) : tt(mm.pausedBody, v) }
}

/** "Lost the live event stream at 09:18:51" and what that means. */
export function lostText(at: Date, lastUpdate: string | undefined, attempt: number, max: number, stopped: boolean, now: Date): { title: string; body: string } {
  const since = lastUpdate ? new Date(lastUpdate) : at
  const behind = Math.max(1, Math.ceil((now.getTime() - since.getTime()) / 60000))
  const asOf = lastUpdate ? clockSeconds(lastUpdate) : clockSeconds(at)
  const retry = stopped ? tt(mm.lostStopped, { max }) : tt(mm.lostRetrying, { attempt: Math.max(1, attempt), max })
  return {
    title: tt(mm.lostTitle, { time: clockSeconds(at) }),
    body: `${tt(mm.lostBody, { asOf, behind })} ${retry}`,
  }
}

/** Mobile progress line: "ETA 13 min · about 09:34". */
export function mobileEta(s: Snapshot): string {
  if (s.etaSeconds === undefined) return s.state === 'paused' ? tt(mm.mobileEtaPaused) : ''
  const eta = duration(s.etaSeconds / 60)
  return s.finishesAt ? tt(mm.mobileEta, { eta, time: whenText(s.finishesAt, s.at) }) : tt(mm.timingEta, { eta })
}

/** The delivered-share legend: whole percents that add up to 100 (largest remainder). */
export function deliveredMix(s: Snapshot): { key: MailboxKey; name: string; percent: number }[] {
  const rows = s.deliveredMix.map((m) => ({ key: m.key as MailboxKey, name: m.key === 'other' ? mailboxNames.other : m.name, exact: m.percent, percent: Math.floor(m.percent) }))
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

/** Segment widths as a share of the audience; the bar's track is the rest. */
export function barSegments(s: Snapshot, incident?: Incident): (Segment & { width: number })[] {
  const base = Math.max(s.recipients, 1)
  return progressSegments(s, incident).map((seg) => ({ ...seg, width: Math.min(100, (seg.value / base) * 100) }))
}

export type LegendItem = { key: string; color: string; text: string }

/** What is still to send now (queued, and waiting for the A/B winner), outside any hold. */
export function queuedCount(s: Snapshot, incident?: Incident): number {
  return s.queued + s.waiting + (incident ? 0 : Math.max(s.held - warmupHeld(s), 0))
}

/** The routes whose share today's warm-up cap (or a paused plan) holds (review PR-8). */
export function warmupHolds(s: Snapshot): Provider[] {
  return s.providers.filter((p) => p.warmupHold)
}

/** How many recipients wait for a warm-up cap. */
export function warmupHeld(s: Snapshot): number {
  return warmupHolds(s).reduce((n, p) => n + (p.warmupHold?.held ?? 0), 0)
}

/**
 * The legend under the bar: "Amazon SES 17,635 · 14.2 /s", "Held, Outlook
 * 2,522", "Queued 12,010" (the phone layout: "SES 17,635").
 */
export function progressLegend(s: Snapshot, incident?: Incident, mobile = false): LegendItem[] {
  const items: LegendItem[] = s.providers
    .filter((p) => p.sent > 0 || p.planned > 0)
    .map((p) => ({
      key: p.id,
      color: providerColor[p.color],
      text: mobile ? `${p.short} ${count(p.sent)}` : tt(mm.legendProvider, { name: p.name, sent: count(p.sent), rate: perSec(p.rate) }),
    }))
  const warm = warmupHolds(s)
  if (!incident && warm.length && warmupHeld(s) > 0) {
    for (const p of warm) items.push({ key: `held-${p.id}`, color: heldRouteColor, text: tt(mm.legendHeldOn, { provider: p.short, count: count(p.warmupHold?.held ?? 0) }) })
  }
  if (incident && s.held > 0) {
    if (incident.kind === 'provider_outage') {
      items.push({ key: 'held', color: heldRouteColor, text: tt(mm.legendHeldOn, { provider: incident.providerShort, count: count(s.held) }) })
    } else {
      const who = scopeName(incident)
      items.push({ key: 'held', color: heldMailboxColor, text: who ? tt(mm.legendHeldWho, { who, count: count(s.held) }) : tt(mm.legendHeld, { count: count(s.held) }) })
    }
  }
  const queued = queuedCount(s, incident)
  if (queued > 0) {
    const others = incident?.kind === 'provider_outage' ? continuingProviders(s, incident) : []
    items.push({
      key: 'queued',
      color: queuedColor,
      text: others.length === 1 ? tt(mm.legendQueuedOn, { provider: others[0]!.short, count: count(queued) }) : tt(mm.legendQueued, { count: count(queued) }),
    })
  }
  return items
}

/** The phone layout's tiles: shorter captions, human opens first. */
export function mobileTiles(s: Snapshot): Tile[] {
  const t = s.totals
  return [
    countTile('delivered', mm.tileDelivered, t.delivered, tt(mm.mobileDelivered, { rate: pct1(share(t.delivered, s.sent) ?? 0), retrying: s.retrying })),
    countTile('bounced', mm.tileBounced, t.bounced, tt(mm.bouncedBlock, { rate: rate(share(t.bounced, s.sent) ?? 0), block: count(t.blockBounced) })),
    countTile('complained', mm.tileComplained, t.complained, t.complaintBase > 0 ? tt(mm.complainedWhereReported, { rate: pct2(share(t.complained, t.complaintBase)) }) : tt(mm.notAvailable)),
    countTile('opened', mm.tileOpenedHuman, t.openedHuman, tt(mm.openedMachine, { n: count(t.openedMachine) })),
    countTile('clicked', mm.tileClicked, t.clicked, tt(mm.clickedSub, { rate: pct1(share(t.clicked, t.delivered) ?? 0), ctor: pct1(share(t.clicked, t.opened) ?? 0) })),
  ]
}

/** "14.2 /s · 99.1% delivered" under a provider's name on phones. */
export function providerLine(p: Provider): string {
  return tt(mm.providerLine, { rate: perSec(p.rate), delivered: pct1(p.deliveredRate ?? 0) })
}

/**
 * The send's open notices (review GS-11, GS-12, CO-4): what the guard did
 * without stopping the send (a slowdown, a warning, a reply rule's alert)
 * and the recipients the template failed for. Each is acknowledged, not
 * decided; complaint alerts after the send have their own banner.
 */
export function openNotices(s: Snapshot): Incident[] {
  return s.incidents.filter(
    (i) =>
      !i.resolvedAt &&
      (i.kind === 'render_error' || ((i.kind === 'threshold' || i.kind === 'reply_rule') && (i.detail.action === 'slowed' || i.detail.action === 'warned' || i.detail.action === 'alerted'))),
  )
}

/** A notice's heading and text, in the reader's language as the server wrote them. */
export function noticeText(i: Incident): { title: string; body: string; samples: string[] } {
  const cause = i.detail.causes?.[0]
  const body = [cause?.body ?? '', ...(i.detail.recommendations ?? [])].filter(Boolean).join(' ')
  const samples = i.kind === 'render_error' ? (i.detail.log ?? []).map((l) => l.text) : []
  return { title: cause?.lead || tt(mm.noticeTitle), body, samples }
}

/**
 * The header's freshness pill (LiveMonitor, LiveMonitorStates): "Live ·
 * updated 1 s ago", or "Checking every 10 s · updated 90 s ago" when the
 * stream went quiet and the monitor polls instead (review GS-14).
 */
export function freshness(s: Snapshot, now: Date, mode: 'live' | 'polling' | 'reconnecting'): { text: string; stale: boolean } {
  // Seconds up to two minutes ("updated 90 s ago", as the stale banner says).
  const secs = Math.round((now.getTime() - new Date(s.at).getTime()) / 1000)
  const ago = secs < 120 ? tt(mm.secondsAgo, { n: Math.max(1, secs) }) : liveAgo(s.at, now)
  if (mode === 'polling') return { text: tt(mm.freshPolling, { ago }), stale: true }
  if (mode === 'reconnecting') return { text: tt(mm.freshReconnecting, { ago }), stale: true }
  return { text: tt(mm.freshLive, { ago }), stale: false }
}

/**
 * Who can act, in people's words (review GS-14): "Anyone who can send can
 * pause · Linh Nguyen and Marcus Chen can resume".
 */
export function whoLine(s: Snapshot, kind: 'live' | 'canary' = 'live'): string {
  const send = s.helpers?.send ?? []
  const approve = s.helpers?.approve ?? []
  if (kind === 'canary') {
    return [tt(mm.whoCanaryStop), approve.length ? tt(mm.whoCanaryContinue, { names: andList(approve) }) : ''].filter(Boolean).join(' · ')
  }
  return [tt(mm.whoPause), send.length ? tt(mm.whoResume, { names: andList(send) }) : ''].filter(Boolean).join(' · ')
}

/** "SES done about 09:51 · Postal about 09:48": when each route is done at current rates (review GS-8). */
export function routeDoneLine(s: Snapshot): string {
  const parts = s.providers
    .filter((p) => p.doneAt && p.remaining > 0)
    .map((p, i) => tt(i === 0 ? mm.routeDoneFirst : mm.routeDoneNext, { provider: p.short, time: whenText(p.doneAt!, s.at) }))
  return parts.join(' · ')
}

/**
 * A time on the snapshot's day ("09:51"), else with its day: "tomorrow
 * 00:12" (a share waiting for tomorrow's warm-up cap), "Thu 29 Oct 00:12".
 */
export function whenText(d: string, ref: string): string {
  const day = longDate(d)
  if (day === longDate(ref)) return clock(d)
  if (day === longDate(new Date(new Date(ref).getTime() + 86_400_000))) return tt(mm.tomorrowAt, { time: clock(d) })
  return `${shortDate(d)} ${clock(d)}`
}

/** API error rates: "0.1%", "0.4%", "100%". */
export function errorRate(v: number | undefined): string {
  if (v === undefined) return '—'
  return v >= 10 ? pct(Math.round(v), 0) : pct(v, 1)
}

/** "17,635 / 28,444": sent of the provider's planned share. */
export function sentOfPlanned(p: Provider): string {
  return `${count(p.sent)} / ${count(p.planned)}`
}

/** The complaint column's heading in the decision view: "Complaint · limit 0.10%". */
export function complaintHeading(s: Snapshot): string {
  const limit = s.thresholds.find((t) => t.key === 'mailbox_complaint')?.limit ?? s.thresholds.find((t) => t.key === 'complaint')?.limit
  return limit === undefined ? tt(mm.complaintHeading) : tt(mm.complaintHeadingLimit, { limit: pct2(limit) })
}

/** "Automations and System already failed over to Postmark and SES on their own". */
export function failoverCaption(s: Snapshot): string {
  if (s.failovers.length)
    return tt(mm.failoverDone, { streams: andList(s.failovers.map((f) => streamLabel(f.stream))), providers: andList(s.failovers.map((f) => f.to)) })
  const held = s.providers.filter((p) => p.routeStatus === 'held').map((p) => p.short)
  if (!held.length) return tt(mm.whoDelivers)
  if (!s.sharedStreams.length) return tt(mm.failoverUnused, { providers: andList(held) })
  return tt(mm.failoverShared, { count: s.sharedStreams.length, streams: andList(s.sharedStreams.map(streamLabel)), providers: andList(held) })
}

/** "Delivered share: Gmail 46%, Outlook 21%, …" for the stacked bar. */
export function mixLabel(mix: { name: string; percent: number }[]): string {
  return tt(mm.mixLabel, { items: mix.map((x) => `${x.name} ${pct(x.percent, 0)}`).join(', ') })
}

/** "Scheduled for Tue 14 Oct · 09:00". */
export function scheduledLine(at: string | undefined): string {
  return at ? tt(mm.scheduledFor, { date: shortDate(at), time: clock(at) }) : tt(mm.scheduled)
}

/** A feed event's words as the server writes them, in the app language (its English otherwise). */
const eventWords: Record<string, MessageDescriptor> = {
  delivered: mm.evDelivered,
  'hard bounce': mm.evHardBounce,
  'soft bounce': mm.evSoftBounce,
  deferred: mm.evDeferred,
  complaint: mm.evComplaint,
  rejected: mm.evRejected,
  'opened · machine': mm.evOpenedMachine,
  'opened · human': mm.evOpenedHuman,
  'clicked · scanner': mm.evClickedScanner,
  clicked: mm.evClicked,
  unsubscribed: mm.evUnsubscribed,
  bounce: mm.evBounce,
  open: mm.evOpen,
  scanner: mm.evScanner,
  click: mm.evClick,
}

/** An event's word ("hard bounce") or its short form ("bounce"), translated when known. */
export function eventWord(word: string): string {
  const d = eventWords[word]
  return d ? tt(d) : word
}

/** A complaint watch row: a mailbox provider that reports complaints, against its limit (LiveMonitorStates: complete). */
export type WatchRow = { key: string; name: string; counts: string; value: string; state: { tone: Tone; text: string }; width: number; color: string }

/** The complaint watch's rows after the send (review GS-6). */
export function watchRows(s: Snapshot): WatchRow[] {
  return (s.watch?.rows ?? []).map((r) => {
    const v = r.value ?? 0
    const collecting = r.state === 'collecting'
    return {
      key: r.mailbox ?? r.key,
      name: mailboxName(r.mailbox),
      counts: tt(mm.thresholdIn, { events: count(r.events), base: count(r.base), unit: 'delivered' }),
      value: tt(mm.ofLimit, { value: pct(r.value ?? undefined, 2), limit: pct(r.limit, 2) }),
      state: { tone: stateTone[r.state], text: tt(mm.thresholdState, { state: r.state }) },
      width: usageWidth(v, r.limit),
      color: collecting ? 'var(--ink-4)' : usageColor(v, r.limit),
    }
  })
}

/** "Complaint watch · 72 h · 12 complaints so far · you'll be alerted if Outlook passes 0.10%". */
export function watchTitle(s: Snapshot): string {
  const w = s.watch
  if (!w) return ''
  const rows = [...(w.rows ?? [])].sort((a, b) => (b.value ?? 0) - (a.value ?? 0))
  const who = rows[0]?.mailbox ? mailboxName(rows[0].mailbox) : tt(mm.aMailboxProvider)
  return tt(mm.watchTitle, {
    hours: w.hours,
    complaints: s.totals.complained,
    alert: w.alert && w.active ? 'yes' : 'no',
    who,
    limit: pct(w.limit, 2),
  })
}

