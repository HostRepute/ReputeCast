import { isLiteralElement, isPluralElement, isSelectElement, isTagElement, parse, type MessageFormatElement } from '@formatjs/icu-messageformat-parser'
import { printAST } from '@formatjs/icu-messageformat-parser/printer.js'

/**
 * The pseudo-locale "Pseudo (+40%)" (en-XA): every letter accented, about
 * 40% longer and wrapped in ⟦ ⟧, so cut-off text shows by its missing ⟧ and
 * text that skipped web/src/i18n shows without brackets. Only the literal
 * text of a message is accented; its values (names, numbers, dates) keep
 * their spelling. internal/i18n/pseudo.go does the same on the server.
 */
export const PSEUDO_LOCALE = 'en-XA'

const accents: Record<string, string> = {
  a: 'á', b: 'b', c: 'ç', d: 'ð', e: 'é', f: 'ƒ', g: 'ĝ', h: 'ĥ', i: 'î', j: 'ĵ', k: 'ķ', l: 'ļ', m: 'm',
  n: 'ñ', o: 'ö', p: 'þ', q: 'q', r: 'ŕ', s: 'š', t: 'ţ', u: 'û', v: 'v', w: 'ŵ', x: 'x', y: 'ý', z: 'ž',
  A: 'Á', B: 'B', C: 'Ç', D: 'Ð', E: 'É', F: 'F', G: 'Ĝ', H: 'Ĥ', I: 'Î', J: 'Ĵ', K: 'Ķ', L: 'Ļ', M: 'M',
  N: 'Ñ', O: 'Ö', P: 'Þ', Q: 'Q', R: 'Ŕ', S: 'Š', T: 'Ţ', U: 'Û', V: 'V', W: 'Ŵ', X: 'X', Y: 'Ý', Z: 'Ž',
}

/** Names and acronyms (a capital after the first letter: ReputeCast, SES, A/B) keep their spelling. */
function accentWord(word: string): string {
  if (/^.+\p{Lu}/u.test(word)) return word
  let out = ''
  for (const c of word) out += accents[c] ?? c
  return out
}

/** Accents every letter of plain text, word by word. */
export function accent(text: string): string {
  return text.replace(/\S+/g, accentWord)
}

function transform(els: MessageFormatElement[]): MessageFormatElement[] {
  return els.map((el) => {
    if (isLiteralElement(el)) return { ...el, value: accent(el.value) }
    if (isPluralElement(el) || isSelectElement(el)) {
      const options = Object.fromEntries(Object.entries(el.options).map(([k, o]) => [k, { ...o, value: transform(o.value) }]))
      return { ...el, options }
    }
    if (isTagElement(el)) return { ...el, children: transform(el.children) }
    return el
  })
}

const sources = new Map<string, string>()

/** The source message with its literal text accented (still ICU). */
export function pseudoSource(message: string): string {
  let out = sources.get(message)
  if (out === undefined) {
    try {
      out = printAST(transform(parse(message)))
    } catch {
      out = message
    }
    sources.set(message, out)
  }
  return out
}

/** ⟦ and " ··⟧": the brackets and the padding that make text about 40% longer. */
export function pseudoWrap(text: string): [open: string, close: string] {
  const n = [...text].length
  if (n === 0) return ['', '']
  const dots = Math.max(1, Math.round(n * 0.4) - 1)
  return ['⟦', ` ${'·'.repeat(dots)}⟧`]
}
