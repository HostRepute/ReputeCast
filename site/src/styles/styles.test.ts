/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { join } from 'node:path'

const keyframes = (css: string) => new Set([...css.matchAll(/@keyframes\s+([\w-]+)/g)].map((m) => m[1]!))
const read = (name: string) => readFileSync(join(import.meta.dirname, name), 'utf8')

describe('the site stylesheet', () => {
  it("never reuses a keyframes name from Tailwind's theme (the minifier keeps only one)", () => {
    const require = createRequire(import.meta.url)
    const tailwind = keyframes(readFileSync(require.resolve('tailwindcss/theme.css'), 'utf8'))
    expect(tailwind.size).toBeGreaterThan(0)
    const ours = new Set([...keyframes(read('site.css')), ...keyframes(read('flow-pulse.css'))])
    expect([...ours].filter((k) => tailwind.has(k))).toEqual([])
  })

  it('turns every animation off for reduced motion and pauses loops with the visible control', () => {
    const css = read('site.css')
    expect(css).toContain('@media (prefers-reduced-motion: reduce)')
    expect(css).toMatch(/\[data-motion='paused'\] \*,[\s\S]*animation-play-state: paused !important/)
    // Pausing must never leave scroll reveals invisible.
    expect(css).toMatch(/\[data-motion='paused'\] :is\(\.reveal, \.grow i\)/)
  })

  it('does not load a border-box reset (the board uses the browser default)', () => {
    const css = read('site.css')
    expect(css).not.toContain('tailwindcss/preflight')
    expect(css).not.toContain("@import 'tailwindcss';")
    // No universal selector switches the box model; components that are
    // border-box on the board (.btn, .in, .wrap…) say so themselves.
    const rules = css.replace(/\/\*[\s\S]*?\*\//g, '')
    expect(rules).not.toMatch(/(^|,|\s)\*[^{,]*\{[^}]*box-sizing:\s*border-box/m)
  })
})
