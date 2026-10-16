import { PSEUDO_LOCALE } from './pseudo'
import { REGION_FORMATS, regionHourCycle, regionName, regionWeekStart, type HourCycle, type RegionFormat, type WeekStart } from './regions'
import { COMPLETE_LANGUAGES } from './status.gen'
import type { AppLocale, Preferences } from './store'

export interface AppLanguage {
  readonly code: AppLocale
  /** The language's name in itself, in every interface language. */
  readonly name: string
}

/** The languages ReputeCast's interface is translated into, in picker order. */
export const APP_LANGUAGES: readonly AppLanguage[] = [
  { code: 'en', name: 'English' },
  { code: 'vi', name: 'Tiếng Việt' },
]

export const PSEUDO_LANGUAGE: AppLanguage = { code: PSEUDO_LOCALE, name: 'Pseudo (+40%)' }

export function languageName(code: string): string {
  if (code === PSEUDO_LOCALE) return PSEUDO_LANGUAGE.name
  return APP_LANGUAGES.find((l) => l.code === code)?.name ?? code
}

export function isAppLocale(v: unknown): v is AppLocale {
  return v === PSEUDO_LOCALE || APP_LANGUAGES.some((l) => l.code === v)
}

/**
 * The languages that can be picked. A language shows once its translation
 * is complete for this release, as `pnpm run i18n:check` found when the app
 * was built (status.gen.ts); in development every language shows. The
 * pseudo-locale shows when the server has it on (development, or
 * i18n.pseudo_locale).
 */
export function pickableLanguages(opts: { serverLanguages?: readonly string[]; pseudo?: boolean } = {}): AppLanguage[] {
  const server = opts.serverLanguages
  const out = APP_LANGUAGES.filter((l) => {
    if (server && !server.includes(l.code)) return false
    return l.code === 'en' || import.meta.env.DEV || COMPLETE_LANGUAGES.includes(l.code)
  })
  if (opts.pseudo && (!server || server.includes(PSEUDO_LOCALE))) out.push(PSEUDO_LANGUAGE)
  return out
}

/** Every region format with its name in its own language, in picker order. */
export function regionOptions(): { code: RegionFormat; name: string }[] {
  return REGION_FORMATS.map((code) => ({ code, name: regionName(code) }))
}

/** The region format a language follows by default. */
export function defaultRegionFor(locale: AppLocale): RegionFormat {
  return locale === 'vi' ? 'vi-VN' : 'standard'
}

/**
 * The browser's preferred language when the app has it: navigator.languages
 * in order, matched by base language ("vi-VN" is Vietnamese).
 */
export function browserLanguage(available: readonly AppLanguage[], languages: readonly string[] = typeof navigator === 'undefined' ? [] : navigator.languages ?? [navigator.language]): AppLocale | undefined {
  for (const tag of languages) {
    const base = tag.toLowerCase().split(/[-_]/)[0]
    const hit = available.find((l) => l.code !== PSEUDO_LOCALE && l.code === base)
    if (hit) return hit.code
  }
  return undefined
}

/** A member's choices as the API stores them: "default" and "auto" follow. */
export interface MemberChoices {
  language: string
  regionFormat: string
  hourCycle: string
  weekStart: string
}

export interface InstanceDefaults {
  defaultLanguage: string
  defaultRegionFormat: string
}

function asLocale(v: string | undefined, fallback: AppLocale): AppLocale {
  return isAppLocale(v) ? v : fallback
}

function asRegion(v: string | undefined): RegionFormat | undefined {
  return REGION_FORMATS.includes(v as RegionFormat) ? (v as RegionFormat) : undefined
}

/**
 * How a member reads ReputeCast: their own choices where they made them,
 * else the instance defaults. A region they didn't choose follows their
 * language, or the instance's default region when the language is the
 * instance's too. internal/i18n.Resolve does the same on the server.
 */
export function resolvePreferences(choices: MemberChoices | undefined, inst: InstanceDefaults | undefined): Required<Preferences> {
  const defLocale = asLocale(inst?.defaultLanguage, 'en')
  const defRegion = asRegion(inst?.defaultRegionFormat) ?? defaultRegionFor(defLocale)
  const chose = choices && isAppLocale(choices.language)
  const locale = chose ? (choices.language as AppLocale) : defLocale
  let regionFormat = asRegion(choices?.regionFormat) ?? (chose ? defaultRegionFor(locale) : defRegion)
  // The pseudo-locale uses the ReputeCast standard formats.
  if (locale === PSEUDO_LOCALE) regionFormat = 'standard'
  const hourCycle: HourCycle = choices?.hourCycle === 'h12' || choices?.hourCycle === 'h23' ? choices.hourCycle : regionHourCycle(regionFormat)
  const weekStart: WeekStart = choices?.weekStart === 'monday' || choices?.weekStart === 'sunday' ? choices.weekStart : regionWeekStart(regionFormat)
  return { locale, regionFormat, hourCycle, weekStart }
}

/**
 * How the sign-in, invitation and password pages read: the language picked
 * in their menu, else the page's own (an invitation's language), else the
 * browser's when the app has it, else the instance default.
 */
export function resolveVisitor(opts: { chosen?: string; page?: string; inst?: InstanceDefaults; available: readonly AppLanguage[]; browser?: readonly string[] }): Required<Preferences> {
  const defLocale = asLocale(opts.inst?.defaultLanguage, 'en')
  const has = (v?: string): v is AppLocale => !!v && opts.available.some((l) => l.code === v)
  const locale: AppLocale = has(opts.chosen) ? opts.chosen : has(opts.page) ? opts.page : (browserLanguage(opts.available, opts.browser) ?? defLocale)
  const regionFormat = locale === defLocale ? (asRegion(opts.inst?.defaultRegionFormat) ?? defaultRegionFor(locale)) : defaultRegionFor(locale)
  return { locale, regionFormat: locale === PSEUDO_LOCALE ? 'standard' : regionFormat, hourCycle: regionHourCycle(regionFormat), weekStart: regionWeekStart(regionFormat) }
}
