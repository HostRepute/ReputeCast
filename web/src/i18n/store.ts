import type { Catalog } from './define'
import { PSEUDO_LOCALE } from './pseudo'
import { isRegionFormat, regionHourCycle, regionLocale, regionWeekStart, type HourCycle, type RegionFormat, type WeekStart } from './regions'

/** The languages the app's own interface comes in, and the pseudo-locale. */
export type AppLocale = 'en' | 'de' | 'es' | 'fr' | 'vi' | typeof PSEUDO_LOCALE

export interface I18nState {
  /** The app language: picks the translation and the plural rules. */
  readonly locale: AppLocale
  /** BCP 47 locale for numbers and dates inside messages (the region format). */
  readonly region: string
  /** The region format lib/format writes dates, times and numbers with. */
  readonly regionFormat: RegionFormat
  readonly hourCycle: HourCycle
  readonly weekStart: WeekStart
  readonly catalog: Catalog
}

type Loader = () => Promise<Catalog>

// Translations sit next to their English source, as `messages.<lang>.ts` or
// `<name>.messages.<lang>.ts`, and load only when the language is picked.
// So does the list of a language's files (catalogs/<lang>.ts): the first
// page load carries no language's translations, whichever it shows.
type Modules = Record<string, () => Promise<Catalog>>
const translations: Record<Exclude<AppLocale, 'en' | typeof PSEUDO_LOCALE>, () => Promise<{ default: Modules }>> = {
  de: () => import('./catalogs/de'),
  es: () => import('./catalogs/es'),
  fr: () => import('./catalogs/fr'),
  vi: () => import('./catalogs/vi'),
}

const loadAll =
  (list: () => Promise<{ default: Modules }>): Loader =>
  async () => {
    const { default: modules } = await list()
    return Object.assign({}, ...(await Promise.all(Object.values(modules).map((load) => load()))))
  }

const loaders = new Map<AppLocale, Loader>([
  ['en', async () => ({})],
  ...Object.entries(translations).map(([code, modules]) => [code as AppLocale, loadAll(modules)] as const),
  // The pseudo-locale transforms the English sources as they're formatted.
  [PSEUDO_LOCALE, async () => ({})],
])

let state: I18nState = { locale: 'en', region: 'en-US', regionFormat: 'standard', hourCycle: 'h23', weekStart: 'monday', catalog: {} }
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

/**
 * Sets the locale used for numbers and dates inside messages. A region
 * format's code ("vi-VN") also switches lib/format to it, with its clock
 * and first day of the week.
 */
export function setRegion(region: string): void {
  if (isRegionFormat(region)) {
    if (region === state.regionFormat && regionLocale(region) === state.region) return
    publish({ ...state, region: regionLocale(region), regionFormat: region, hourCycle: regionHourCycle(region), weekStart: regionWeekStart(region) })
    return
  }
  if (region !== state.region) publish({ ...state, region })
}

/** What a member reads ReputeCast in. Clock and week default to the region's. */
export interface Preferences {
  locale: AppLocale
  regionFormat: RegionFormat
  hourCycle?: HourCycle
  weekStart?: WeekStart
}

/** Applies a member's language and formats at once, loading translations first. */
export async function setPreferences(p: Preferences): Promise<void> {
  const ticket = ++latest
  const catalog = p.locale === state.locale ? state.catalog : await (loaders.get(p.locale) ?? loaders.get('en')!)()
  if (ticket !== latest) return
  const next: I18nState = {
    locale: p.locale,
    catalog,
    region: regionLocale(p.regionFormat),
    regionFormat: p.regionFormat,
    hourCycle: p.hourCycle ?? regionHourCycle(p.regionFormat),
    weekStart: p.weekStart ?? regionWeekStart(p.regionFormat),
  }
  const same = (Object.keys(next) as (keyof I18nState)[]).every((k) => next[k] === state[k])
  if (!same) publish(next)
}

/** Loads a language's translations without switching to it (a line in another language). */
export function loadCatalog(locale: AppLocale): Promise<Catalog> {
  return (loaders.get(locale) ?? loaders.get('en')!)()
}

/** Replaces how a language's translations load (tests, development locales). */
export function setCatalogLoader(locale: AppLocale, loader: Loader): void {
  loaders.set(locale, loader)
}

/** A key that changes whenever anything the screens format with changes. */
export function i18nKey(s: I18nState = state): string {
  return `${s.locale}|${s.regionFormat}|${s.hourCycle}|${s.weekStart}`
}
