import type { Schemas } from '@/api/client'
import { formatList, t as tt } from '@/i18n'
import { ago, dayMonthDate, duration } from '@/lib/format'
import { mailboxName } from '@/features/live/model'
import { m } from './traffic.messages'

export type Rules = Schemas['TrafficRules']
export type Form = Schemas['TrafficRulesInput']
export type ReplyRule = Schemas['TrafficReplyRule']
export type Thresholds = Schemas['TrafficThresholds']
export type ThresholdAction = Thresholds['blockAction']

/** The percent fields people type, with their bounds (the API's). */
export const thresholdBounds = {
  blockBounce: [0.01, 20],
  unknownWarn: [0.1, 50],
  unknownPause: [0.1, 50],
  deferralSlow: [1, 100],
  deferral: [1, 100],
  complaint: [0.01, 5],
} as const satisfies Partial<Record<keyof Thresholds, readonly [number, number]>>

export const gateBounds = {
  bounce: [0.1, 20],
  blockBounce: [0.01, 20],
  deferral: [1, 100],
  complaint: [0.01, 5],
} as const

export type ThresholdField = keyof typeof thresholdBounds
export type GateField = keyof typeof gateBounds | 'seedSpam' | 'percent'

/** What each threshold's action can be, in the board's order. */
export const blockActions: ThresholdAction[] = ['pause_mailbox', 'slow', 'pause_send', 'warn']
export const unknownWarnActions: Thresholds['unknownWarnAction'][] = ['warn', 'slow', 'pause_send']
export const unknownPauseActions: Thresholds['unknownPauseAction'][] = ['pause_send', 'warn', 'slow']
export const deferralSlowActions: Thresholds['deferralSlowAction'][] = ['slow', 'pause_mailbox', 'warn']
export const deferralActions: Thresholds['deferralAction'][] = ['pause_mailbox', 'slow', 'warn']
export const complaintActions: Thresholds['complaintAction'][] = ['pause_mailbox', 'pause_send', 'warn']
export const windows = [10, 5, 15, 30] as const
export const watchMinutes = [20, 10, 30, 60] as const
export const watchHours = [72, 24, 48, 168] as const

/** The add-rule form's presets ("10 replies in 10 min"). */
export const whenCounts = [1, 5, 10, 25] as const
export const replyActions: ReplyRule['action'][] = ['slow', 'pause', 'pause_route', 'alert']
export const durations = [30, 60, 120, 360, 0] as const
export const mailboxKeys: ReplyRule['mailbox'][] = ['gmail', 'outlook', 'yahoo', 'icloud', 'other']

/** The form the screen edits: what it saves. */
export function toForm(r: Rules): Form {
  return { thresholds: { ...r.thresholds }, replyRules: r.replyRules.map((x) => ({ ...x })), canary: { ...r.canary, gates: { ...r.canary.gates } }, watch: { ...r.watch } }
}

/** A number as people type it: "0.5", "0.10" (complaints keep two decimals). */
export function numberText(v: number, digits?: number): string {
  if (digits !== undefined) return v.toFixed(digits)
  return String(Number(v.toFixed(2)))
}

/** A typed number: commas or points, spaces ignored; undefined when it isn't one. */
export function parseNumber(s: string): number | undefined {
  const v = Number(s.trim().replace(/\s/g, '').replace(',', '.'))
  return s.trim() === '' || !Number.isFinite(v) ? undefined : v
}

/** Whether a typed value is in its bounds. */
export function inBounds(s: string, [min, max]: readonly [number, number]): boolean {
  const v = parseNumber(s)
  return v !== undefined && v >= min && v <= max
}

/** "On the first reply", "After 10 replies in 10 min". */
export function ruleWhen(r: Pick<ReplyRule, 'count' | 'windowMinutes'>): string {
  return tt(m.ruleWhen, { count: r.count, window: r.windowMinutes })
}

/** "Pause Yahoo", "Slow Gmail to 50%", "Alert only". */
export function ruleAction(r: Pick<ReplyRule, 'action' | 'mailbox'>): string {
  return tt(m.ruleAction, { action: r.action, mailbox: mailboxName(r.mailbox) })
}

/** "for 2 h", "until someone resumes". */
export function ruleDuration(r: Pick<ReplyRule, 'durationMinutes'>): string {
  return tt(m.ruleDuration, { kind: r.durationMinutes > 0 ? 'for' : 'until', duration: r.durationMinutes > 0 ? duration(r.durationMinutes) : '' })
}

/** "23 Sep · Weekly Digest #139", or "never". */
export function lastMatched(r: Pick<ReplyRule, 'lastMatchedAt' | 'lastCampaign'>): string {
  if (!r.lastMatchedAt) return tt(m.never)
  const date = dayMonthDate(r.lastMatchedAt)
  return r.lastCampaign ? tt(m.lastMatchedIn, { date, campaign: r.lastCampaign }) : date
}

/** "3 of 3 on". */
export function rulesOn(rules: Pick<ReplyRule, 'enabled'>[]): string {
  return tt(m.rulesOn, { on: rules.filter((r) => r.enabled).length, total: rules.length })
}

/** The save line: unsaved changes, or who saved last and when. */
export function savedText(r: Pick<Rules, 'savedAt' | 'savedBy'>, dirty: boolean, now: Date = new Date()): string {
  if (dirty) return tt(m.unsaved)
  if (!r.savedAt) return tt(m.defaults)
  return r.savedBy ? tt(m.savedBy, { ago: ago(r.savedAt, now), name: r.savedBy }) : tt(m.savedAgo, { ago: ago(r.savedAt, now) })
}

/** "Linh Nguyen and Marcus Chen". */
export function peopleList(names: string[]): string {
  return formatList(names)
}

/** A canary is on when any reason runs it. */
export function canaryOn(c: Form['canary']): boolean {
  return c.newDomain || c.neverMailed || c.overUsual
}

/** About how much later a send with a canary finishes: the watch, plus a few minutes of sending. */
export function canaryDelay(c: Form['canary']): number {
  return c.watchMinutes + 5
}
