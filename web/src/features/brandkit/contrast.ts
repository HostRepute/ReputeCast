import { decimal } from '@/lib/format'

/** WCAG contrast for the brand kit's colours (#RRGGBB). */

export const hexColour = /^#[0-9A-Fa-f]{6}$/

function channel(c: number): number {
  const v = c / 255
  return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4
}

function luminance(hex: string): number {
  const n = Number.parseInt(hex.slice(1), 16)
  return 0.2126 * channel((n >> 16) & 255) + 0.7152 * channel((n >> 8) & 255) + 0.0722 * channel(n & 255)
}

/** The contrast ratio of two colours, 1 to 21; 0 when one isn't a colour. */
export function contrast(a: string, b: string): number {
  if (!hexColour.test(a) || !hexColour.test(b)) return 0
  const [x, y] = [luminance(a), luminance(b)].sort((p, q) => q - p) as [number, number]
  return (x + 0.05) / (y + 0.05)
}

/** AAA from 7:1, AA from 4.5:1 (small text), else low. */
export function grade(ratio: number): 'AAA' | 'AA' | 'low' {
  return ratio >= 7 ? 'AAA' : ratio >= 4.5 ? 'AA' : 'low'
}

/** The ratios WCAG grades at (large text, AA, AAA): a shown ratio never rounds up past one. */
const THRESHOLDS = [3, 4.5, 7]

/**
 * A ratio as people read it, in their number format, to one decimal:
 * rounded (14.68 reads 14.7, as on the BrandKit board), but cut where
 * rounding would cross a threshold (4.47 reads 4.4, not a passing 4.5).
 */
export function ratioText(ratio: number): string {
  let r = Math.round(ratio * 10) / 10
  if (THRESHOLDS.some((th) => ratio < th && r >= th)) r = Math.floor(ratio * 10) / 10
  return decimal(r, 1)
}
