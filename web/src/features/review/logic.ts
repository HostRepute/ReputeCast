import type { Schemas } from '@/api/client'
import { count } from '@/lib/format'

export type Review = Schemas['CampaignReview']
export type Run = Schemas['ReadinessRun']
export type Check = Schemas['ReadinessCheck']
export type Fix = Schemas['ReadinessFix']
export type Group = Schemas['ReadinessGroup']
export type Counts = Schemas['ReadinessCounts']
export type Schedule = Schemas['ReadinessSchedule']
export type Thresholds = Schemas['ReadinessThresholds']
export type Verdict = Run['verdict']
export type CheckStatus = Check['status']

export function plural(n: number, one: string, many: string): string {
  return n === 1 ? one : many
}

/** "Ready", "Ready with warnings", "Blocked". */
export function verdictTitle(v: Verdict): string {
  return v === 'blocked' ? 'Blocked' : v === 'warnings' ? 'Ready with warnings' : 'Ready'
}

/** The badge word of a check row. */
export function statusWord(s: CheckStatus): string {
  return s === 'pass' ? 'Pass' : s === 'warn' ? 'Warn' : s === 'block' ? 'Block' : 'Unknown'
}

/** "27 checks passed, 2 warnings, 0 blocks". */
export function countsLine(c: Counts): string {
  return `${c.passed} ${plural(c.passed, 'check', 'checks')} passed, ${c.warnings} ${plural(c.warnings, 'warning', 'warnings')}, ${c.blocks} ${plural(c.blocks, 'block', 'blocks')}`
}

/** "25 passed · 2 warnings · 0 blocks" (the approval cards). */
export function countsShort(c: { passed: number; warnings: number; blocks: number }): string {
  return `${c.passed} passed · ${c.warnings} ${plural(c.warnings, 'warning', 'warnings')} · ${c.blocks} ${plural(c.blocks, 'block', 'blocks')}`
}

/**
 * The Blocked banner's sentence: "3 blocking issues, 2 warnings, 24 checks
 * passed. Sending stays disabled until every block is fixed. Each one has a
 * one-click fix below."
 */
export function blockedLine(c: Counts, checks: Check[]): string {
  const blocks = checks.filter((x) => x.status === 'block')
  const fixable = blocks.filter((x) => primaryFix(x)).length
  const head = `${c.blocks} blocking ${plural(c.blocks, 'issue', 'issues')}, ${c.warnings} ${plural(c.warnings, 'warning', 'warnings')}, ${c.passed} ${plural(c.passed, 'check', 'checks')} passed.`
  const tail =
    fixable === 0
      ? ''
      : fixable === blocks.length
        ? blocks.length === 1
          ? ' It has a one-click fix below.'
          : ' Each one has a one-click fix below.'
        : ` ${fixable} of them ${plural(fixable, 'has', 'have')} a one-click fix below.`
  return `${head} Sending stays disabled until every block is fixed.${tail}`
}

/** The one-click fix a "fix all" button applies for a check: its first apply fix. */
export function primaryFix(c: Check): Fix | undefined {
  return c.fixes.find((f) => f.kind === 'apply')
}

/** "Fix both warnings", "Apply all 3 fixes". */
export function fixAllLabel(kind: 'warn' | 'block', n: number): string {
  if (kind === 'block') return n === 1 ? 'Apply the fix' : n === 2 ? 'Apply both fixes' : `Apply all ${n} fixes`
  return n === 1 ? 'Fix the warning' : n === 2 ? 'Fix both warnings' : `Fix all ${n} warnings`
}

/** "Schedule send to 31,604 recipients" or "Send to 47,842 recipients". */
export function sendLabel(mode: Schedule['mode'], recipients: number): string {
  return `${mode === 'now' ? 'Send' : 'Schedule send'} to ${count(recipients)} ${plural(recipients, 'recipient', 'recipients')}`
}

const smallWords = ['', 'one', 'two', 'three']

/** "You'll confirm the audience and the two warnings on the next step." */
export function confirmNote(warnings: number): string {
  if (warnings === 0) return "You'll confirm the audience on the next step."
  if (warnings === 1) return "You'll confirm the audience and the warning on the next step."
  return `You'll confirm the audience and the ${smallWords[warnings] ?? warnings} warnings on the next step.`
}

/** When the next check runs, for the verdict banner. */
export function recheckText(mode: Schedule['mode']): string {
  return mode === 'now' ? 'and will run again when the send starts' : 'and will run again 15 minutes before the scheduled send'
}

const weekdays = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

/** "Wednesday 14 Oct 2026" from a local "2026-10-14" (no timezone shift). */
export function localDateWords(date: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date)
  if (!m) return date
  const d = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])))
  return `${weekdays[d.getUTCDay()]} ${d.getUTCDate()} ${months[d.getUTCMonth()]} ${d.getUTCFullYear()}`
}

/** "Nov 4" from a local "2026-11-04". */
export function localMonthDay(date: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date)
  if (!m) return date
  return `${months[Number(m[2]) - 1]} ${Number(m[3])}`
}

/** "Tuesday 14 Oct 2026 · 09:00 · America/New_York" (Scheduling card). */
export function scheduleCaption(s: Schedule): string {
  if (!s.date || !s.time) return 'Pick a date and a time'
  return `${localDateWords(s.date)} · ${s.time} · ${s.timezone ?? ''}`.replace(/ · $/, '')
}

/** "Tuesday 14 Oct 2026 at 09:00 America/New_York" (confirmation). */
export function scheduleSentence(s: Schedule): string {
  if (!s.date || !s.time) return ''
  return `${localDateWords(s.date)} at ${s.time} ${s.timezone ?? ''}`.trim()
}

/** "Nov 4, 09:00" (the approved card's schedule button). */
export function scheduleShort(s: Schedule): string {
  if (s.mode === 'now' || !s.date || !s.time) return 'now'
  return `${localMonthDay(s.date)}, ${s.time}`
}

/** "1 d 22 h 19 m" until a time; "now" once it passed. */
export function sendsIn(at: string | number | Date, now: Date = new Date()): string {
  const total = Math.max(0, Math.floor((new Date(at).getTime() - now.getTime()) / 60_000))
  if (total <= 0) return 'now'
  const d = Math.floor(total / 1440)
  const h = Math.floor((total % 1440) / 60)
  const m = total % 60
  return [d ? `${d} d` : '', d || h ? `${h} h` : '', `${m} m`].filter(Boolean).join(' ')
}

/** "about 3 hours", "about 45 minutes" (median wait). */
export function waitWords(minutes: number): string {
  if (minutes < 60) return `about ${Math.max(1, Math.round(minutes))} ${plural(Math.round(minutes), 'minute', 'minutes')}`
  const h = Math.round(minutes / 60)
  return `about ${h} ${plural(h, 'hour', 'hours')}`
}

/** "2 h 40 min" (median wait in the inbox). */
export function waitShort(minutes: number): string {
  if (minutes < 60) return `${Math.max(1, Math.round(minutes))} min`
  const h = Math.floor(minutes / 60)
  const m = Math.round(minutes % 60)
  return m ? `${h} h ${m} min` : `${h} h`
}

/** "bounce 2.0% · complaint 0.10%". */
export function thresholdsLine(t: Thresholds): string {
  return `bounce ${t.bounce.toFixed(1)}% · complaint ${t.complaint.toFixed(2)}%`
}

/** The thresholds inputs show "2.0 %", "0.10 %", "5 %". */
export function percentText(v: number, digits: number): string {
  return `${v.toFixed(digits)} %`
}

/** Reads "2.5", "2,5 %" or "2.5%"; null when it isn't a number. */
export function parsePercent(s: string): number | null {
  const t = s.replace('%', '').replace(',', '.').trim()
  if (!/^\d*\.?\d+$/.test(t)) return null
  const v = Number(t)
  return Number.isFinite(v) ? v : null
}

export type ThresholdField = 'bounce' | 'complaint' | 'mailboxComplaint' | 'providerErrors'

/** Limits of each threshold (the API's), with the sentence when outside. */
export const thresholdLimits: Record<ThresholdField, { min: number; max: number; digits: number; label: string }> = {
  bounce: { min: 0.1, max: 20, digits: 1, label: 'Bounce rate' },
  complaint: { min: 0.01, max: 5, digits: 2, label: 'Complaint rate' },
  mailboxComplaint: { min: 0.01, max: 5, digits: 2, label: 'Complaints, per mailbox provider' },
  providerErrors: { min: 1, max: 100, digits: 0, label: 'Provider errors over 3 min' },
}

/** Validates one threshold value; returns the error sentence or undefined. */
export function thresholdError(field: ThresholdField, v: number | null, t: Thresholds): string | undefined {
  const l = thresholdLimits[field]
  if (v === null) return 'Enter a percentage, for example 2.0.'
  if (v < l.min || v > l.max) return `Keep it between ${l.min} % and ${l.max} %.`
  if (field === 'mailboxComplaint' && v > t.complaint * 5) return 'Keep it within five times the overall complaint rate.'
  if (field === 'complaint' && t.mailboxComplaint > v * 5) return 'The per-mailbox threshold would be more than five times this; lower it first.'
  return undefined
}

/** The next whole day at 09:00 in a zone, as local date and time strings. */
export function defaultScheduleAt(tz: string, now: Date = new Date()): { date: string; time: string } {
  let date: string
  try {
    const p = new Intl.DateTimeFormat('en-CA', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(now.getTime() + 86_400_000))
    date = p
  } catch {
    date = new Date(now.getTime() + 86_400_000).toISOString().slice(0, 10)
  }
  return { date, time: '09:00' }
}

/** Timezone labels as on the board: "America/New_York (ET)", "UTC". */
export function zoneLabel(zone: string): string {
  if (zone === 'UTC') return 'UTC'
  try {
    const name = new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'shortGeneric' }).formatToParts(new Date()).find((p) => p.type === 'timeZoneName')?.value
    if (name && name.length <= 5 && !name.startsWith('GMT')) return `${zone} (${name})`
    const off = new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'shortOffset' }).formatToParts(new Date()).find((p) => p.type === 'timeZoneName')?.value
    return off ? `${zone} (${off.replace('GMT', 'UTC')})` : zone
  } catch {
    return zone
  }
}

/** Every zone the browser knows, with the current one first when missing. */
export function zoneOptions(current: string): string[] {
  let zones: string[]
  try {
    zones = (Intl as unknown as { supportedValuesOf?: (k: string) => string[] }).supportedValuesOf?.('timeZone') ?? []
  } catch {
    zones = []
  }
  if (!zones.includes('UTC')) zones = [...zones, 'UTC']
  if (current && !zones.includes(current)) zones = [current, ...zones]
  return zones
}

/** Mailbox provider colours (chart palette, fixed order). */
export const mailboxColor: Record<string, string> = {
  gmail: 'var(--chart-gmail)',
  outlook: 'var(--chart-outlook)',
  yahoo: 'var(--chart-yahoo)',
  icloud: 'var(--chart-icloud)',
  other: 'var(--chart-other)',
}

/** Sending provider colours. */
export const providerColor: Record<string, string> = {
  ses: 'var(--chart-ses)',
  postal: 'var(--chart-postal)',
  postmark: 'var(--chart-postmark)',
  other: 'var(--chart-other)',
}

/** "34 min", "1 h 12 min". */
export function minutes(m: number): string {
  if (m < 60) return `${Math.max(1, m)} min`
  const h = Math.floor(m / 60)
  const r = m % 60
  return r ? `${h} h ${r} min` : `${h} h`
}

/** Prices on the boards: "$2.84", "$0.00". */
export function dollars(v: number): string {
  return `$${v.toFixed(2)}`
}

/** A check's title with its group, for lists ordered by severity. */
export function flatTitle(c: Check): string {
  return c.flatTitle || c.title
}

/** The checks of a run, in the order of the checklist groups. */
export function checksByGroup(run: Run): { group: Group; checks: Check[] }[] {
  return run.groups.map((g) => ({ group: g, checks: run.checks.filter((c) => c.group === g.key) }))
}

/**
 * Warnings people confirm before sending (keys and sentences); the ones
 * with a one-click alternative ("Exclude them instead") come first.
 */
export function acknowledgements(run: Run): { key: string; text: string; fix?: Fix }[] {
  return run.checks
    .filter((c) => c.status === 'warn')
    .map((c) => ({ key: c.key, text: c.ack || `I understand: ${c.title}.`, fix: c.fixes.find((f) => f.kind === 'apply' && f.action === 'exclude_sunset') }))
    .sort((a, b) => Number(Boolean(b.fix)) - Number(Boolean(a.fix)))
}

/** "Nov 4, 2026" in a zone. */
export function zonedDate(at: string | Date, tz?: string): string {
  try {
    return new Intl.DateTimeFormat('en-US', { timeZone: tz || undefined, month: 'short', day: 'numeric', year: 'numeric' }).format(new Date(at))
  } catch {
    return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric' }).format(new Date(at))
  }
}

/** "09:00" in a zone. */
export function zonedTime(at: string | Date, tz?: string): string {
  try {
    return new Intl.DateTimeFormat('en-GB', { timeZone: tz || undefined, hour: '2-digit', minute: '2-digit', hour12: false }).format(new Date(at))
  } catch {
    return new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit', hour12: false }).format(new Date(at))
  }
}

/** "Nov 4, 2026 · 09:00 America/New_York". */
export function zonedWhen(at: string | Date, tz?: string): string {
  return `${zonedDate(at, tz)} · ${zonedTime(at, tz)}${tz ? ` ${tz}` : ''}`
}

/** Group names in running text: "Reputation, authentication, content, compliance and routing". */
export const groupWords: Record<string, string> = {
  reputation: 'reputation',
  authentication: 'authentication',
  content: 'content',
  links: 'links',
  compliance: 'compliance',
  audience: 'audience quality',
  routing: 'routing',
}

/** Group headings: "Sender reputation", "Audience quality". */
export const groupTitles: Record<string, string> = {
  reputation: 'Sender reputation',
  authentication: 'Authentication',
  content: 'Content',
  links: 'Links',
  compliance: 'Compliance',
  audience: 'Audience quality',
  routing: 'Routing',
}

/** Group names that start a one-line row: "Audience: 0 recently bounced, …". */
export const flatGroupLabels: Record<string, string> = { ...groupTitles, audience: 'Audience' }

/**
 * The order of blocks and warnings when they're listed by severity: legal
 * requirements and what filters every recipient first, then the rest.
 */
export const severityOrder = ['compliance', 'links', 'content', 'authentication', 'reputation', 'audience', 'routing']

export function bySeverity(a: Check, b: Check): number {
  return severityOrder.indexOf(a.group) - severityOrder.indexOf(b.group)
}

/** "just now", "12 minutes ago", "2 hours ago", "yesterday", "3 days ago". */
export function agoWords(d: string | number | Date, now: Date = new Date()): string {
  const m = Math.round((now.getTime() - new Date(d).getTime()) / 60_000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m} ${plural(m, 'minute', 'minutes')} ago`
  const h = Math.round(m / 60)
  if (h < 24) return `${h} ${plural(h, 'hour', 'hours')} ago`
  const days = Math.round(h / 24)
  return days === 1 ? 'yesterday' : `${days} days ago`
}

/** Splits text so web addresses ("bit.ly/3xK9tQ", "acme.io/blog/…") can be set in mono. */
export function linkParts(text: string): { text: string; link: boolean }[] {
  const out: { text: string; link: boolean }[] = []
  const re = /\b(?:https?:\/\/)?(?:[a-z0-9-]+\.)+[a-z]{2,}\/[^\s,;)]*[^\s,;.)]/gi
  let last = 0
  for (const m of text.matchAll(re)) {
    if (m.index! > last) out.push({ text: text.slice(last, m.index), link: false })
    out.push({ text: m[0], link: true })
    last = m.index! + m[0].length
  }
  if (last < text.length) out.push({ text: text.slice(last), link: false })
  return out
}
