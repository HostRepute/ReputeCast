/**
 * WCAG 2.2 contrast for the color picker's chip, the same math as the
 * server's checks (internal/accessibility/color.go): the parity test feeds
 * both the same colors.
 */

export interface RGBA {
  r: number
  g: number
  b: number
  a: number
}

/** Contrast requirements of WCAG 2.2. */
export const AA_NORMAL = 4.5
export const AA_LARGE = 3
export const AAA_NORMAL = 7
export const AAA_LARGE = 4.5

const clamp = (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v))
const byte = (v: number) => Math.round(clamp(v, 0, 255))

/** "#RRGGBB" in capitals, as the boards write colors. */
export function hex(c: RGBA): string {
  return `#${[c.r, c.g, c.b].map((v) => v.toString(16).padStart(2, '0')).join('')}`.toUpperCase()
}

/** Composites a color onto an opaque background. */
export function over(c: RGBA, bg: RGBA): RGBA {
  if (c.a >= 0.999) return { ...c, a: 1 }
  const mix = (f: number, b: number) => Math.round(f * c.a + b * (1 - c.a))
  return { r: mix(c.r, bg.r), g: mix(c.g, bg.g), b: mix(c.b, bg.b), a: 1 }
}

function args(s: string): string[] | null {
  const open = s.indexOf('(')
  const close = s.lastIndexOf(')')
  if (open < 0 || close < open) return null
  return s
    .slice(open + 1, close)
    .replace(/[/,]/g, ' ')
    .split(/\s+/)
    .filter(Boolean)
}

/** "128", "50%" (of scale) or "0.5". */
function num(s: string, scale: number): number | null {
  const pct = s.endsWith('%')
  const v = Number(pct ? s.slice(0, -1) : s)
  if (s === '' || Number.isNaN(v)) return null
  return pct ? (v / 100) * scale : v
}

/**
 * Reads a CSS color: #RGB, #RGBA, #RRGGBB, #RRGGBBAA, rgb(), rgba(), hsl(),
 * hsla() and the named colors. Null for values that aren't a color.
 */
export function parseColor(value: string | undefined | null): RGBA | null {
  let s = (value ?? '').trim().toLowerCase()
  s = s.replace(/!important$/, '').trim().replace(/;$/, '').trim()
  if (!s) return null
  if (s === 'transparent') return { r: 0, g: 0, b: 0, a: 0 }
  if (s.startsWith('#')) {
    let h = s.slice(1)
    if (h.length === 3 || h.length === 4) h = [...h].map((x) => x + x).join('')
    if ((h.length !== 6 && h.length !== 8) || !/^[0-9a-f]+$/.test(h)) return null
    const at = (i: number) => parseInt(h.slice(i, i + 2), 16)
    return { r: at(0), g: at(2), b: at(4), a: h.length === 8 ? at(6) / 255 : 1 }
  }
  if (s.startsWith('rgb')) {
    const a = args(s)
    if (!a || a.length < 3) return null
    const ch = a.slice(0, 3).map((x) => num(x, 255))
    if (ch.some((x) => x === null)) return null
    let alpha = 1
    if (a.length >= 4) {
      const v = num(a[3]!, 1)
      if (v === null) return null
      alpha = clamp(v, 0, 1)
    }
    return { r: byte(ch[0]!), g: byte(ch[1]!), b: byte(ch[2]!), a: alpha }
  }
  if (s.startsWith('hsl')) {
    const a = args(s)
    if (!a || a.length < 3) return null
    const h = Number(a[0]!.replace(/deg$/, ''))
    const sat = num(a[1]!, 1)
    const light = num(a[2]!, 1)
    if (Number.isNaN(h) || sat === null || light === null) return null
    const c = fromHSL({ h, s: sat, l: light })
    if (a.length >= 4) {
      const v = num(a[3]!, 1)
      if (v !== null) c.a = clamp(v, 0, 1)
    }
    return c
  }
  const named = NAMED[s]
  return named ? { ...named, a: 1 } : null
}

/** The WCAG 2.2 relative luminance. */
export function luminance(c: RGBA): number {
  const ch = (v: number) => {
    const s = v / 255
    return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * ch(c.r) + 0.7152 * ch(c.g) + 0.0722 * ch(c.b)
}

/** The contrast ratio of two opaque colors (1 to 21). */
export function ratio(a: RGBA, b: RGBA): number {
  let la = luminance(a)
  let lb = luminance(b)
  if (la < lb) [la, lb] = [lb, la]
  return (la + 0.05) / (lb + 0.05)
}

/** Rounds a ratio down to one decimal, so 4.49 never shows as 4.5. */
export function roundDown(r: number): number {
  return Math.floor(r * 10 + 1e-9) / 10
}

/** "2.1:1". */
export function formatRatio(r: number): string {
  return `${roundDown(r).toFixed(1)}:1`
}

/** Large text: 24 px, or 18.66 px bold. */
export function isLarge(sizePx: number, bold: boolean): boolean {
  return sizePx >= 24 || (bold && sizePx >= 18.66)
}

/** Compares with a tolerance for floating-point error; the ratio itself is never rounded up. */
export function passes(r: number, need: number): boolean {
  return r + 1e-9 >= need
}

interface HSL {
  h: number
  s: number
  l: number
}

function toHSL(c: RGBA): HSL {
  const r = c.r / 255
  const g = c.g / 255
  const b = c.b / 255
  const mx = Math.max(r, g, b)
  const mn = Math.min(r, g, b)
  const l = (mx + mn) / 2
  if (mx === mn) return { h: 0, s: 0, l }
  const d = mx - mn
  const s = d / (1 - Math.abs(2 * l - 1))
  let h: number
  if (mx === r) h = ((g - b) / d) % 6
  else if (mx === g) h = (b - r) / d + 2
  else h = (r - g) / d + 4
  h *= 60
  if (h < 0) h += 360
  return { h, s, l }
}

function fromHSL(x: HSL): RGBA {
  const c = (1 - Math.abs(2 * x.l - 1)) * x.s
  const hp = (x.h % 360) / 60
  const xx = c * (1 - Math.abs((hp % 2) - 1))
  const [r, g, b] = hp < 1 ? [c, xx, 0] : hp < 2 ? [xx, c, 0] : hp < 3 ? [0, c, xx] : hp < 4 ? [0, xx, c] : hp < 5 ? [xx, 0, c] : [c, 0, xx]
  const m = x.l - c / 2
  const to = (v: number) => Math.round(clamp(v + m, 0, 1) * 255)
  return { r: to(r), g: to(g), b: to(b), a: 1 }
}

/**
 * The color closest to `from` (same hue and saturation, the lightness moved
 * one percent at a time) that reaches `need` against `fixed`, with its
 * ratio. Null when no shade of that hue gets there.
 */
export function nearest(from: RGBA, fixed: RGBA, need: number): { color: RGBA; ratio: number } | null {
  const r0 = ratio(from, fixed)
  if (passes(r0, need)) return { color: from, ratio: r0 }
  const base = toHSL(from)
  const attempt = (dir: number) => {
    for (let step = 1; step <= 100; step++) {
      const l = base.l + (dir * step) / 100
      if (l < 0 || l > 1) break
      const c = fromHSL({ h: base.h, s: base.s, l })
      const r = ratio(c, fixed)
      if (passes(r, need)) return { color: c, ratio: r, step }
    }
    return null
  }
  const dark = attempt(-1)
  const light = attempt(1)
  if (dark && light) return light.step < dark.step ? light : dark
  return dark ?? light
}

/** "white" or "black" for those two, the hex value otherwise. */
export function colorName(c: RGBA): string {
  const h = hex(c)
  return h === '#FFFFFF' ? 'white' : h === '#000000' ? 'black' : h
}

export interface ContrastVerdict {
  ratio: number
  large: boolean
  aaNormal: boolean
  aaaNormal: boolean
  aaLarge: boolean
  aaaLarge: boolean
  /** What this text needs: AA and AAA for its size. */
  needsAA: number
  needsAAA: number
  passesAA: boolean
  passesAAA: boolean
}

/** Every requirement for one text color on one background. */
export function verdict(text: RGBA, background: RGBA, sizePx: number, bold: boolean): ContrastVerdict {
  const bg = over(background, { r: 255, g: 255, b: 255, a: 1 })
  const r = ratio(over(text, bg), bg)
  const large = isLarge(sizePx, bold)
  const needsAA = large ? AA_LARGE : AA_NORMAL
  const needsAAA = large ? AAA_LARGE : AAA_NORMAL
  return {
    ratio: r,
    large,
    aaNormal: passes(r, AA_NORMAL),
    aaaNormal: passes(r, AAA_NORMAL),
    aaLarge: passes(r, AA_LARGE),
    aaaLarge: passes(r, AAA_LARGE),
    needsAA,
    needsAAA,
    passesAA: passes(r, needsAA),
    passesAAA: passes(r, needsAAA),
  }
}

export type SuggestionReason = 'other_buttons' | 'nearest_aa' | 'aaa'

export interface Suggestion {
  color: string
  ratio: number
  reason: SuggestionReason
}

/**
 * The picker's suggestions for a color that fails AA, as the server makes
 * them: the most used passing color among the email's other buttons, the
 * nearest shade that passes AA, and the nearest that passes AAA.
 */
export function suggestions(from: RGBA, against: RGBA, needAA: number, needAAA: number, others: string[] = []): Suggestion[] {
  const out: Suggestion[] = []
  const seen = new Set<string>([hex(from)])
  const counts = new Map<string, number>()
  for (const o of others) {
    const c = parseColor(o)
    if (!c || c.a < 0.999) continue
    const h = hex(c)
    if (h === hex(from) || !passes(ratio(c, against), needAA)) continue
    counts.set(h, (counts.get(h) ?? 0) + 1)
  }
  // The most used; on a tie the lowest hex value, as the server picks it.
  const best = [...counts.entries()].sort((a, b) => b[1] - a[1] || (a[0] < b[0] ? -1 : 1))[0]
  if (best) {
    out.push({ color: best[0], ratio: roundDown(ratio(parseColor(best[0])!, against)), reason: 'other_buttons' })
    seen.add(best[0])
  }
  const aa = nearest(from, against, needAA)
  if (aa && !seen.has(hex(aa.color))) {
    out.push({ color: hex(aa.color), ratio: roundDown(aa.ratio), reason: 'nearest_aa' })
    seen.add(hex(aa.color))
  }
  const aaa = nearest(from, against, needAAA)
  if (aaa && !seen.has(hex(aaa.color))) out.push({ color: hex(aaa.color), ratio: roundDown(aaa.ratio), reason: 'aaa' })
  return out
}

type RGB = { r: number; g: number; b: number }
const rgb = (n: number): RGB => ({ r: (n >> 16) & 255, g: (n >> 8) & 255, b: n & 255 })

/** CSS named colors. */
const NAMED: Record<string, RGB> = Object.fromEntries(
  (
    [
      ['aliceblue', 0xf0f8ff], ['antiquewhite', 0xfaebd7], ['aqua', 0x00ffff], ['aquamarine', 0x7fffd4], ['azure', 0xf0ffff], ['beige', 0xf5f5dc],
      ['bisque', 0xffe4c4], ['black', 0x000000], ['blanchedalmond', 0xffebcd], ['blue', 0x0000ff], ['blueviolet', 0x8a2be2], ['brown', 0xa52a2a],
      ['burlywood', 0xdeb887], ['cadetblue', 0x5f9ea0], ['chartreuse', 0x7fff00], ['chocolate', 0xd2691e], ['coral', 0xff7f50], ['cornflowerblue', 0x6495ed],
      ['cornsilk', 0xfff8dc], ['crimson', 0xdc143c], ['cyan', 0x00ffff], ['darkblue', 0x00008b], ['darkcyan', 0x008b8b], ['darkgoldenrod', 0xb8860b],
      ['darkgray', 0xa9a9a9], ['darkgreen', 0x006400], ['darkgrey', 0xa9a9a9], ['darkkhaki', 0xbdb76b], ['darkmagenta', 0x8b008b], ['darkolivegreen', 0x556b2f],
      ['darkorange', 0xff8c00], ['darkorchid', 0x9932cc], ['darkred', 0x8b0000], ['darksalmon', 0xe9967a], ['darkseagreen', 0x8fbc8f], ['darkslateblue', 0x483d8b],
      ['darkslategray', 0x2f4f4f], ['darkslategrey', 0x2f4f4f], ['darkturquoise', 0x00ced1], ['darkviolet', 0x9400d3], ['deeppink', 0xff1493], ['deepskyblue', 0x00bfff],
      ['dimgray', 0x696969], ['dimgrey', 0x696969], ['dodgerblue', 0x1e90ff], ['firebrick', 0xb22222], ['floralwhite', 0xfffaf0], ['forestgreen', 0x228b22],
      ['fuchsia', 0xff00ff], ['gainsboro', 0xdcdcdc], ['ghostwhite', 0xf8f8ff], ['gold', 0xffd700], ['goldenrod', 0xdaa520], ['gray', 0x808080],
      ['green', 0x008000], ['greenyellow', 0xadff2f], ['grey', 0x808080], ['honeydew', 0xf0fff0], ['hotpink', 0xff69b4], ['indianred', 0xcd5c5c],
      ['indigo', 0x4b0082], ['ivory', 0xfffff0], ['khaki', 0xf0e68c], ['lavender', 0xe6e6fa], ['lavenderblush', 0xfff0f5], ['lawngreen', 0x7cfc00],
      ['lemonchiffon', 0xfffacd], ['lightblue', 0xadd8e6], ['lightcoral', 0xf08080], ['lightcyan', 0xe0ffff], ['lightgoldenrodyellow', 0xfafad2], ['lightgray', 0xd3d3d3],
      ['lightgreen', 0x90ee90], ['lightgrey', 0xd3d3d3], ['lightpink', 0xffb6c1], ['lightsalmon', 0xffa07a], ['lightseagreen', 0x20b2aa], ['lightskyblue', 0x87cefa],
      ['lightslategray', 0x778899], ['lightslategrey', 0x778899], ['lightsteelblue', 0xb0c4de], ['lightyellow', 0xffffe0], ['lime', 0x00ff00], ['limegreen', 0x32cd32],
      ['linen', 0xfaf0e6], ['magenta', 0xff00ff], ['maroon', 0x800000], ['mediumaquamarine', 0x66cdaa], ['mediumblue', 0x0000cd], ['mediumorchid', 0xba55d3],
      ['mediumpurple', 0x9370db], ['mediumseagreen', 0x3cb371], ['mediumslateblue', 0x7b68ee], ['mediumspringgreen', 0x00fa9a], ['mediumturquoise', 0x48d1cc], ['mediumvioletred', 0xc71585],
      ['midnightblue', 0x191970], ['mintcream', 0xf5fffa], ['mistyrose', 0xffe4e1], ['moccasin', 0xffe4b5], ['navajowhite', 0xffdead], ['navy', 0x000080],
      ['oldlace', 0xfdf5e6], ['olive', 0x808000], ['olivedrab', 0x6b8e23], ['orange', 0xffa500], ['orangered', 0xff4500], ['orchid', 0xda70d6],
      ['palegoldenrod', 0xeee8aa], ['palegreen', 0x98fb98], ['paleturquoise', 0xafeeee], ['palevioletred', 0xdb7093], ['papayawhip', 0xffefd5], ['peachpuff', 0xffdab9],
      ['peru', 0xcd853f], ['pink', 0xffc0cb], ['plum', 0xdda0dd], ['powderblue', 0xb0e0e6], ['purple', 0x800080], ['rebeccapurple', 0x663399],
      ['red', 0xff0000], ['rosybrown', 0xbc8f8f], ['royalblue', 0x4169e1], ['saddlebrown', 0x8b4513], ['salmon', 0xfa8072], ['sandybrown', 0xf4a460],
      ['seagreen', 0x2e8b57], ['seashell', 0xfff5ee], ['sienna', 0xa0522d], ['silver', 0xc0c0c0], ['skyblue', 0x87ceeb], ['slateblue', 0x6a5acd],
      ['slategray', 0x708090], ['slategrey', 0x708090], ['snow', 0xfffafa], ['springgreen', 0x00ff7f], ['steelblue', 0x4682b4], ['tan', 0xd2b48c],
      ['teal', 0x008080], ['thistle', 0xd8bfd8], ['tomato', 0xff6347], ['turquoise', 0x40e0d0], ['violet', 0xee82ee], ['wheat', 0xf5deb3],
      ['white', 0xffffff], ['whitesmoke', 0xf5f5f5], ['yellow', 0xffff00], ['yellowgreen', 0x9acd32],
    ] as [string, number][]
  ).map(([k, v]) => [k, rgb(v)]),
)
