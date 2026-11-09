/**
 * The app's light or dark theme (M54, Account & security › Appearance).
 *
 * A member chooses Same as device (the default), Light or Dark. The choice
 * is saved on their account (PUT /me/appearance, /me's user.theme) so it
 * follows them to every browser, and kept in this browser too, so the very
 * first paint is already right: src/early.ts applies it before the app's
 * scripts load, with no request.
 *
 * The theme is html[data-theme] ('light' or 'dark'); styles/themes.css
 * (generated from the design tokens) holds the dark values. Before that
 * attribute is set, CSS follows the device on its own.
 *
 * Kept free of React and the API client, so src/early.ts can import it.
 */

export type ThemeChoice = 'system' | 'light' | 'dark'
export type Theme = 'light' | 'dark'

export const THEME_CHOICES: readonly ThemeChoice[] = ['system', 'light', 'dark']

const KEY = 'rc.theme'
const DARK = '(prefers-color-scheme: dark)'

export function isThemeChoice(v: unknown): v is ThemeChoice {
  return v === 'system' || v === 'light' || v === 'dark'
}

function stored(): ThemeChoice {
  try {
    const v = window.localStorage.getItem(KEY)
    return isThemeChoice(v) ? v : 'system'
  } catch {
    // Private windows and blocked storage: follow the device.
    return 'system'
  }
}

function store(choice: ThemeChoice) {
  try {
    if (choice === 'system') window.localStorage.removeItem(KEY)
    else window.localStorage.setItem(KEY, choice)
  } catch {
    // The theme still applies to this page.
  }
}

function media(): MediaQueryList | undefined {
  return typeof window !== 'undefined' && typeof window.matchMedia === 'function' ? window.matchMedia(DARK) : undefined
}

/** The device's own setting. */
export function deviceTheme(): Theme {
  return media()?.matches ? 'dark' : 'light'
}

/** The theme a choice comes to on this device. */
export function resolveTheme(choice: ThemeChoice): Theme {
  return choice === 'system' ? deviceTheme() : choice
}

let choice: ThemeChoice = typeof window === 'undefined' ? 'system' : stored()
const listeners = new Set<() => void>()
let snapshot = { choice, theme: resolveTheme(choice) }

function notify() {
  const next = { choice, theme: resolveTheme(choice) }
  if (next.choice === snapshot.choice && next.theme === snapshot.theme) return
  snapshot = next
  for (const l of listeners) l()
}

/**
 * Sets html[data-theme] and the browser's toolbar colour. Transitions are
 * held for a frame, so a theme change swaps every colour at once instead of
 * fading switches and buttons one by one.
 */
export function applyTheme(theme: Theme = resolveTheme(choice)): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  const changing = root.dataset.theme !== undefined && root.dataset.theme !== theme
  if (changing) root.setAttribute('data-theme-switching', '')
  root.dataset.theme = theme
  // The toolbar takes the page's ground in dark, the brand teal in light
  // (index.html's colour), once the stylesheet is there to read them from.
  const css = getComputedStyle(root)
  const colour = (theme === 'dark' ? css.getPropertyValue('--ground') : css.getPropertyValue('--accent')).trim()
  if (colour) for (const m of document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]')) m.content = colour
  if (changing) {
    const done = () => root.removeAttribute('data-theme-switching')
    if (typeof requestAnimationFrame === 'function') requestAnimationFrame(() => requestAnimationFrame(done))
    else done()
  }
}

/** The member's choice and what it comes to; for useSyncExternalStore. */
export function getThemeSnapshot(): { choice: ThemeChoice; theme: Theme } {
  return snapshot
}

export function subscribeTheme(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

/** Switches the theme at once and keeps the choice in this browser. */
export function setThemeChoice(next: ThemeChoice): void {
  choice = next
  store(next)
  applyTheme()
  notify()
}

let started = false

/**
 * Applies this browser's choice and keeps following: the device switching
 * between light and dark (for Same as device) and another tab changing the
 * choice. Safe to call more than once.
 */
export function startTheme(): void {
  if (typeof window === 'undefined') return
  applyTheme()
  if (started) return
  started = true
  media()?.addEventListener('change', () => {
    if (choice !== 'system') return
    applyTheme()
    notify()
  })
  window.addEventListener('storage', (e) => {
    if (e.key !== KEY && e.key !== null) return
    choice = stored()
    applyTheme()
    notify()
  })
}

/** Forgets the choice kept in this browser: the next page follows the device. Test helper. */
export function resetThemeForTests(): void {
  store('system')
  choice = 'system'
  snapshot = { choice, theme: resolveTheme(choice) }
  if (typeof document !== 'undefined') {
    delete document.documentElement.dataset.theme
    document.documentElement.removeAttribute('data-theme-switching')
  }
}
