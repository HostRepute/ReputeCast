import { getI18n } from './store'

/**
 * Whether a language keeps a word's capital inside a sentence: German
 * capitalises every noun ("Vorname"), where English and the other app
 * languages write a label in lower case there ("first name").
 * internal/i18n.KeepsCapitals is the server's side.
 */
export function keepsCapitals(locale: string = getI18n().locale): boolean {
  return locale === 'de'
}

/** A label put inside a sentence: in lower case ("first name"), unless the language keeps capitals. */
export function lowerInSentence(s: string, locale?: string): string {
  return keepsCapitals(locale) ? s : s.toLowerCase()
}

/** A phrase put inside a sentence: its first letter in lower case, unless the language keeps capitals. */
export function lowerFirstInSentence(s: string, locale?: string): string {
  if (!s || keepsCapitals(locale)) return s
  return s.charAt(0).toLowerCase() + s.slice(1)
}
