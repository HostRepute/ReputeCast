import { isPluralElement, isSelectElement, isTagElement, type MessageFormatElement } from '@formatjs/icu-messageformat-parser'
import { IntlMessageFormat, type Formatters, type PrimitiveType } from 'intl-messageformat'
import { Fragment, cloneElement, isValidElement, type ReactNode } from 'react'
import type { MessageDescriptor } from './define'
import { PSEUDO_LOCALE, pseudoSource, pseudoWrap } from './pseudo'
import type { I18nState } from './store'

/** Values for a plain-text message. */
export type Values = Record<string, PrimitiveType>

/** Values for a rich message: React nodes, and a function per tag. */
export type RichValues = Record<string, PrimitiveType | ReactNode | ((chunks: ReactNode[]) => ReactNode)>

const numberFormats = new Map<string, Intl.NumberFormat>()
const dateFormats = new Map<string, Intl.DateTimeFormat>()

// Numbers and dates follow the region format; plural rules follow the
// language of the text (an English fallback keeps English plurals).
function formatters(region: string): Formatters {
  return {
    getNumberFormat: (_locales, opts) => {
      const key = `${region}|${JSON.stringify(opts ?? {})}`
      let f = numberFormats.get(key)
      if (!f) {
        f = new Intl.NumberFormat(region, opts as Intl.NumberFormatOptions)
        numberFormats.set(key, f)
      }
      return f
    },
    getDateTimeFormat: (_locales, opts) => {
      const key = `${region}|${JSON.stringify(opts ?? {})}`
      let f = dateFormats.get(key)
      if (!f) {
        f = new Intl.DateTimeFormat(region, opts)
        dateFormats.set(key, f)
      }
      return f
    },
    getPluralRules: (locales, opts) => new Intl.PluralRules(locales, opts),
  }
}

interface Compiled {
  readonly mf: IntlMessageFormat
  /** Tags in the message, rendered as their text when no function is given. */
  readonly tags: Readonly<Record<string, (chunks: unknown[]) => unknown>>
}

const compiled = new Map<string, Compiled | null>()

function collectTags(els: MessageFormatElement[], into: Set<string>) {
  for (const el of els) {
    if (isTagElement(el)) {
      into.add(el.value)
      collectTags(el.children, into)
    } else if (isPluralElement(el) || isSelectElement(el)) {
      for (const opt of Object.values(el.options)) collectTags(opt.value, into)
    }
  }
}

function compile(text: string, locale: string, region: string): Compiled | null {
  const key = `${locale}\u0000${region}\u0000${text}`
  let c = compiled.get(key)
  if (c === undefined) {
    try {
      const mf = new IntlMessageFormat(text, locale, undefined, { formatters: formatters(region) })
      const names = new Set<string>()
      collectTags(mf.getAst(), names)
      const tags: Record<string, (chunks: unknown[]) => unknown> = {}
      for (const n of names) tags[n] = (chunks) => chunks
      c = { mf, tags }
    } catch (err) {
      if (import.meta.env.DEV) console.error(`i18n: can't parse "${text}"`, err)
      c = null
    }
    compiled.set(key, c)
  }
  return c
}

function resolve(state: I18nState, d: MessageDescriptor): { text: string; locale: string; pseudo?: boolean } {
  // The pseudo-locale accents the English source; formatting brackets it.
  if (state.locale === PSEUDO_LOCALE) return { text: pseudoSource(d.message), locale: 'en', pseudo: true }
  const translated = state.catalog[d.id]
  return translated === undefined ? { text: d.message, locale: 'en' } : { text: translated, locale: state.locale }
}

function plain(state: I18nState, d: MessageDescriptor, text: string, locale: string, values?: Values): string {
  const c = compile(text, locale, state.region)
  if (!c) return text
  try {
    const out = c.mf.format<unknown>({ ...c.tags, ...values } as Record<string, PrimitiveType>)
    return Array.isArray(out) ? out.flat(Infinity).join('') : String(out)
  } catch (err) {
    if (import.meta.env.DEV) console.error(`i18n: can't format ${d.id}`, err)
    return text
  }
}

/** Formats a message as plain text. Tags keep their text and lose their markup. */
export function formatText(state: I18nState, d: MessageDescriptor, values?: Values): string {
  const { text, locale, pseudo } = resolve(state, d)
  const out = plain(state, d, text, locale, values)
  if (!pseudo) return out
  const [open, close] = pseudoWrap(out)
  return open + out + close
}

/** Rich values as text, to measure a pseudo-locale string: React nodes count as nothing. */
function textValues(values?: RichValues): Values {
  const out: Values = {}
  for (const [k, v] of Object.entries(values ?? {})) {
    if (typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean' || v === null || v === undefined || v instanceof Date) out[k] = v
    else if (typeof v !== 'function') out[k] = ''
  }
  return out
}

function keyed(nodes: ReactNode[]): ReactNode[] {
  return nodes.map((n, i) => (isValidElement(n) && n.key == null ? cloneElement(n, { key: i }) : n))
}

/** Formats a message with React nodes and tag functions. */
export function formatRich(state: I18nState, d: MessageDescriptor, values?: RichValues): ReactNode {
  const { text, locale, pseudo } = resolve(state, d)
  const c = compile(text, locale, state.region)
  if (!c) return text
  const vals: Record<string, unknown> = { ...c.tags }
  for (const [k, v] of Object.entries(values ?? {})) {
    vals[k] = typeof v === 'function' ? (chunks: ReactNode[]) => (v as (chunks: ReactNode[]) => ReactNode)(keyed(chunks)) : v
  }
  try {
    const out = c.mf.format<ReactNode>(vals as Record<string, PrimitiveType | ReactNode>)
    const parts = Array.isArray(out) ? out : [out]
    if (pseudo) {
      const [open, close] = pseudoWrap(plain(state, d, text, locale, textValues(values)))
      parts.unshift(open)
      parts.push(close)
    }
    if (parts.length === 1) return parts[0]
    return parts.map((part, i) => <Fragment key={i}>{part}</Fragment>)
  } catch (err) {
    if (import.meta.env.DEV) console.error(`i18n: can't format ${d.id}`, err)
    return text
  }
}
