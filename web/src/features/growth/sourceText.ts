import type { Schemas } from '@/api/client'
import { formatList, t } from '@/i18n'
import { count, dayMonthDate, decimal, monthName, pct, zonedParts } from '@/lib/format'
import { m } from './messages'

export type Source = Schemas['GrowthSource']

/** A calendar day ("2026-09-16") as a moment that falls on that day in any timezone. */
export function dayOf(day: string): Date {
  return new Date(`${day}T12:00:00Z`)
}

/** "Feb 2024", "June 2026". */
export function monthYear(d: string | Date, style: 'short' | 'long' = 'short'): string {
  return `${monthName(d, style)} ${zonedParts(d).year}`
}

/** Whether the source is an import's (its own or one it carried). */
export function isImport(s: Source): boolean {
  return s.kind === 'import' || (s.kind === 'carried' && (s.carried ?? '').startsWith('import'))
}

/** "Newsletter signup", "API key “Website signup form”", "Brevo import (18 Sep)". */
export function sourceName(s: Source): string {
  switch (s.kind) {
    case 'api_key':
      return t(m.srcApiKey, { name: s.name })
    case 'subscribe_page':
      return t(m.srcSubscribePage)
    case 'person':
      return t(m.srcPerson)
    case 'preference_center':
      return t(m.srcPreferenceCenter)
    case 'import':
      return t(m.srcImport, { name: s.name, day: s.importedAt ? dayMonthDate(s.importedAt) : '' })
    case 'carried':
      if (s.carried === 'import:csv') return t(m.srcCsvImport, { month: s.firstJoin ? monthYear(s.firstJoin) : '' })
      return s.carriedLabel || s.name || t(m.srcUnknown)
    case 'other':
      return t(m.srcOther)
  }
  return s.name || t(m.srcUnknown)
}

/** The chart tooltip's short name ("API key"). */
export function shortName(s: Source): string {
  return s.kind === 'api_key' ? t(m.srcApiKeyShort) : sourceName(s)
}

const bare = (u: string) => u.replace(/^https?:\/\//, '').replace(/\/$/, '')

/** The second line: what kind of source, and where. */
export function sourceLine(s: Source, list?: string): string {
  switch (s.kind) {
    case 'form':
      if (list && s.listChoice) return t(m.lineFormList, { list })
      return s.hosts.length ? t(m.lineForm, { kind: s.formKind ?? 'embedded', hosts: formatList(s.hosts) }) : t(m.lineFormNoHosts, { kind: s.formKind ?? 'embedded' })
    case 'subscribe_page':
      return t(m.lineSubscribePage, { url: bare(s.url ?? '') })
    case 'api_key':
      return t(m.lineApiKey, { prefix: s.keyPrefix ?? '' })
    case 'integration':
      return t(m.lineIntegration, { day: s.firstJoin ? dayMonthDate(s.firstJoin) : '' })
    case 'person':
      return t(m.linePerson)
    case 'preference_center':
      return t(m.linePreferenceCenter, { list: list ?? '' })
    case 'import':
      if (s.importCount && s.tookImportDay !== undefined) return t(m.lineImportTook, { count: s.tookImportDay })
      return t(m.lineImportCount, { count: s.tookImportDay ?? 0 })
    case 'carried':
      if ((s.carried ?? '').startsWith('import')) return t(list ? m.lineCarriedImportList : m.lineCarriedImport)
      return t(m.lineCarried)
  }
  return ''
}

/** "0.3%", "0.01%": rates with the precision the boards use. */
export function ratePct(v: number | undefined | null, digits: number): string {
  if (v === undefined || v === null) return '–'
  return pct(v * 100, digits)
}

/** "3.5": a flag factor with one decimal. */
export function factor(v: number): string {
  return decimal(v, 1)
}

/** "+1,284", "−3". */
export function signedCount(n: number): string {
  return n > 0 ? `+${count(n)}` : n < 0 ? `−${count(-n)}` : '0'
}
