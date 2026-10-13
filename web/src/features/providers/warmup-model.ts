import type { Schemas } from '@/api/client'
import { count } from '@/lib/format'

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
const tierName: Record<string, string> = { active: 'Active', cooling: 'Cooling', at_risk: 'At risk', sunset: 'Sunset' }

/** "Active tier", "Active + Cooling", "All tiers except Sunset", "All". */
export function tiersText(tiers: string[]): string {
  const set = new Set(tiers)
  if (tierOrder.every((t) => set.has(t))) return 'All'
  if (tiers.length === 1) return `${tierName[tiers[0]!] ?? tiers[0]} tier`
  if (tierOrder.slice(0, 3).every((t) => set.has(t)) && !set.has('sunset')) return 'All tiers except Sunset'
  return tierOrder.filter((t) => set.has(t)).map((t) => tierName[t]).join(' + ')
}

/** The audience column: tiers, campaigns using the day, share changes. */
export function audienceText(d: WarmupDay): string {
  let s = tiersText(d.tiers)
  for (const u of d.uses) s += ` · ${u.name} uses ${count(u.count)}`
  if (d.eligible) s += ` · share ${d.share}%, eligible as failover`
  else if (d.shareChange) s += ` · share rises to ${d.share}%`
  return s
}

export const dayStatus: Record<WarmupDay['status'], { tone: 'ok' | 'warn' | 'info' | 'neutral'; text: (guard: number) => string }> = {
  done: { tone: 'ok', text: () => 'Done' },
  slowdown: { tone: 'warn', text: (g) => `Slowdown · bounces over ${Number(g.toFixed(2))}%` },
  resumed: { tone: 'ok', text: () => 'Done · ramp resumed' },
  today: { tone: 'info', text: () => 'Today' },
  planned: { tone: 'neutral', text: () => 'Planned' },
}
