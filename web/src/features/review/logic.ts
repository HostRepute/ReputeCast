import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { calendarDate, decimal, inZone, money, pct } from '@/lib/format'
import { m } from './messages'

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
  return v === 'blocked' ? t(m.verdictBlocked) : v === 'warnings' ? t(m.verdictWarnings) : t(m.verdictReady)
}

/** A campaign's status in the header badge: "Draft", "Waiting for approval", "Scheduled". */
export function campaignWord(status: string): string {
  const words: Record<string, Parameters<typeof t>[0]> = {
    draft: m.campaignDraft,
    approval: m.campaignApproval,
    approved: m.campaignApproved,
    scheduled: m.campaignScheduled,
    sending: m.campaignSending,
    paused: m.campaignPaused,
    sent: m.campaignSent,
    cancelled: m.campaignCancelled,
    failed: m.campaignFailed,
  }
  const d = words[status]
  return d ? t(d) : status
}

/** The badge word of a check row. */
export function statusWord(s: CheckStatus): string {
  return s === 'pass' ? t(m.statusPass) : s === 'warn' ? t(m.statusWarn) : s === 'block' ? t(m.statusBlock) : t(m.statusUnknown)
}

/** "27 checks passed, 2 warnings, 0 blocks". */
export function countsLine(c: Counts): string {
  return t(m.countsLine, { passed: c.passed, warnings: c.warnings, blocks: c.blocks })
}

/** "25 passed · 2 warnings · 0 blocks" (the approval cards). */
export function countsShort(c: { passed: number; warnings: number; blocks: number }): string {
  return t(m.countsShort, { passed: c.passed, warnings: c.warnings, blocks: c.blocks })
}

/**
 * The Blocked banner's sentence: "3 blocking issues, 2 warnings, 24 checks
 * passed. Sending stays disabled until every block is fixed. Each one has a
 * one-click fix below."
 */
export function blockedLine(c: Counts, checks: Check[]): string {
  const blocks = checks.filter((x) => x.status === 'block')
  const fixable = blocks.filter((x) => primaryFix(x)).length
  const head = t(m.blockedHead, { blocks: c.blocks, warnings: c.warnings, passed: c.passed })
  const tail =
    fixable === 0
      ? ''
      : fixable === blocks.length
        ? blocks.length === 1
          ? t(m.blockedTailOne)
          : t(m.blockedTailEach)
        : t(m.blockedTailSome, { fixable })
  return tail ? `${head} ${tail}` : head
}

/**
 * The one-click fix a "fix all" button applies for a check: its first apply
 * fix. Marking an image decorative is a judgment, and removing a sponsor's
 * block a decision (Earn, X17): neither is applied in bulk.
 */
export function primaryFix(c: Check): Fix | undefined {
  return c.fixes.find((f) => f.kind === 'apply' && f.action !== 'accessibility_decorative' && f.action !== 'sponsor_remove_block')
}

/** "Fix both warnings", "Apply all 3 fixes". */
export function fixAllLabel(kind: 'warn' | 'block', n: number): string {
  return kind === 'block' ? t(m.fixAllBlocks, { n }) : t(m.fixAllWarnings, { n })
}

/** "Schedule send to 31,604 recipients" or "Send to 47,842 recipients". */
export function sendLabel(mode: Schedule['mode'], recipients: number): string {
  return mode === 'now' ? t(m.sendNow, { recipients }) : t(m.sendScheduled, { recipients })
}

/** "You'll confirm the audience and the two warnings on the next step." */
export function confirmNote(warnings: number): string {
  return t(m.confirmNote, { warnings })
}

/** "Wednesday 14 Oct 2026" from a local "2026-10-14" (no timezone shift). */
export function localDateWords(date: string): string {
  return calendarDate(date, 'longDate')
}

/** "Nov 4" from a local "2026-11-04". */
export function localMonthDay(date: string): string {
  return calendarDate(date, 'monthDay')
}

/** "Tuesday 14 Oct 2026 · 09:00 · America/New_York" (Scheduling card). */
export function scheduleCaption(s: Schedule): string {
  if (!s.date || !s.time) return t(m.pickDateTime)
  return [localDateWords(s.date), s.time, s.timezone].filter(Boolean).join(' · ')
}

/** "Tuesday 14 Oct 2026 at 09:00 America/New_York" (confirmation). */
export function scheduleSentence(s: Schedule): string {
  if (!s.date || !s.time) return ''
  return t(m.scheduleSentence, { date: localDateWords(s.date), time: s.time, timezone: s.timezone ?? '' }).trim()
}

/** "Nov 4, 09:00" (the approved card's schedule button). */
export function scheduleShort(s: Schedule): string {
  if (s.mode === 'now' || !s.date || !s.time) return t(m.now)
  return `${localMonthDay(s.date)}, ${s.time}`
}

/** "1 d 22 h 19 m" until a time; "now" once it passed. */
export function sendsIn(at: string | number | Date, now: Date = new Date()): string {
  const total = Math.max(0, Math.floor((new Date(at).getTime() - now.getTime()) / 60_000))
  if (total <= 0) return t(m.now)
  const d = Math.floor(total / 1440)
  const h = Math.floor((total % 1440) / 60)
  const mins = total % 60
  return [d ? t(m.days, { n: d }) : '', d || h ? t(m.hours, { n: h }) : '', t(m.minutes, { n: mins })].filter(Boolean).join(' ')
}

/** "about 3 hours", "about 45 minutes" (median wait). */
export function waitWords(minutes: number): string {
  if (minutes < 60) return t(m.waitMinutes, { n: Math.max(1, Math.round(minutes)) })
  return t(m.waitHours, { n: Math.round(minutes / 60) })
}

/** "2 h 40 min" (median wait in the inbox). */
export function waitShort(minutes: number): string {
  if (minutes < 60) return t(m.durMinutes, { m: Math.max(1, Math.round(minutes)) })
  const h = Math.floor(minutes / 60)
  const rest = Math.round(minutes % 60)
  return rest ? t(m.durHoursMinutes, { h, m: rest }) : t(m.durHours, { h })
}

/** "bounce 2.0% · complaint 0.10%". */
export function thresholdsLine(th: Thresholds): string {
  return t(m.thresholdsLine, { bounce: pct(th.bounce, 1), complaint: pct(th.complaint, 2) })
}

/** The thresholds inputs show "2.0 %", "0.10 %", "5 %". */
export function percentText(v: number, digits: number): string {
  return `${decimal(v, digits)} %`
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
  bounce: {
    min: 0.1,
    max: 20,
    digits: 1,
    get label() {
      return t(m.thresholdBounce)
    },
  },
  complaint: {
    min: 0.01,
    max: 5,
    digits: 2,
    get label() {
      return t(m.thresholdComplaint)
    },
  },
  mailboxComplaint: {
    min: 0.01,
    max: 5,
    digits: 2,
    get label() {
      return t(m.thresholdMailbox)
    },
  },
  providerErrors: {
    min: 1,
    max: 100,
    digits: 0,
    get label() {
      return t(m.thresholdErrors)
    },
  },
}

/** Validates one threshold value; returns the error sentence or undefined. */
export function thresholdError(field: ThresholdField, v: number | null, th: Thresholds): string | undefined {
  const l = thresholdLimits[field]
  if (v === null) return t(m.errPercent, { example: decimal(2, 1) })
  if (v < l.min || v > l.max) return t(m.errRange, { min: l.min, max: l.max })
  if (field === 'mailboxComplaint' && v > th.complaint * 5) return t(m.errFiveTimes)
  if (field === 'complaint' && th.mailboxComplaint > v * 5) return t(m.errLowerFirst)
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

export const providerColor: Record<string, string> = {
  ses: 'var(--chart-ses)',
  postal: 'var(--chart-postal)',
  postmark: 'var(--chart-postmark)',
  other: 'var(--chart-other)',
}

/** "34 min", "1 h 12 min". */
export function minutes(n: number): string {
  if (n < 60) return t(m.durMinutes, { m: Math.max(1, n) })
  const h = Math.floor(n / 60)
  const r = n % 60
  return r ? t(m.durHoursMinutes, { h, m: r }) : t(m.durHours, { h })
}

/** Prices on the boards: "$2.84", "$0.00" (2,84 US$ in Vietnamese). */
export function dollars(v: number): string {
  return money(v, 2)
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
export function acknowledgements(run: Run): { key: string; keys: string[]; text: string; fix?: Fix }[] {
  const out: { key: string; keys: string[]; text: string; fix?: Fix }[] = []
  for (const c of run.checks.filter((x) => x.status === 'warn')) {
    const text = c.ack || t(m.ackDefault, { title: c.title })
    // Warnings confirmed in the same words (the serious accessibility
    // issues) are one checkbox that confirms them all.
    const same = out.find((a) => a.text === text)
    if (same) {
      same.keys.push(c.key)
      continue
    }
    out.push({ key: c.key, keys: [c.key], text, fix: c.fixes.find((f) => f.kind === 'apply' && f.action === 'exclude_sunset') })
  }
  return out.sort((a, b) => Number(Boolean(b.fix)) - Number(Boolean(a.fix)))
}

/** The warning keys the checked confirmations cover. */
export function acknowledgedKeys(acks: { key: string; keys: string[] }[], checked: string[]): string[] {
  return acks.filter((a) => checked.includes(a.key)).flatMap((a) => a.keys)
}

/** "Nov 4, 2026" in a zone. */
export function zonedDate(at: string | Date, tz?: string): string {
  return inZone(at, tz).date
}

/** "09:00" in a zone. */
export function zonedTime(at: string | Date, tz?: string): string {
  return inZone(at, tz).time
}

/** "Nov 4, 2026 · 09:00 America/New_York". */
export function zonedWhen(at: string | Date, tz?: string): string {
  return `${zonedDate(at, tz)} · ${zonedTime(at, tz)}${tz ? ` ${tz}` : ''}`
}

/** Words by group key, read in the app's language when used. */
function wordsOf(map: Record<string, Parameters<typeof t>[0]>): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, d] of Object.entries(map)) Object.defineProperty(out, k, { get: () => t(d), enumerable: true })
  return out
}

/** Group names in running text: "Reputation, authentication, content, compliance and routing". */
export const groupWords: Record<string, string> = wordsOf({
  reputation: m.wordReputation,
  authentication: m.wordAuthentication,
  content: m.wordContent,
  links: m.wordLinks,
  compliance: m.wordCompliance,
  audience: m.wordAudience,
  routing: m.wordRouting,
  accessibility: m.wordAccessibility,
})

const groupTitleMessages = {
  reputation: m.groupReputation,
  authentication: m.groupAuthentication,
  content: m.groupContent,
  links: m.groupLinks,
  compliance: m.groupCompliance,
  audience: m.groupAudience,
  routing: m.groupRouting,
  accessibility: m.groupAccessibility,
  languages: m.groupLanguages,
}

/** Group headings: "Sender reputation", "Audience quality". */
export const groupTitles: Record<string, string> = wordsOf(groupTitleMessages)

/** Group names that start a one-line row: "Audience: 0 recently bounced, …". */
export const flatGroupLabels: Record<string, string> = wordsOf({ ...groupTitleMessages, audience: m.groupAudienceShort })

/**
 * The order of blocks and warnings when they're listed by severity: legal
 * requirements and what filters every recipient first, then the rest.
 */
export const severityOrder = ['compliance', 'links', 'content', 'accessibility', 'authentication', 'reputation', 'audience', 'routing']

export function bySeverity(a: Check, b: Check): number {
  return severityOrder.indexOf(a.group) - severityOrder.indexOf(b.group)
}

/** "just now", "12 minutes ago", "2 hours ago", "yesterday", "3 days ago". */
export function agoWords(d: string | number | Date, now: Date = new Date()): string {
  const mins = Math.round((now.getTime() - new Date(d).getTime()) / 60_000)
  if (mins < 1) return t(m.agoJustNow)
  if (mins < 60) return t(m.agoMinutes, { n: mins })
  const h = Math.round(mins / 60)
  if (h < 24) return t(m.agoHours, { n: h })
  const days = Math.round(h / 24)
  return days === 1 ? t(m.agoYesterday) : t(m.agoDays, { n: days })
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
