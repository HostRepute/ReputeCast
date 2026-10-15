import type { QueryClient } from '@tanstack/react-query'
import { http } from '@/api/client'
import { pickableLanguages, resolvePreferences, resolveVisitor, type InstanceDefaults, type MemberChoices } from './languages'
import { getI18n, setPreferences, type Preferences } from './store'

/**
 * Keeps the app in the right language and formats:
 * - before the first render, from what this browser last used, so nothing
 *   flashes in another language;
 * - for a signed-in member, from their choices in /me;
 * - on the sign-in, invitation and password pages, from the language picked
 *   in their menu, the invitation, the browser or the instance default.
 */

const APPLIED_KEY = 'rc.i18n'
const VISITOR_KEY = 'rc.language'
const DEFAULTS_KEY = 'rc.i18n.defaults'

function read<T>(key: string): T | undefined {
  try {
    const raw = window.localStorage.getItem(key)
    return raw ? (JSON.parse(raw) as T) : undefined
  } catch {
    return undefined
  }
}

function write(key: string, value: unknown) {
  try {
    if (value === undefined) window.localStorage.removeItem(key)
    else window.localStorage.setItem(key, JSON.stringify(value))
  } catch {
    // Private windows and blocked storage: the language still applies to this page.
  }
}

/** The pages people see before they sign in. */
export function isVisitorPath(path: string): boolean {
  return /^\/(sign-in|invite\/|reset-password)/.test(path)
}

interface MeLike {
  user?: { preferences?: MemberChoices }
  instance?: InstanceDefaults & { languages?: string[] }
}

interface MetaLike extends InstanceDefaults {
  languages?: string[]
}

let mode: 'member' | 'visitor' = 'visitor'
let meta: MetaLike | undefined = read<MetaLike>(DEFAULTS_KEY)
let pageLanguage: string | undefined

function visitorPreferences(): Preferences {
  return resolveVisitor({
    chosen: read<string>(VISITOR_KEY),
    page: pageLanguage,
    inst: meta,
    available: pickableLanguages({ serverLanguages: meta?.languages }),
  })
}

async function apply(p: Preferences) {
  await setPreferences(p)
  if (mode === 'member') write(APPLIED_KEY, p)
}

/** Applies the language before the first render. */
export async function bootI18n(path = typeof location === 'undefined' ? '/' : location.pathname): Promise<void> {
  const last = read<Preferences>(APPLIED_KEY)
  if (!isVisitorPath(path) && last?.locale) {
    mode = 'member'
    await setPreferences(last)
    return
  }
  mode = 'visitor'
  await setPreferences(visitorPreferences())
}

/** Applies a signed-in member's language from /me. */
export function applyMember(me: MeLike): Promise<void> {
  mode = 'member'
  if (me.instance) {
    meta = { defaultLanguage: me.instance.defaultLanguage, defaultRegionFormat: me.instance.defaultRegionFormat, languages: me.instance.languages }
    write(DEFAULTS_KEY, meta)
  }
  return apply(resolvePreferences(me.user?.preferences, me.instance))
}

/** Applies the visitor's language once the instance defaults (/meta) are known. */
export function applyVisitor(next?: MetaLike): Promise<void> {
  if (next) {
    meta = { defaultLanguage: next.defaultLanguage, defaultRegionFormat: next.defaultRegionFormat, languages: next.languages }
    write(DEFAULTS_KEY, meta)
  }
  if (mode === 'member' && typeof location !== 'undefined' && !isVisitorPath(location.pathname)) return Promise.resolve()
  mode = 'visitor'
  return setPreferences(visitorPreferences())
}

/** The language picked in the menu at the bottom of the sign-in pages. */
export function chooseVisitorLanguage(locale: string): Promise<void> {
  write(VISITOR_KEY, locale)
  mode = 'visitor'
  return setPreferences(visitorPreferences())
}

/** The language the visitor picked in the menu, if any. */
export function visitorChoice(): string | undefined {
  return read<string>(VISITOR_KEY)
}

/** A page's own language, such as an invitation's: used unless the visitor picked one. */
export function setPageLanguage(locale: string | undefined): Promise<void> {
  if (locale === pageLanguage) return Promise.resolve()
  pageLanguage = locale
  if (mode !== 'visitor') return Promise.resolve()
  return setPreferences(visitorPreferences())
}

/** Forgets the member's language after signing out. */
export function forgetMember(): void {
  write(APPLIED_KEY, undefined)
  mode = 'visitor'
}

let connected = false

/**
 * Follows /me and /meta in the query cache, and tells the API which
 * language to answer in (errors, "who can help").
 */
export function connectI18n(qc: QueryClient): void {
  if (connected) return
  connected = true
  http.use({
    onRequest({ request }) {
      if (!request.headers.has('Accept-Language')) request.headers.set('Accept-Language', getI18n().locale)
      return request
    },
  })
  qc.getQueryCache().subscribe((event) => {
    if (event.type !== 'updated' || event.action.type !== 'success') return
    const key = event.query.queryKey[0]
    const data = event.query.state.data as unknown
    if (key === 'me') {
      if (data) void applyMember(data as MeLike)
      else if (mode === 'member') {
        forgetMember()
        void applyVisitor()
      }
    } else if (key === 'meta' && data) {
      void applyVisitor(data as MetaLike)
    }
  })
}
