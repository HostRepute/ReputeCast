import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { build, palette, readTokens, themesCSS, tokensCSS } from './tokens.ts'

/**
 * The design tokens in both themes (M54): the generated stylesheets are up
 * to date, and the pairs the app draws meet WCAG 2.2 AA in light and dark
 * (design/design-system/README.md, "Accessibility").
 */

function luminance(hex: string): number {
  const h = hex.replace('#', '')
  const full = h.length === 3 ? [...h].map((c) => c + c).join('') : h
  const [r, g, b] = [0, 2, 4].map((i) => {
    const c = parseInt(full.slice(i, i + 2), 16) / 255
    return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r! + 0.7152 * g! + 0.0722 * b!
}

export function contrast(a: string, b: string): number {
  const [x, y] = [luminance(a), luminance(b)]
  return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05)
}

const t = readTokens()
const themes = ['light', 'dark'] as const

// Where text sits: every surface a screen paints.
const grounds = ['surface', 'ground', 'ground-2', 'ground-3', 'raised', 'field', 'side', 'hover', 'hover-soft']
// Text colours that may sit on any of them.
const text = ['ink', 'ink-2', 'ink-3', 'ink-nav', 'accent', 'ok-fg', 'warn-fg', 'bad-fg', 'info-fg', 'purple-fg', 'neutral-fg', 'bad-strong', 'info-strong', 'ok-ink', 'warn-ink', 'bad-ink', 'info-ink']
// Text on its own fill.
const pairs: [string, string][] = [
  ['ok-fg', 'ok-bg'],
  ['warn-fg', 'warn-bg'],
  ['bad-fg', 'bad-bg'],
  ['info-fg', 'info-bg'],
  ['neutral-fg', 'neutral-bg'],
  ['purple-fg', 'purple-bg'],
  ['ok-ink', 'ok-tint'],
  ['warn-ink', 'warn-tint'],
  ['warn-ink', 'warn-bg'],
  ['bad-ink', 'bad-tint'],
  ['bad-ink', 'bad-wash'],
  ['bad-fg', 'bad-tint'],
  ['info-ink', 'info-tint'],
  ['info-fg', 'info-tint'],
  ['accent', 'accent-soft'],
  ['accent', 'accent-tint'],
  ['ink', 'accent-soft'],
  ['ink', 'selection'],
  ['ink-3', 'skeleton'],
  ['placeholder', 'field'],
  ['muted-ink', 'surface'],
  ['muted-ink', 'field'],
  ['on-strong', 'accent'],
  ['on-strong', 'accent-hover'],
  ['on-strong', 'bad-strong'],
  ['on-strong', 'bad-fg'],
  ['on-strong', 'bad-bar'],
  ['on-strong', 'ok-bar'],
  ['on-strong', 'warn-bar'],
  ['on-strong', 'info-strong'],
  ['on-strong', 'inverse'],
  ['inverse-muted', 'inverse'],
  ['dark-text', 'code-bg'],
  ['dark-muted', 'code-bg'],
  ['dark-text', 'dark-surface'],
  // The engagement tier badges (audience.css).
  ['on-strong', 'chart-tier-1'],
  ['on-strong', 'chart-tier-2'],
  ['badge-tier-3-fg', 'chart-tier-3'],
  ['ink', 'badge-tier-4-bg'],
]
// Things that aren't text but must be seen (WCAG 1.4.11): control borders,
// the focus ring, bars, the switch, icons on inverse.
const nonText: [string, string][] = [
  ['line-input', 'field'],
  ['line-input', 'surface'],
  ['line-input', 'raised'],
  ['focus', 'surface'],
  ['focus', 'ground'],
  ['focus', 'raised'],
  ['switch-off', 'surface'],
  ['on-fixed', 'switch-off'],
  ['ok-bar', 'surface'],
  ['warn-bar', 'surface'],
  ['bad-bar', 'surface'],
  ['inverse-faint', 'inverse'],
]

describe('generated stylesheets', () => {
  it('tokens.css and themes.css are what tokens.json gives (run `pnpm tokens`)', () => {
    const out = build(t)
    expect(readFileSync(tokensCSS, 'utf8')).toBe(out.tokens)
    expect(readFileSync(themesCSS, 'utf8')).toBe(out.themes)
  })

  it('the dark theme names every colour the light theme has, and only real colours', () => {
    for (const theme of themes) {
      const p = palette(t, theme)
      for (const [name, v] of Object.entries(p)) expect(v, `${theme} ${name}`).toMatch(/^(#[0-9a-f]{6}|#[0-9a-f]{3}|rgba?\([\d.,\s]+\)|\d+ \d+ \d+)$/i)
    }
  })
})

describe.each(themes)('%s theme contrast', (theme) => {
  const p = palette(t, theme)
  const hex = (name: string) => {
    const v = p[name]
    if (!v || !v.startsWith('#')) throw new Error(`${name} is not a hex colour in ${theme}`)
    return v
  }

  it('text tokens are 4.5:1 on every surface', () => {
    const fails = text.flatMap((a) => grounds.map((b) => [a, b, contrast(hex(a), hex(b))] as const)).filter(([, , r]) => r < 4.5)
    expect(fails.map(([a, b, r]) => `${a} on ${b}: ${r.toFixed(2)}`)).toEqual([])
  })

  it('text is 4.5:1 on its own fill', () => {
    const fails = pairs.map(([a, b]) => [a, b, contrast(hex(a), hex(b))] as const).filter(([, , r]) => r < 4.5)
    expect(fails.map(([a, b, r]) => `${a} on ${b}: ${r.toFixed(2)}`)).toEqual([])
  })

  it('borders, bars, the focus ring and the switch are 3:1', () => {
    const fails = nonText.map(([a, b]) => [a, b, contrast(hex(a), hex(b))] as const).filter(([, , r]) => r < 3)
    expect(fails.map(([a, b, r]) => `${a} on ${b}: ${r.toFixed(2)}`)).toEqual([])
  })
})

describe('dark theme charts', () => {
  // The light chart palette is the boards' (Sunset and machine opens are
  // drawn faint on purpose); the dark one was picked so every series is
  // 3:1 on cards and on its track.
  const p = palette(t, 'dark')
  const series = ['chart-gmail', 'chart-outlook', 'chart-yahoo', 'chart-icloud', 'chart-other', 'chart-ses', 'chart-postal', 'chart-postmark', 'chart-tier-1', 'chart-tier-2', 'chart-tier-3', 'chart-tier-4', 'chart-human', 'chart-machine', 'chart-score-1', 'chart-score-2', 'chart-score-3', 'chart-spam']
  it('every series is 3:1 on cards and on the empty track', () => {
    const fails = series.flatMap((s) => ['surface', 'chart-track'].map((b) => [s, b, contrast(p[s]!, p[b]!)] as const)).filter(([, , r]) => r < 3)
    expect(fails.map(([a, b, r]) => `${a} on ${b}: ${r.toFixed(2)}`)).toEqual([])
  })
  it('the engagement tiers step from light (Active) to deep (Sunset), and the score bands from deep (Cool) to light (Hot)', () => {
    const tiers = ['chart-tier-1', 'chart-tier-2', 'chart-tier-3', 'chart-tier-4'].map((n) => luminance(p[n]!))
    for (let i = 1; i < tiers.length; i++) expect(tiers[i]!).toBeLessThan(tiers[i - 1]!)
    const bands = ['chart-score-1', 'chart-score-2', 'chart-score-3'].map((n) => luminance(p[n]!))
    for (let i = 1; i < bands.length; i++) expect(bands[i]!).toBeGreaterThan(bands[i - 1]!)
  })
})
