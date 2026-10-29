import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import { t as tt } from '@/i18n'
import { andList, clock, clockSeconds, count, pct, trimPct } from '@/lib/format'
import { mc } from './canary.messages'
import { mailboxName, mailboxRate, pct1, pct2, tiles, type Snapshot, type Tile } from './model'
import { mm } from './model.messages'

export type Canary = Schemas['LiveCanary']
export type Gate = Schemas['CanaryGate']
export type Placement = Schemas['SeedPlacement']

/** The canary is still deciding: sending, watching, or held on a failed gate. */
export function canaryActive(c: Canary | undefined | null): c is Canary {
  return !!c && (c.phase === 'sending' || c.phase === 'watching' || c.phase === 'held')
}

/** "5%": the canary's share of the audience. */
function percentOf(c: Canary): string {
  return trimPct(c.percent)
}

/** Whole minutes left until a time (at least 0). */
export function minutesLeft(until: string | undefined, now: Date): number {
  if (!until) return 0
  return Math.max(0, Math.ceil((new Date(until).getTime() - now.getTime()) / 60000))
}

/** The canary card's heading (LiveMonitorCanary). */
export function canaryTitle(c: Canary, now: Date): string {
  const v = { size: count(c.size), percent: percentOf(c), seeds: c.seeds }
  switch (c.phase) {
    case 'sending':
      return tt(mc.titleSending, v)
    case 'held': {
      const failing = c.gates.find((g) => g.state === 'failing')
      return tt(mc.titleHeld, { gate: gateName(failing?.key ?? 'bounce') })
    }
    case 'stopped':
      return tt(mc.titleStopped, { ...v, time: clock(c.decidedAt ?? now) })
  }
  return tt(mc.titleWatching, { ...v, until: clock(c.until ?? now), left: minutesLeft(c.until, now) })
}

/** "Sent 09:00:04 to 09:05:12. 29,632 recipients are queued until the canary passes." */
export function canarySentLine(c: Canary): string {
  if (c.phase === 'stopped') return tt(mc.stoppedLine, { queued: count(c.queued) })
  if (c.phase === 'sending' || !c.sentAt) return tt(mc.sendingLine, { from: clockSeconds(c.startedAt ?? new Date()), queued: count(c.queued) })
  return tt(mc.sentLine, { from: clockSeconds(c.startedAt ?? c.sentAt), to: clockSeconds(c.sentAt), queued: count(c.queued) })
}

/** A gate's name: "Bounces", "Seed placement". */
export function gateName(key: Gate['key']): string {
  return tt(mc.gate, { key })
}

/**
 * The summary line (the screen's one polite live region): how the gates
 * stand, said again only when a gate's state, the watch or a decision
 * changes.
 */
export function canarySummary(c: Canary): string {
  if (c.phase === 'stopped') return tt(mc.sumStopped, { time: clock(c.decidedAt ?? new Date()), queued: count(c.queued) })
  const failing = c.gates.find((g) => g.state === 'failing')
  if (failing) return tt(mc.sumFailed, { gate: gateName(failing.key).toLowerCase() })
  const collecting = c.gates.filter((g) => g.state === 'collecting')
  const passing = c.gates.filter((g) => g.state === 'passing').length
  const gates = tt(mc.sumGates, {
    passing,
    total: c.gates.length,
    collecting: collecting.length,
    names: andList(collecting.map((g) => gateName(g.key).toLowerCase())),
    // "seed placement is", "complaints are".
    singular: collecting.length === 1 && collecting[0]?.key === 'seeds' ? 'yes' : 'no',
  })
  return c.extended > 0 && c.until ? `${tt(mc.sumWatchingUntil, { until: clock(c.until) })} ${gates}` : gates
}

/** "203.0.113.9 on Postal and 203.0.113.10 on Postal": new sending IPs (review GS-10). */
export function newIPsText(ips: { ip: string; provider: string }[]): string {
  return andList(ips.map((x) => tt(mc.ipOn, { ip: x.ip, provider: x.provider })))
}

/** Why the canary runs, in a sentence or two. */
export function canaryWhy(c: Canary): string {
  const parts: string[] = []
  if (c.chosen) parts.push(tt(mc.whyChosen))
  for (const r of c.reasons) {
    if (r === 'never_mailed' && c.neverMailed > 0) parts.push(tt(mc.whyNeverMailed, { count: c.neverMailed, domain: c.domain }))
    if (r === 'new_domain') parts.push(tt(mc.whyNewDomain, { domain: c.domain }))
    if (r === 'new_ip' && c.newIps.length) parts.push(tt(mc.whyNewIP, { count: c.newIps.length, ips: newIPsText(c.newIps) }))
    if (r === 'over_usual') parts.push(tt(mc.whyOverUsual, { audience: count(c.size && c.percent ? Math.round((c.size * 100) / c.percent) : 0), usual: count(c.usual) }))
  }
  return parts.join(' ')
}

export type GateRow = { key: string; name: string; value: string; note?: string; bar?: number; limit: string; state: { tone: Tone; text: string } }

const gateTone: Record<Gate['state'], Tone> = { passing: 'ok', failing: 'bad', collecting: 'neutral' }

/** A gate's line: its rate with the counts, its limit and its state. */
export function gateRow(g: Gate, s: Snapshot): GateRow {
  const state = { tone: gateTone[g.state], text: tt(mc.gateState, { state: g.state }) }
  const base = { key: g.key, name: gateName(g.key), state }
  switch (g.key) {
    case 'complaint': {
      const reported = andList((g.reported ?? []).map(mailboxName))
      const unreported = (g.notReported ?? []).map(mailboxName)
      const value = tt(mc.complaintValue, { unreported: unreported.length ? andList(unreported) : 'none', count: count(g.events), reported: reported || '—' })
      const collecting = g.state === 'collecting' && g.mailbox && g.base < g.minBase
      return {
        ...base,
        value,
        note: collecting ? tt(mc.complaintNeeds, { base: count(g.base), min: count(g.minBase), mailbox: mailboxName(g.mailbox) }) : undefined,
        bar: collecting ? Math.min(100, (g.base / Math.max(g.minBase, 1)) * 100) : undefined,
        limit: tt(mc.limitPerMailbox, { limit: pct(g.limit, 2) }),
      }
    }
    case 'seeds': {
      const p = g.seeds
      const value = p
        ? [
            tt(mc.seedValue, { inbox: p.inbox, promotions: p.promotions, spam: p.spam }),
            p.pending ? tt(mc.seedPending, { pending: p.pending }) : '',
            p.missing ? tt(mc.seedMissing, { missing: p.missing }) : '',
          ]
            .filter(Boolean)
            .join(' · ')
        : tt(mc.seedPending, { pending: g.base })
      return { ...base, value, limit: tt(mc.limitSeeds, { limit: g.limit }) }
    }
  }
  const value = g.events === 0 ? count(0) : tt(mc.valueOf, { rate: pct(g.value ?? 0, g.key === 'block_bounce' ? 2 : 1), events: count(g.events), base: count(g.base) })
  void s
  return { ...base, value, limit: tt(mc.limit, { limit: g.key === 'deferral' ? trimPct(g.limit) : pct(g.limit, g.key === 'block_bounce' ? 1 : 1) }) }
}

/** "4.0% · 46 since delivered, 16 retrying": what became of the deferred. */
function deferredSub(deferred: number, retrying: number, rate: number): string {
  const since = Math.max(0, deferred - retrying)
  const detail = [since > 0 ? tt(mc.sinceDelivered, { count: count(since) }) : '', retrying > 0 ? tt(mc.retryingCount, { count: count(retrying) }) : ''].filter(Boolean)
  return detail.length ? `${pct(rate, 1)} · ${detail.join(', ')}` : tt(mc.deferredSub, { rate: pct(rate, 1) })
}

/** When everyone has it if the main send starts at a time. */
export function etaFrom(c: Canary, start: Date | string): string {
  const at = new Date(typeof start === 'string' ? start : start.getTime()).getTime() + (c.mainSeconds ?? 0) * 1000
  return clock(new Date(at))
}

/** Ten minutes after the watch's end (Watch 10 more minutes). */
export function untilPlus(c: Canary, now: Date): Date {
  const base = c.until ? new Date(c.until) : now
  return new Date(Math.max(base.getTime(), now.getTime()) + 10 * 60000)
}

/** The seed placement's rows per mailbox provider. */
export function seedRows(p: Placement): { key: string; name: string; note: string; value: string }[] {
  return p.byMailbox.map((r) => ({
    key: r.mailbox,
    name: mailboxName(r.mailbox),
    note: [r.mailbox === 'icloud' ? '' : tt(mc.seedsIncl, { mailbox: r.mailbox }), tt(mc.seedsOf, { count: r.seeds })].filter(Boolean).join(' · '),
    value: tt(mc.seedValue, { inbox: r.inbox, promotions: r.mailbox === 'gmail' || r.promotions > 0 ? r.promotions : 0, spam: r.spam }),
  }))
}

/** The canary's progress: what went, per sending provider, and what waits. */
export function canaryProgress(
  c: Canary,
  s: Snapshot,
  now: Date,
): { line: string; sent: number; parts: { key: string; text: string; value: number; color: string }[]; queued: string } {
  const sentPct = s.recipients > 0 ? (s.sent / s.recipients) * 100 : 0
  const v = { sent: count(s.sent), total: count(s.recipients), percent: pct(sentPct, 1) }
  const line =
    c.phase === 'stopped' || !c.until ? tt(mc.progressLineNow, v) : tt(mc.progressLine, { ...v, until: clock(c.until), eta: etaFrom(c, c.until ?? now) })
  const parts = s.providers
    .filter((p) => p.sent > 0)
    .map((p) => ({ key: p.id, text: tt(mc.viaProvider, { provider: p.name, count: count(p.sent) }), value: p.sent, color: `var(--chart-${p.color})` }))
  // "Queued 29,632 · SES 26,669 · Postal 2,963".
  const routes = s.providers.filter((p) => p.remaining > 0)
  const queued = tt(mc.queuedLegend, {
    count: count(c.queued),
    routes: routes.length > 1 ? routes.map((p) => `${p.short} ${count(p.remaining)}`).join(' · ') : 'none',
  })
  return { line, sent: s.sent, parts, queued }
}

/** The canary's mailbox provider rows (LiveMonitorCanary): what it did, the main send's rate, whether thresholds can trip yet. */
export function canaryMailboxRows(s: Snapshot) {
  const min = canaryMinBase(s)
  return s.mailboxes
    .filter((mb) => mb.sent > 0 || mb.remaining > 0)
    .map((mb) => ({
      key: mb.key,
      name: mailboxName(mb.key),
      sent: count(mb.sent),
      delivered: mb.retrying > 0 ? tt(mc.deliveredRetrying, { rate: pct1(mb.deliveredRate), retrying: count(mb.retrying) }) : pct1(mb.deliveredRate),
      deferred: pct1(mb.sent > 0 ? (mb.deferred / mb.sent) * 100 : undefined),
      bounce: pct1(mb.bounceRate),
      na: Boolean(mb.complaintReason),
      complaint: mb.complaintReason ? tt(mm.notAvailable) : mb.complained === 0 ? tt(mc.noneSoFar) : pct2(mb.complaintRate),
      rate: mailboxRate(mb),
      state:
        mb.delivered >= min
          ? { tone: 'ok' as Tone, text: tt(mm.thresholdState, { state: 'ok' }) }
          : { tone: 'neutral' as Tone, text: tt(mm.thresholdState, { state: 'collecting' }) },
    }))
}

/** What a mailbox provider needs delivered before its thresholds can trip (500). */
export function canaryMinBase(s: Snapshot): number {
  return s.thresholds.find((x) => x.key === 'mailbox_complaint')?.minBase || 500
}

/** "A mailbox provider's thresholds need 500 delivered to it before they can trip, except …; in the canary only Gmail has that many." */
export function canaryMinNote(s: Snapshot): string {
  const min = canaryMinBase(s)
  const enough = s.mailboxes.filter((mb) => mb.delivered >= min).map((mb) => mailboxName(mb.key))
  return tt(mc.minNote, { min: count(min), count: enough.length, names: andList(enough) })
}

/** The badge after the breadcrumb, and the live view's line once it passed. */
export function canaryPassedLine(c: Canary): string | undefined {
  if (c.phase !== 'passed' && c.phase !== 'continued') return undefined
  return tt(c.phase === 'passed' ? mc.passed : mc.passedEarly, { percent: percentOf(c), time: clock(c.decidedAt ?? new Date()) })
}

/**
 * The canary's counters (LiveMonitorCanary): of the canary, with deferrals
 * on their own tile; complaints only where they're reported.
 */
export function canaryTiles(s: Snapshot): Tile[] {
  const t = s.totals
  const [delivered, bounced, complained, opened] = [...tiles(s)].filter((x) => x.key !== 'clicked')
  const reported = s.mailboxes.filter((m) => !m.complaintReason && m.sent > 0).map((m) => mailboxName(m.key))
  const unreported = t.notReported.map(mailboxName)
  const deferredRate = s.sent > 0 ? (t.deferred / s.sent) * 100 : 0
  return [
    { ...delivered!, sub: tt(mc.deliveredSub, { rate: pct(s.sent > 0 ? (t.delivered / s.sent) * 100 : 0, 1), retrying: s.retrying }) },
    bounced!,
    { key: 'deferred', label: tt(mc.tileDeferred), value: count(t.deferred), num: t.deferred, sub: deferredSub(t.deferred, s.retrying, deferredRate) },
    {
      ...complained!,
      sub: tt(mc.complainedSub, { reported: andList(reported) || tt(mm.notAvailable), unreported: unreported.length ? andList(unreported) : 'none' }),
    },
    opened!,
  ]
}
