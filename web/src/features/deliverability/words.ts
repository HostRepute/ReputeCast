import type { Schemas } from '@/api/client'
import type { Tone } from '@/components/ui/Badge'
import { andList } from '@/lib/format'

export type Overview = Schemas['DeliverabilityOverview']
export type AssetHealth = Schemas['DeliverabilityAssetHealth']
export type CheckRun = Schemas['DeliverabilityCheckRun']

const plainDate = new Intl.DateTimeFormat('en-US', { day: 'numeric', month: 'short', timeZone: 'UTC' })

/** "2026-09-12" → "12 Sep": a calendar date, not a moment, so no timezone shift. */
export function dayMonthOf(date: string): string {
  const p: Record<string, string> = {}
  for (const x of plainDate.formatToParts(new Date(`${date.slice(0, 10)}T00:00:00Z`))) p[x.type] = x.value
  return `${p.day} ${p.month}`
}

/** "high" → "High" (Postmaster reputations, SNDS filter results). */
export function cap(s: string | undefined | null): string {
  if (!s) return '—'
  return s[0]!.toUpperCase() + s.slice(1)
}

/** How alerts and cards name a source: "UCEPROTECT Level 3" → "UCEPROTECT L3". */
export function shortSource(s: string): string {
  return s.replace(' Level ', ' L')
}

/** "pass · pass · p=quarantine" */
export function authLine(a: Schemas['DeliverabilityDomainAuth']): string {
  const dmarc = ['none', 'quarantine', 'reject'].includes(a.dmarc) ? `p=${a.dmarc}` : a.dmarc
  return `${a.spf} · ${a.dkim} · ${dmarc}`
}

/** "Sending domain · used by SES, Postmark and Postal" */
export function usedBy(names: string[]): string {
  return names.length ? `used by ${andList(names)}` : ''
}

/** Severity of an incident as a badge tone. */
export function severityTone(s: string): Tone {
  switch (s) {
    case 'bad':
      return 'bad'
    case 'warn':
      return 'warn'
    case 'ok':
      return 'ok'
  }
  return 'neutral'
}

/** A card tone as a badge tone. */
export function toneOf(t: string): Tone {
  return t === 'ok' || t === 'warn' || t === 'bad' || t === 'info' ? t : 'neutral'
}

/** Bar colours by tone, as on the board (ok-bar, warn-bar, bad-bar). */
export const barColor: Record<string, string> = { ok: '#15803D', warn: '#B45309', bad: '#DC2626' }

/** Sending-provider series colours (README › Chart palette). */
export function providerColor(adapter?: string): string {
  switch (adapter) {
    case 'ses':
      return 'var(--chart-ses)'
    case 'postal':
      return 'var(--chart-postal)'
    case 'postmark':
      return 'var(--chart-postmark)'
    case undefined:
    case '':
      return '#B91C1C'
  }
  return 'var(--chart-other)'
}

export function streamName(s: string): string {
  return s === 'broadcast' ? 'Broadcast' : s === 'automation' ? 'Automation' : s === 'system' ? 'System' : s
}

/** "about 6 s left" from a run's progress so far. */
export function eta(run: CheckRun, now: number): string {
  const started = new Date(run.startedAt).getTime()
  const elapsed = Math.max(0.5, (now - started) / 1000)
  if (!run.answered || !run.total) return 'starting'
  const left = ((run.total - run.answered) * elapsed) / run.answered
  if (left < 1) return 'finishing'
  return left < 90 ? `about ${Math.ceil(left)} s left` : `about ${Math.ceil(left / 60)} min left`
}

/** First-run state: a check is running and nothing has been checked yet. */
export function firstRun(o: Overview): boolean {
  return Boolean(o.checks.running) && o.assets.length > 0 && o.assets.every((a) => !a.blocklists.checkedAt && !a.auth)
}
