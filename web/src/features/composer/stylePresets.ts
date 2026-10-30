/**
 * The Style panel in the writer's words (review 2026-10, CO-14, R4.U3):
 * presets for every block, with the MJML attributes behind them under
 * Advanced. Pure values and matching here; the panel is Inspector.tsx.
 */

/** Spacing of a section: px above and below, and at the sides (Composer board). */
export const SECTION_SPACING = { S: [8, 24], M: [16, 24], L: [32, 24] } as const

/** Space above and below a block in a column (MJML's own is 10 px); its sides stay. */
export const BLOCK_SPACING = { S: 4, M: 10, L: 20 } as const

/** Space above and below a hero's content. */
export const HERO_SPACING = { S: 24, M: 48, L: 80 } as const

/** Space inside a column, on every side. */
export const COLUMN_SPACING = { none: 0, S: 8, M: 16, L: 24 } as const

/** A button's size: its inner padding. Medium is MJML's own. */
export const BUTTON_SIZES = { S: '8px 16px', M: '10px 25px', L: '14px 32px' } as const

/** Corners: a button can also be a pill. */
export const BUTTON_CORNERS = { square: '0px', rounded: '6px', pill: '999px' } as const
export const CORNERS = { square: '0px', rounded: '8px' } as const

/** A divider's line and length. */
export const LINE_WEIGHTS = { thin: '1px', medium: '2px', thick: '4px' } as const
export const LINE_LENGTHS = { full: '100%', wide: '75%', short: '50%' } as const

/** "16px" → 16; anything else (em, %, auto) → NaN. */
export function px(v: string): number {
  const m = /^(-?\d+(?:\.\d+)?)(px)?$/.exec(v.trim())
  return m ? Number(m[1]) : NaN
}

/** The four sides of a box value ("10px 25px" → [10, 25, 10, 25]). */
export function sides(v: string): [number, number, number, number] | null {
  const p = v.trim().split(/\s+/).filter(Boolean).map(px)
  if (!p.length || p.length > 4 || p.some((n) => Number.isNaN(n))) return null
  const [t, r = t, b = t, l = r] = p as [number, number?, number?, number?]
  return [t, r, b, l]
}

/** The key of the preset a value is, compared as numbers ("10px 25px" is "10px 25px 10px 25px"). */
export function presetOf<K extends string>(presets: Record<K, string>, value: string): K | undefined {
  const v = sides(value)
  if (!v) return undefined
  return (Object.keys(presets) as K[]).find((k) => {
    const p = sides(presets[k])
    return p !== null && p.every((n, i) => n === v[i])
  })
}

/** The preset whose px a length is (a corner radius, a line weight). */
export function lengthPreset<K extends string>(presets: Record<K, string>, value: string): K | undefined {
  const v = value.trim() === '' ? NaN : value.trim().endsWith('%') ? value.trim() : px(value)
  return (Object.keys(presets) as K[]).find((k) => (typeof v === 'number' ? px(presets[k]) === v : presets[k] === v))
}

/** The vertical preset a block's padding is: the same space above and below. */
export function verticalPreset<K extends string>(presets: Record<K, number>, top: number, bottom: number): K | undefined {
  return top === bottom ? (Object.keys(presets) as K[]).find((k) => presets[k] === top) : undefined
}

/** The column preset: the same space on every side. */
export function evenPreset<K extends string>(presets: Record<K, number>, box: readonly number[]): K | undefined {
  return box.every((n) => n === box[0]) ? (Object.keys(presets) as K[]).find((k) => presets[k] === box[0]) : undefined
}
