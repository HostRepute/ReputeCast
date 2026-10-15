import { useCallback, useSyncExternalStore, type ReactNode } from 'react'
import type { MessageDescriptor } from './define'
import { formatRich, formatText, type RichValues, type Values } from './format'
import { getI18n, subscribeI18n, type I18nState } from './store'

export { defineMessages, sourceMessages, translate, type Catalog, type MessageDescriptor, type Messages } from './define'
export type { RichValues, Values } from './format'
export { getI18n, setCatalogLoader, setLocale, setRegion, subscribeI18n, type AppLocale, type I18nState } from './store'

/** The current language, region format and translations. */
export function useI18n(): I18nState {
  return useSyncExternalStore(subscribeI18n, getI18n, getI18n)
}

/** Formats messages as plain text, re-rendering when the language changes. */
export function useT(): (d: MessageDescriptor, values?: Values) => string {
  const state = useI18n()
  return useCallback((d: MessageDescriptor, values?: Values) => formatText(state, d, values), [state])
}

/**
 * A message with markup or React values:
 *
 *   <T msg={m.sentTo} values={{ count: 3, b: (chunks) => <b>{chunks}</b> }} />
 */
export function T({ msg, values }: { msg: MessageDescriptor; values?: RichValues }): ReactNode {
  return formatRich(useI18n(), msg, values)
}

/** Formats a message outside React (toasts, document titles, formatters). */
export function t(d: MessageDescriptor, values?: Values): string {
  return formatText(getI18n(), d, values)
}

const collators = new Map<string, Intl.Collator>()

/** Compares text by the app language's sorting rules. */
export function compareText(a: string, b: string): number {
  const { locale } = getI18n()
  let c = collators.get(locale)
  if (!c) {
    c = new Intl.Collator(locale, { sensitivity: 'base', numeric: true })
    collators.set(locale, c)
  }
  return c.compare(a, b)
}

const listFormats = new Map<string, Intl.ListFormat>()

/** Joins items as the app language does: "A, B and C", or "A, B or C". */
export function formatList(items: readonly string[], type: 'conjunction' | 'disjunction' = 'conjunction'): string {
  const key = `${getI18n().locale}|${type}`
  let f = listFormats.get(key)
  if (!f) {
    // English lists follow the house style, without the serial comma.
    f = new Intl.ListFormat(getI18n().locale === 'en' ? 'en-GB' : getI18n().locale, { style: 'long', type })
    listFormats.set(key, f)
  }
  return f.format(items)
}
