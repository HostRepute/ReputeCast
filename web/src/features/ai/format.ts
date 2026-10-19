import { getI18n, t } from '@/i18n'
import { count, dayMonthYear, decimal, money } from '@/lib/format'
import { m } from './messages'

/** Millionths of a dollar as money: 7840000 → $7.84. */
export function dollars(micros: number | null | undefined): string {
  return money((micros ?? 0) / 1e6, 2)
}

/** Token counts the way the board writes them: 1.62M, 460K, 14. */
export function tokens(n: number | null | undefined): string {
  const v = n ?? 0
  if (v >= 1_000_000) return `${trimZeros(decimal(v / 1_000_000, v >= 10_000_000 ? 1 : 2))}M`
  if (v >= 10_000) return `${count(Math.round(v / 1000))}K`
  return count(v)
}

/** "20.0" → "20", "1.50" → "1.5" (either decimal separator). */
function trimZeros(s: string): string {
  return s.replace(/([.,]\d*?)0+$/, '$1').replace(/[.,]$/, '')
}

/** A request's estimated cost for the card footer: "About $0.01." or "Less than $0.01." */
export function costLine(micros: number | null | undefined): string {
  const v = micros ?? 0
  if (v < 5_000) return t(m.costLess)
  return t(m.costAbout, { amount: money(Math.max(0.01, Math.round(v / 10_000) / 100), 2) })
}

/**
 * The cost before asking: about four characters a token in, and the
 * answer's expected size out, at the connection's prices (dollars per
 * million tokens).
 */
export function estimateMicros(chars: number, outTokens: number, prices: { input: number; output: number }): number {
  const inTokens = Math.ceil(chars / 4) + 450
  return inTokens * prices.input + outTokens * prices.output
}

/** Seconds, one decimal: 1200 ms → "1.2". */
export function seconds(ms: number): string {
  return decimal(ms / 1000, 1)
}

const languageNames = new Map<string, Intl.DisplayNames>()

/** A language code's name in the app's language: "es" → "Spanish". */
export function langName(code: string): string {
  const locale = getI18n().locale
  const base = code.toLowerCase().split(/[-_]/)[0] ?? code
  try {
    let n = languageNames.get(locale)
    if (!n) {
      n = new Intl.DisplayNames([locale], { type: 'language' })
      languageNames.set(locale, n)
    }
    return n.of(base) ?? base
  } catch {
    return base
  }
}

/**
 * A date the server keeps without a time (the price list's, the prices'):
 * "1 Oct 2026" whatever the viewer's timezone.
 */
export function calendarDay(iso: string): string {
  const d = new Date(iso)
  return dayMonthYear(new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate(), 12)))
}
