import type { Schemas } from '@/api/client'
import { t } from '@/i18n'
import { m } from './messages'

export type Band = 'cold' | 'warm' | 'hot'
export type Bands = Schemas['ScoringBands']

/**
 * The score bands' ordinal ramp (design-system §7, chart-score-1 to 3):
 * never the status colours, never the tier palette.
 */
export const bandColor: Record<Band, string> = {
  cold: 'var(--chart-score-1)',
  warm: 'var(--chart-score-2)',
  hot: 'var(--chart-score-3)',
}

export const bandOrder: Band[] = ['cold', 'warm', 'hot']

export function bandName(b: Band): string {
  return t(m[b])
}

/** The band a score falls in under the thresholds. */
export function bandOf(score: number, bands: Bands): Band {
  if (score >= bands.hotFrom) return 'hot'
  if (score >= bands.warmFrom) return 'warm'
  return 'cold'
}

/** A band's score range: Cold 0–19, Warm 20–49, Hot 50–100. */
export function bandRange(b: Band, bands: Bands): [number, number] {
  switch (b) {
    case 'cold':
      return [0, bands.warmFrom - 1]
    case 'warm':
      return [bands.warmFrom, bands.hotFrom - 1]
  }
  return [bands.hotFrom, 100]
}
