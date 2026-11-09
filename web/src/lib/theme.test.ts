import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getThemeSnapshot, resetThemeForTests, resolveTheme, setThemeChoice, startTheme, subscribeTheme } from './theme'

/** A device whose light or dark setting the test flips. */
function device(dark: boolean) {
  const listeners = new Set<() => void>()
  const list = {
    matches: dark,
    media: '(prefers-color-scheme: dark)',
    addEventListener: (_: string, l: () => void) => listeners.add(l),
    removeEventListener: (_: string, l: () => void) => listeners.delete(l),
  }
  vi.spyOn(window, 'matchMedia').mockImplementation(() => list as unknown as MediaQueryList)
  return {
    set(next: boolean) {
      list.matches = next
      for (const l of listeners) l()
    },
  }
}

const html = () => document.documentElement

describe('theme', () => {
  beforeEach(() => resetThemeForTests())
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    Reflect.deleteProperty(window, '__rcEarly')
  })

  it('follows the device until the member chooses', () => {
    device(true)
    expect(resolveTheme('system')).toBe('dark')
    expect(resolveTheme('light')).toBe('light')
    setThemeChoice('system')
    expect(html().dataset.theme).toBe('dark')
    expect(localStorage.getItem('rc.theme')).toBeNull()
  })

  it('applies a choice at once and keeps it in this browser', () => {
    device(false)
    const seen: string[] = []
    const stop = subscribeTheme(() => seen.push(`${getThemeSnapshot().choice}/${getThemeSnapshot().theme}`))
    setThemeChoice('dark')
    expect(html().dataset.theme).toBe('dark')
    expect(localStorage.getItem('rc.theme')).toBe('dark')
    setThemeChoice('light')
    expect(html().dataset.theme).toBe('light')
    setThemeChoice('system')
    expect(localStorage.getItem('rc.theme')).toBeNull()
    stop()
    expect(seen).toEqual(['dark/dark', 'light/light', 'system/light'])
  })

  it('holds transitions while the colours swap', () => {
    device(false)
    setThemeChoice('light')
    setThemeChoice('dark')
    expect(html().hasAttribute('data-theme-switching')).toBe(true)
  })

  it('follows the device switching while on Same as device, and only then', () => {
    const d = device(false)
    startTheme()
    setThemeChoice('system')
    d.set(true)
    expect(html().dataset.theme).toBe('dark')
    expect(getThemeSnapshot()).toEqual({ choice: 'system', theme: 'dark' })
    setThemeChoice('light')
    d.set(false)
    d.set(true)
    expect(html().dataset.theme).toBe('light')
  })

  it('follows a choice made in another tab', () => {
    device(false)
    startTheme()
    setThemeChoice('light')
    localStorage.setItem('rc.theme', 'dark')
    window.dispatchEvent(new StorageEvent('storage', { key: 'rc.theme', newValue: 'dark' }))
    expect(getThemeSnapshot().choice).toBe('dark')
    expect(html().dataset.theme).toBe('dark')
  })

  // src/early.ts runs before the app and can't import this module: it must
  // come to the same theme from the same kept choice and device.
  it.each([
    ['dark', false, 'dark'],
    ['light', true, 'light'],
    [null, true, 'dark'],
    [null, false, 'light'],
  ] as const)('src/early.ts sets the theme before the app loads (kept %s, dark device %s)', async (kept, dark, want) => {
    device(dark)
    if (kept) localStorage.setItem('rc.theme', kept)
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response(null, { status: 401 }))))
    // early.ts does its work when it loads: a fresh import per case runs it again.
    vi.resetModules()
    await import('../early')
    expect(html().dataset.theme).toBe(want)
    expect(resolveTheme(kept ?? 'system')).toBe(want)
  })
})
