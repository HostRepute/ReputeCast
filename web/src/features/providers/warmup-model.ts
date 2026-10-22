import type { Schemas } from '@/api/client'
import { t, type MessageDescriptor } from '@/i18n'
import { count, trimPct } from '@/lib/format'
import { mw } from './warmup.messages'

export type WarmupDay = Schemas['WarmupDay']
export type WarmupPlan = Schemas['WarmupPlanView']

/**
 * Bar height (percent of the chart) for a day's volume. The ramp grows about
 * 240× from day 1 to the last day, so the chart uses a compressed scale
 * (value^0.72) to keep the first days visible; exact numbers are in the
 * table and the chart's label.
 */
export function rampHeight(value: number, max: number): number {
  if (max <= 0 || value <= 0) return 0
  return Math.max(1, Math.round(Math.pow(Math.min(1, value / max), 0.72) * 100))
}

/** Rows the table shows: the last four finished days, today, the next three and the last day. */
export function visibleDays(schedule: WarmupDay[]): WarmupDay[] {
  const today = schedule.findIndex((d) => d.status === 'today')
  if (today < 0) return schedule
  const keep = new Set<number>()
  for (let i = Math.max(0, today - 4); i <= Math.min(schedule.length - 1, today + 3); i++) keep.add(i)
  keep.add(schedule.length - 1)
  return schedule.filter((_, i) => keep.has(i))
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
export function audienceText(d: WarmupDay): string {
  const parts = [tiersText(d.tiers)]
  for (const u of d.uses) parts.push(t(mw.dayUses, { name: u.name, count: count(u.count) }))
  if (d.eligible) parts.push(t(mw.dayEligible, { share: trimPct(d.share) }))
  else if (d.shareChange) parts.push(t(mw.dayShareRises, { share: trimPct(d.share) }))
  return parts.join(' · ')
}

export const dayStatus: Record<WarmupDay['status'], { tone: 'ok' | 'warn' | 'info' | 'neutral'; text: (guard: number) => string }> = {
  done: { tone: 'ok', text: () => t(mw.statusDone) },
  slowdown: { tone: 'warn', text: (g) => t(mw.statusSlowdown, { limit: trimPct(g) }) },
  resumed: { tone: 'ok', text: () => t(mw.statusResumed) },
  today: { tone: 'info', text: () => t(mw.statusToday) },
  planned: { tone: 'neutral', text: () => t(mw.statusPlanned) },
}

/**
 * A percentage typed in a guard field: "2.0 %", "2,0 %" or "2%" (the field
 * shows the region's decimal mark); undefined when it isn't a number.
 */
export function guardIn(s: string): number | undefined {
  const v = s.replace('%', '').replace(',', '.').trim()
  if (!/^\d*\.?\d+$/.test(v)) return undefined
  const n = Number(v)
  return Number.isFinite(n) ? n : undefined
}
