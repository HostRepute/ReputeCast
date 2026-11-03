import { isArgumentElement, isLiteralElement, isPluralElement, isSelectElement, isTagElement, type MessageFormatElement } from '@formatjs/icu-messageformat-parser'

// A language's name takes the case its place needs in some languages.
// Vietnamese names a language with the common noun "tiếng": "Tiếng Anh" on
// its own and at the start of a sentence, "tiếng Anh" inside one ("Thêm
// tiếng Anh"). Spanish and French write language names in lower case
// inside a sentence and capitalise them on their own: "Inglés" in a list of
// languages, "Añadir inglés"; "Anglais", "Version en anglais". English and
// German write a name the same everywhere.
//
// Translators write {language} wherever the sentence needs it: such a
// message renames, when it compiles, the plain arguments that sit inside a
// sentence (markInSentence), and a value that is a language's name, or a
// list of them, takes the case its place needs (caseNames). Other values
// are kept as they are. internal/i18n/names.go is the server's side of the
// same rule.

/** Whether a language writes a language's name in lower case inside a sentence. */
export function lowersLanguageNames(locale: string): boolean {
  return locale === 'vi' || locale === 'es' || locale === 'fr'
}

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

// Spanish and French lists of names: "Inglés, español y alemán", "Anglais, espagnol et allemand".
const listSeparators: Record<string, RegExp> = {
  es: /(, | y | e | o | u )/,
  fr: /(, | et | ou )/,
}

// The ISO 639-1 languages: the names the browser gives them are the language
// names a message can be given (internal/i18n/names.go has the same list).
const ISO639 =
  'aa ab ae af ak am an ar as av ay az ba be bg bi bm bn bo br bs ca ce ch co cr cs cu cv cy da de dv dz ee el en eo es et eu fa ff fi fj fo fr fy ga gd gl gn gu gv ha he hi ho hr ht hu hy hz ia id ie ig ii ik io is it iu ja jv ka kg ki kj kk kl km kn ko kr ks ku kv kw ky la lb lg li ln lo lt lu lv mg mh mi mk ml mn mr ms mt my na nb nd ne ng nl nn no nr nv ny oc oj om or os pa pi pl ps pt qu rm rn ro ru rw sa sc sd se sg si sk sl sm sn so sq sr ss st su sv sw ta te tg th ti tk tl tn to tr ts tt tw ty ug uk ur uz ve vi vo wa wo xh yi yo za zh zu fil haw'

/** Each language's language names, in lower case. */
const known = new Map<string, Set<string>>()

function namesOf(locale: string): Set<string> {
  let set = known.get(locale)
  if (!set) {
    set = new Set()
    try {
      const n = new Intl.DisplayNames([locale], { type: 'language' })
      for (const code of ISO639.split(' ')) {
        const name = n.of(code)
        if (name && name !== code) set.add(name.toLocaleLowerCase(locale))
      }
    } catch {
      // No display names in this browser: names keep their case.
    }
    known.set(locale, set)
  }
  return set
}

function isLanguageName(locale: string, s: string): boolean {
  const key = s.trim().toLocaleLowerCase(locale)
  if (!key) return false
  const set = namesOf(locale)
  if (set.has(key)) return true
  const i = key.indexOf(' (')
  return i > 0 && key.endsWith(')') && set.has(key.slice(0, i))
}

function upperFirst(locale: string, s: string): string {
  return s.charAt(0).toLocaleUpperCase(locale) + s.slice(1)
}

function lowerFirst(locale: string, s: string): string {
  // An acronym or a name written in capitals keeps them.
  if (s.length > 1 && s.charAt(1) !== s.charAt(1).toLocaleLowerCase(locale)) return s
  return s.charAt(0).toLocaleLowerCase(locale) + s.slice(1)
}

/**
 * A language's name as it stands on its own (a label, a list of languages,
 * a table cell): Spanish and French capitalise it ("inglés" → "Inglés").
 * The name is noted, so it takes the case its place needs inside a sentence
 * too. Other languages' names come back as they are.
 */
export function standaloneLanguageName(locale: string, name: string): string {
  if ((locale !== 'es' && locale !== 'fr') || !name) return name
  namesOf(locale).add(name.toLocaleLowerCase(locale))
  return upperFirst(locale, name)
}

/**
 * A Spanish or French language name, or a list of them, capitalised at the
 * start of a sentence (the first name only) and in lower case elsewhere.
 * Any other text comes back as it is.
 */
export function romanceNames(locale: string, s: string, inSentence: boolean): string {
  const sep = listSeparators[locale]
  if (!sep || !s) return s
  const parts = s.split(sep)
  for (let i = 0; i < parts.length; i += 2) if (!isLanguageName(locale, parts[i] ?? '')) return s
  return parts.map((p, i) => (i % 2 ? p : i === 0 && !inSentence ? upperFirst(locale, p) : lowerFirst(locale, p))).join('')
}

/** A value in the case its place needs, in a language whose names change case. */
export function caseNames(locale: string, s: string, inSentence: boolean): string {
  if (locale === 'vi') return viNames(s, inSentence)
  if (locale === 'es' || locale === 'fr') return romanceNames(locale, s, inSentence)
  return s
}

/** A message's values: language names in the case each place needs. */
export function namedValues<V>(values: Record<string, V> | undefined, inside: ReadonlySet<string>, locale = 'vi'): Record<string, V> | undefined {
  if (!values) return values
  const out: Record<string, V> = {}
  for (const [k, v] of Object.entries(values)) out[k] = typeof v === 'string' ? (caseNames(locale, v, false) as V) : v
  for (const k of inside) {
    const v = values[k]
    if (v !== undefined) out[k + IN_SENTENCE] = typeof v === 'string' ? (caseNames(locale, v, true) as V) : v
  }
  return out
}
