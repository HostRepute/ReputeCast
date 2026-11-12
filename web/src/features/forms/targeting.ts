import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { andList, dayMonthDate, getDisplayTimeZone, orList } from '@/lib/format'
import { isoToZonedLocal, zonedToISO } from '@/features/website/postFormat'
import { m } from './builder.messages'

/**
 * Where and when a form shows (M60): its targeting as the builder's one
 * line says it, and the schedule's dates as datetime-local fields hold
 * them, in the workspace's time zone.
 */

export type Targeting = Schemas['SignupFormTargeting']
export type Schedule = Schemas['SignupFormSchedule']

/** No rules: every page, everyone, any time. */
export function emptyTargeting(): Targeting {
  return {
    pages: { mode: 'all', rules: [] },
    sources: { utmSource: [], utmMedium: [], utmCampaign: [], referrer: 'any' },
    devices: [],
    countries: { mode: 'all', codes: [] },
    visitors: 'all',
    schedule: {},
  }
}

/** A copy the builder can change without touching the saved form. */
export function cloneTargeting(x: Targeting): Targeting {
  return {
    pages: { mode: x.pages.mode, rules: x.pages.rules.map((r) => ({ ...r })) },
    sources: { utmSource: [...x.sources.utmSource], utmMedium: [...x.sources.utmMedium], utmCampaign: [...x.sources.utmCampaign], referrer: x.sources.referrer },
    devices: [...x.devices],
    countries: { mode: x.countries.mode, codes: [...x.countries.codes] },
    visitors: x.visitors,
    schedule: { ...x.schedule },
  }
}

const referrerWords = { search: m.fromSearch, social: m.fromSocial, other_site: m.fromOtherSites, direct: m.directVisits } as const
const deviceWords = { desktop: m.onDesktop, tablet: m.onTablet, mobile: m.onMobile } as const
const deviceOrder = ['desktop', 'tablet', 'mobile'] as const

/**
 * The parts of a form's targeting, in order: pages, sources, devices,
 * countries (as typed), visitors, then the schedule's dates.
 */
export function targetingParts(x: Targeting): string[] {
  const parts: string[] = []
  if (x.pages.mode !== 'all') parts.push(t(x.pages.mode === 'only' ? m.shownOnPages : m.hiddenOnPages, { n: x.pages.rules.length }))
  const utm: [string, string[]][] = [
    ['utm_source', x.sources.utmSource],
    ['utm_medium', x.sources.utmMedium],
    ['utm_campaign', x.sources.utmCampaign],
  ]
  for (const [param, values] of utm) if (values.length) parts.push(t(m.fromUtm, { param, values: orList(values) }))
  if (x.sources.referrer !== 'any') parts.push(t(referrerWords[x.sources.referrer]))
  const devices = deviceOrder.filter((d) => x.devices.includes(d))
  // None ticked, or all three: every device.
  if (devices.length && devices.length < deviceOrder.length) parts.push(andList(devices.map((d) => t(deviceWords[d]))))
  if (x.countries.mode !== 'all' && x.countries.codes.length) {
    parts.push(x.countries.mode === 'only' ? t(m.fromCountries, { codes: andList(x.countries.codes) }) : t(m.notFromCountries, { codes: orList(x.countries.codes) }))
  }
  if (x.visitors !== 'all') parts.push(t(x.visitors === 'new' ? m.newVisitors : m.returningVisitors))
  if (x.schedule.startsAt) parts.push(t(m.fromDate, { date: dayMonthDate(x.schedule.startsAt) }))
  if (x.schedule.endsAt) parts.push(t(m.untilDate, { date: dayMonthDate(x.schedule.endsAt) }))
  return parts
}

/** "Shown on 3 pages · mobile · from US and CA · until 30 Nov", or "Shown on every page, to everyone." */
export function targetingSummary(x: Targeting): string {
  const parts = targetingParts(x)
  return parts.length ? parts.join(' · ') : t(m.shownEverywhere)
}

/** A hosted form's schedule: "Open now, with no end date.", "Opens 1 Dec", "Closes 30 Nov", "Open 1 Dec – 24 Dec". */
export function scheduleSummary(s: Schedule): string {
  if (s.startsAt && s.endsAt) return t(m.openBetween, { start: dayMonthDate(s.startsAt), end: dayMonthDate(s.endsAt) })
  if (s.startsAt) return t(m.opensOn, { date: dayMonthDate(s.startsAt) })
  if (s.endsAt) return t(m.closesOn, { date: dayMonthDate(s.endsAt) })
  return t(m.openNow)
}

/** The workspace's time zone, else the browser's. */
function zone(): string {
  return getDisplayTimeZone() || Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
}

/** A moment as a datetime-local field shows it in the workspace's time zone ("2026-11-30T23:59"); empty for none. */
export function toLocalInput(iso: string | undefined): string {
  return iso ? isoToZonedLocal(iso, zone()) : ''
}

/** The moment a datetime-local value is in the workspace's time zone; undefined when empty. */
export function fromLocalInput(local: string): string | undefined {
  return local ? zonedToISO(local, zone()) : undefined
}

/** "us, CA ,de" → ["us", "CA", "de"]: a comma-separated field's values, trimmed. */
export function listOf(text: string): string[] {
  return text
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}
