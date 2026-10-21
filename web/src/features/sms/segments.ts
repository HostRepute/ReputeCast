/**
 * The SMS segment counter, as the server counts (internal/sms/segments.go):
 * GSM-7 until one character is outside it, then UCS-2; 160/153 septets or
 * 70/67 code units per segment; an escape sequence or a surrogate pair never
 * straddles two parts. Both sides test against
 * internal/sms/testdata/segments.json.
 */

export type Encoding = 'gsm7' | 'ucs2'

export type OutsideKind =
  | 'curly_apostrophe'
  | 'curly_quote'
  | 'en_dash'
  | 'em_dash'
  | 'minus'
  | 'ellipsis'
  | 'non_breaking_space'
  | 'thin_space'
  | 'zero_width_space'
  | 'prime'
  | 'backtick'
  | 'emoji'
  | 'other'

export interface Outside {
  char: string
  kind: OutsideKind
  replacement: string
  removable: boolean
  count: number
  /** Offset in code points of its first use. */
  first: number
}

export interface Count {
  characters: number
  units: number
  encoding: Encoding
  segments: number
  capacity: number
  fill: number[]
  left: number
  outside: Outside[]
}

export const GSM7_SINGLE = 160
export const GSM7_PART = 153
export const UCS2_SINGLE = 70
export const UCS2_PART = 67

const basic = new Set(
  Array.from('@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !"#¤%&\'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà'),
)
const extension = new Set(Array.from('\f^{}\\[~]|€'))

const lookalikes: Record<number, { kind: OutsideKind; with: string }> = {
  0x2018: { kind: 'curly_apostrophe', with: "'" },
  0x2019: { kind: 'curly_apostrophe', with: "'" },
  0x201b: { kind: 'curly_apostrophe', with: "'" },
  0x201a: { kind: 'curly_apostrophe', with: "'" },
  0x201c: { kind: 'curly_quote', with: '"' },
  0x201d: { kind: 'curly_quote', with: '"' },
  0x201e: { kind: 'curly_quote', with: '"' },
  0x201f: { kind: 'curly_quote', with: '"' },
  0x2013: { kind: 'en_dash', with: '-' },
  0x2014: { kind: 'em_dash', with: '-' },
  0x2015: { kind: 'em_dash', with: '-' },
  0x2212: { kind: 'minus', with: '-' },
  0x2026: { kind: 'ellipsis', with: '...' },
  0x00a0: { kind: 'non_breaking_space', with: ' ' },
  0x2009: { kind: 'thin_space', with: ' ' },
  0x202f: { kind: 'thin_space', with: ' ' },
  0x2007: { kind: 'thin_space', with: ' ' },
  0x200b: { kind: 'zero_width_space', with: '' },
  0xfeff: { kind: 'zero_width_space', with: '' },
  0x2032: { kind: 'prime', with: "'" },
  0x2033: { kind: 'prime', with: '"' },
  0x60: { kind: 'backtick', with: "'" },
}

/** Whether a character can travel in GSM-7 (basic or extension table). */
export function inGSM7(ch: string): boolean {
  return basic.has(ch) || extension.has(ch)
}

function isEmoji(cp: number): boolean {
  return cp > 0xffff || (cp >= 0x2600 && cp <= 0x27bf) || (cp >= 0x2b00 && cp <= 0x2bff) || cp === 0xfe0f || cp === 0x200d
}

/** Measures a text exactly as it will be sent. */
export function countText(s: string): Count {
  const chars = Array.from(s)
  const c: Count = { characters: chars.length, units: 0, encoding: 'gsm7', segments: 0, capacity: 0, fill: [], left: 0, outside: [] }
  const idx = new Map<string, number>()
  chars.forEach((ch, pos) => {
    if (inGSM7(ch)) return
    const i = idx.get(ch)
    if (i !== undefined) {
      c.outside[i]!.count++
      return
    }
    const cp = ch.codePointAt(0)!
    const l = lookalikes[cp]
    const o: Outside = { char: ch, kind: 'other', replacement: '', removable: false, count: 1, first: pos }
    if (l) {
      o.kind = l.kind
      o.replacement = l.with
      o.removable = l.with === ''
    } else if (isEmoji(cp)) {
      o.kind = 'emoji'
    }
    idx.set(ch, c.outside.length)
    c.outside.push(o)
  })
  if (c.outside.length) c.encoding = 'ucs2'
  const cost = (ch: string) => {
    if (c.encoding === 'ucs2') return ch.codePointAt(0)! > 0xffff ? 2 : 1
    return extension.has(ch) ? 2 : 1
  }
  for (const ch of chars) c.units += cost(ch)
  const single = c.encoding === 'ucs2' ? UCS2_SINGLE : GSM7_SINGLE
  const part = c.encoding === 'ucs2' ? UCS2_PART : GSM7_PART
  if (c.units === 0) {
    c.capacity = single
    c.left = single
    return c
  }
  if (c.units <= single) {
    c.segments = 1
    c.capacity = single
    c.fill = [c.units]
    c.left = single - c.units
    return c
  }
  c.capacity = part
  let cur = 0
  for (const ch of chars) {
    const u = cost(ch)
    if (cur + u > part) {
      c.fill.push(cur)
      cur = 0
    }
    cur += u
  }
  c.fill.push(cur)
  c.segments = c.fill.length
  c.left = part - cur
  return c
}

/** Replaces every character outside GSM-7 that has a lookalike; emoji stay. */
export function fixEncoding(s: string): string {
  let out = ''
  for (const ch of Array.from(s)) {
    const l = lookalikes[ch.codePointAt(0)!]
    out += l && !inGSM7(ch) ? l.with : ch
  }
  return out
}
