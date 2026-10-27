import type { Schemas } from '@/api/client'
import { t, type MessageDescriptor } from '@/i18n'
import { count, trimPct } from '@/lib/format'
import { mw } from './warmup.messages'

export type WarmupPlan = Schemas['WarmupPlanView']
export type WarmupRow = Schemas['WarmupRow']
export type WarmupCapHold = Schemas['WarmupCapHold']

/**
 * A bar's height (percent of the chart) for a day's volume, drawn to scale
 * as on the board; a day that sent anything stays visible.
 */
export function rampHeight(value: number, max: number): number {
  if (max <= 0 || value <= 0) return 0
  return Math.max(0.4, Math.round((Math.min(1, value / max) * 1000)) / 10)
}

/** One column of the ramp chart: a plan day. */
export interface RampColumn {
  day: number
  /** Heights in percent: what went out, and the schedule's cap above it (or alone, for days ahead). */
  sent: number
  cap: number
  tripped: boolean
  today: boolean
}

/**
 * The ramp chart's columns: days that went out show what they sent (a day
 * that tripped a guard in its own colour, even once it counted on a later
 * date) with the part of the schedule a held cap kept back above it; today
 * and the days ahead show their planned cap.
 */
export function rampColumns(plan: Pick<WarmupPlan, 'ramp' | 'rows'>): RampColumn[] {
  const max = Math.max(...plan.ramp.map((d) => d.plannedCap), 1)
  return plan.ramp.map((d) => {
    const judged = plan.rows.filter((r) => r.day === d.day && (r.status === 'counted' || r.status === 'not_counted'))
    const held = judged.filter((r) => r.held && r.plannedCap && r.plannedCap > r.cap).at(-1)
    if (d.today || d.sent === undefined || judged.length === 0) {
      return { day: d.day, sent: 0, cap: rampHeight(d.plannedCap, max), tripped: false, today: d.today }
    }
    return {
      day: d.day,
      sent: rampHeight(d.sent, max),
      cap: held ? rampHeight((held.plannedCap ?? 0) - held.cap, max) : 0,
      tripped: d.slowed,
      today: false,
    }
  })
}

/** A row of the days table: one date, or the first days folded into one. */
export type TableRow = { kind: 'day'; key: string; row: WarmupRow } | { kind: 'group'; key: string; group: DayGroup }

/** The first days of a plan, folded into one row of the table. */
export interface DayGroup {
  firstDay: number
  lastDay: number
  from: string
  to: string
  /** The caps of the days that counted, and everything sent. */
  cap: number
  sent: number
  /** Rates in percent; undefined when nothing was measured. */
  bounce?: number
  complaint?: number
  tiers: string[]
  counted: boolean
  /** The dates in the group that didn't count. */
  missed: WarmupRow[]
}

/** How many judged dates the table shows one by one; earlier plain ones fold. */
export const RECENT_DAYS = 5

const judgedStatus = (r: WarmupRow) => r.status === 'counted' || r.status === 'not_counted'

/**
 * The rows the days table shows (ProvidersWarmup): the first days folded
 * into one row while they are plain (no guard trip, no held cap), the last
 * five judged dates one by one, today, a repeat of today's day, the next
 * three days and the last one.
 */
export function tableRows(rows: WarmupRow[]): TableRow[] {
  const judged = rows.filter(judgedStatus)
  let fold = judged.slice(0, Math.max(0, judged.length - RECENT_DAYS))
  const notable = fold.findIndex((r) => r.reason === 'guard' || r.held)
  if (notable >= 0) fold = fold.slice(0, notable)
  // A folded row ends on a day that counted; the repeat that follows it
  // stays beside it in the table.
  while (fold.length && fold[fold.length - 1]!.status !== 'counted') fold = fold.slice(0, -1)
  const out: TableRow[] = []
  if (fold.length >= 2) out.push({ kind: 'group', key: `g${fold[0]!.date}`, group: groupDays(fold) })
  else fold = []
  for (const r of judged.slice(fold.length)) out.push({ kind: 'day', key: r.date, row: r })
  for (const r of rows) if (r.status === 'today') out.push({ kind: 'day', key: r.date, row: r })
  const planned = rows.filter((r) => r.status === 'planned')
  let ahead = 0
  planned.forEach((r, i) => {
    const last = i === planned.length - 1
    if (r.repeat || last || ahead < 3) {
      out.push({ kind: 'day', key: r.date, row: r })
      if (!r.repeat && !last) ahead++
    }
  })
  return out
}

/** Adds up the folded first days. */
export function groupDays(rows: WarmupRow[]): DayGroup {
  let cap = 0
  let sent = 0
  let bounced = 0
  let complained = 0
  let delivered = 0
  const tiers = new Set<string>()
  for (const r of rows) {
    if (r.status === 'counted') cap += r.cap
    sent += r.sent ?? 0
    bounced += r.bounced
    complained += r.complained
    delivered += r.delivered
    r.tiers.forEach((x) => tiers.add(x))
  }
  const days = new Set(rows.map((r) => r.day))
  const counted = [...days].every((d) => rows.some((r) => r.day === d && r.status === 'counted'))
  return {
    firstDay: rows[0]!.day,
    lastDay: rows[rows.length - 1]!.day,
    from: rows[0]!.date,
    to: rows[rows.length - 1]!.date,
    cap,
    sent,
    bounce: sent > 0 ? Math.round((bounced / sent) * 1000) / 10 : undefined,
    complaint: delivered > 0 ? Math.round((complained / delivered) * 10000) / 100 : undefined,
    tiers: tierOrder.filter((x) => tiers.has(x)),
    counted,
    missed: rows.filter((r) => r.status !== 'counted'),
  }
}

/** The plan's date of the day after a date ("2026-10-06T00:00:00+07:00" → 6 Oct's next day). */
export function nextDay(date: string): Date {
  return new Date(new Date(date).getTime() + 24 * 3600 * 1000)
}

const tierOrder = ['active', 'cooling', 'at_risk', 'sunset']
const tierName: Record<string, MessageDescriptor> = { active: mw.tierActive, cooling: mw.tierCooling, at_risk: mw.tierAtRisk, sunset: mw.tierSunset }

/** A tier's name in the app language; an unknown key as it is. */
function tierText(key: string): string {
  const d = tierName[key]
  return d ? t(d) : key
}

/** "Active tier", "Active + Cooling", "All tiers except Sunset", "All". */
export function tiersText(tiers: string[]): string {
  const set = new Set(tiers)
  if (tierOrder.every((x) => set.has(x))) return t(mw.tiersAll)
  if (tiers.length === 1) return t(mw.tierOne, { tier: tierText(tiers[0]!) })
  if (tierOrder.slice(0, 3).every((x) => set.has(x)) && !set.has('sunset')) return t(mw.tiersExceptSunset)
  return tierOrder
    .filter((x) => set.has(x))
    .map(tierText)
    .join(' + ')
}

/** The audience column: tiers, campaigns using the day, share changes. */
export function audienceText(d: Pick<WarmupRow, 'tiers' | 'uses' | 'eligible' | 'shareChange' | 'share'>): string {
  const parts = [tiersText(d.tiers)]
  for (const u of d.uses) parts.push(t(mw.dayUses, { name: u.name, count: count(u.count) }))
  if (d.eligible) parts.push(t(mw.dayEligible, { share: trimPct(d.share) }))
  else if (d.shareChange) parts.push(t(mw.dayShareRises, { share: trimPct(d.share) }))
  return parts.join(' · ')
}

/** Mailbox provider colours (chart palette, fixed order). */
export const mailboxColor: Record<string, string> = {
  gmail: 'var(--chart-gmail)',
  outlook: 'var(--chart-outlook)',
  yahoo: 'var(--chart-yahoo)',
  icloud: 'var(--chart-icloud)',
  other: 'var(--chart-other)',
}

/**
 * A percentage typed in a guard field: "2.0 %", "2,0 %" or "2%" (the field
 * shows the region's decimal mark); undefined when it isn't a number.
 */
export function guardIn(s: string): number | undefined {
  const v = s.replaceAll('%', '').replace(',', '.').trim()
  if (!/^\d*\.?\d+$/.test(v)) return undefined
  const n = Number(v)
  return Number.isFinite(n) ? n : undefined
}
