import type { Catalog } from './define'

/** The languages the app's own interface comes in. */
export type AppLocale = 'en' | 'vi'

export interface I18nState {
  /** The app language: picks the translation and the plural rules. */
  readonly locale: AppLocale
  /** BCP 47 locale for numbers and dates inside messages (the region format). */
  readonly region: string
  readonly catalog: Catalog
}

type Loader = () => Promise<Catalog>

// Translations sit next to their English source, as `messages.vi.ts` or
// `<name>.messages.vi.ts`, and load only when the language is picked.
const vi = import.meta.glob<Catalog>('../**/*messages.vi.ts', { import: 'default' })

const loaders = new Map<AppLocale, Loader>([
  ['en', async () => ({})],
  ['vi', async () => Object.assign({}, ...(await Promise.all(Object.values(vi).map((load) => load()))))],
])

let state: I18nState = { locale: 'en', region: 'en-US', catalog: {} }
const listeners = new Set<() => void>()
let latest = 0

export function getI18n(): I18nState {
  return state
}

export function subscribeI18n(listener: () => void): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

function publish(next: I18nState) {
  state = next
  if (typeof document !== 'undefined') document.documentElement.lang = next.locale
  for (const l of listeners) l()
}

/** Switches the app language, loading its translations first. */
export async function setLocale(locale: AppLocale): Promise<void> {
  const ticket = ++latest
  const catalog = await (loaders.get(locale) ?? loaders.get('en')!)()
  if (ticket !== latest) return
  publish({ ...state, locale, catalog })
}

/** Sets the locale used for numbers and dates inside messages. */
export function setRegion(region: string): void {
  if (region !== state.region) publish({ ...state, region })
}

/** Replaces how a language's translations load (tests, development locales). */
export function setCatalogLoader(locale: AppLocale, loader: Loader): void {
  loaders.set(locale, loader)
}
