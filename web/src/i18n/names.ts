import { isArgumentElement, isLiteralElement, isPluralElement, isSelectElement, isTagElement, type MessageFormatElement } from '@formatjs/icu-messageformat-parser'

// Vietnamese names a language with the common noun "tiếng": "Tiếng Anh" on
// its own and at the start of a sentence, "tiếng Anh" inside one ("Thêm
// tiếng Anh"). Translators write {language} wherever the sentence needs it:
// a Vietnamese message renames, when it compiles, the plain arguments that
// sit inside a sentence (markInSentence), and a value that is a language's
// name, or a list of them, takes the case its place needs (viNames). Other
// values are kept as they are. internal/i18n/names.go is the server's side
// of the same rule.

/** The suffix of the name a plain argument takes inside a sentence. */
export const IN_SENTENCE = '\u0000in'

/**
 * Renames the plain arguments that follow other words of their sentence
 * ({language} → {language␀in}) and returns their names. An argument at the
 * start of the message, after the end of a sentence, a colon or a "·"
 * separator starts a sentence (a label's value, "Ngôn ngữ: Tiếng Việt").
 */
export function markInSentence(els: MessageFormatElement[]): { ast: MessageFormatElement[]; inside: Set<string> } {
  const inside = new Set<string>()
  return { ast: markFrom(els, true, inside)[0], inside }
}

function markFrom(els: MessageFormatElement[], start: boolean, inside: Set<string>): [MessageFormatElement[], boolean] {
  const out: MessageFormatElement[] = []
  for (const el of els) {
    if (isLiteralElement(el)) {
      start = startsSentence(el.value, start)
      out.push(el)
    } else if (isArgumentElement(el)) {
      if (start) out.push(el)
      else {
        inside.add(el.value)
        out.push({ ...el, value: el.value + IN_SENTENCE })
      }
      start = false
    } else if (isPluralElement(el) || isSelectElement(el)) {
      const options: typeof el.options = {}
      for (const [k, opt] of Object.entries(el.options)) options[k] = { ...opt, value: markFrom(opt.value, start, inside)[0] }
      out.push({ ...el, options })
      start = false
    } else if (isTagElement(el)) {
      const [children, after] = markFrom(el.children, start, inside)
      out.push({ ...el, children })
      start = after
    } else {
      out.push(el)
      start = false
    }
  }
  return [out, start]
}

/** Whether the word after text `t` starts a sentence; `start` is whether the word before it did. */
function startsSentence(t: string, start: boolean): boolean {
  const s = t.trimEnd()
  if (t.slice(s.length).includes('\n')) return true
  if (!s) return start
  return '.!?…·:'.includes(s.charAt(s.length - 1))
}

// A Vietnamese language name: "tiếng" and up to five capitalised words,
// maybe with a region in brackets ("Tiếng Anh (Mỹ)").
const languageName = /^[Tt]iếng(?: \p{Lu}[\p{L}\p{M}'’-]*){1,5}(?: \([^()]+\))?$/u

/**
 * A language's name, or a list of them, with "tiếng" in the case its place
 * needs: capitalised at the start of a sentence, for the first name only.
 * Any other text comes back as it is.
 */
export function viNames(s: string, inSentence: boolean): string {
  if (!s.includes('iếng ')) return s
  const parts = s.split(/(, | và | hoặc )/)
  for (let i = 0; i < parts.length; i += 2) if (!languageName.test(parts[i] ?? '')) return s
  return parts.map((p, i) => (i % 2 ? p : (i === 0 && !inSentence ? 'T' : 't') + p.slice(1))).join('')
}

/** A Vietnamese message's values: language names in the case each place needs. */
export function namedValues<V>(values: Record<string, V> | undefined, inside: ReadonlySet<string>): Record<string, V> | undefined {
  if (!values) return values
  const out: Record<string, V> = {}
  for (const [k, v] of Object.entries(values)) out[k] = typeof v === 'string' ? (viNames(v, false) as V) : v
  for (const k of inside) {
    const v = values[k]
    if (v !== undefined) out[k + IN_SENTENCE] = typeof v === 'string' ? (viNames(v, true) as V) : v
  }
  return out
}
