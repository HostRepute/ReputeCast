import { AA_NORMAL, AAA_NORMAL, formatRatio, hex, isLarge, luminance, nearest, over, parseColor, passes, ratio, roundDown, suggestions, verdict, type RGBA } from './contrast'

// The same vectors as internal/accessibility/color_test.go: the picker's
// chip and the server's checks must agree on every number.
const c = (s: string): RGBA => {
  const v = parseColor(s)
  if (!v) throw new Error(`not a color: ${s}`)
  return v
}

describe('contrast, as the server computes it', () => {
  it('reads CSS colors', () => {
    const cases: Record<string, string> = {
      '#fff': '#FFFFFF',
      '#5ebfb0': '#5EBFB0',
      '#5EBFB0 !important': '#5EBFB0',
      'rgb(94, 191, 176)': '#5EBFB0',
      'rgb(94 191 176 / 50%)': '#5EBFB0',
      white: '#FFFFFF',
      'hsl(170.7, 43.1%, 55.9%)': '#5EBFB0',
      ReBeccaPurple: '#663399',
    }
    for (const [input, want] of Object.entries(cases)) expect(hex(c(input))).toBe(want)
    for (const bad of ['', 'inherit', 'currentColor', 'url(x.png)', '#12', 'rgb(1,2)']) expect(parseColor(bad)).toBeNull()
    expect(c('rgba(0,0,0,0.5)').a).toBe(0.5)
    expect(c('#00000080').a).toBeLessThan(1)
  })

  it('rounds ratios down', () => {
    const w = c('#FFFFFF')
    const cases: [string, string][] = [
      ['#5EBFB0', '2.1:1'],
      ['#0F766E', '5.4:1'],
      ['#348377', '4.5:1'],
      ['#276259', '7.0:1'],
      ['#2B9A8C', '3.4:1'],
      ['#000000', '21.0:1'],
      ['#FFFFFF', '1.0:1'],
    ]
    for (const [bg, want] of cases) expect(formatRatio(ratio(w, c(bg)))).toBe(want)
    expect(roundDown(4.4999)).toBe(4.4)
    expect(passes(4.4999, AA_NORMAL)).toBe(false)
  })

  it('finds the nearest passing shade', () => {
    const w = c('#FFFFFF')
    const aa = nearest(c('#5EBFB0'), w, AA_NORMAL)!
    expect([hex(aa.color), formatRatio(aa.ratio)]).toEqual(['#348377', '4.5:1'])
    const aaa = nearest(c('#5EBFB0'), w, AAA_NORMAL)!
    expect([hex(aaa.color), formatRatio(aaa.ratio)]).toEqual(['#276259', '7.0:1'])
    expect(hex(nearest(c('#0F766E'), w, AA_NORMAL)!.color)).toBe('#0F766E')
    expect(luminance(nearest(c('#9CA3AF'), w, AA_NORMAL)!.color)).toBeLessThan(luminance(c('#9CA3AF')))
    expect(nearest(c('#FFFFFF'), c('#777777'), 21)).toBeNull()
  })

  it('knows large text and alpha', () => {
    expect([isLarge(24, false), isLarge(18.67, true), isLarge(18.67, false), isLarge(18, true)]).toEqual([true, true, false, false])
    expect(hex(over({ r: 0, g: 0, b: 0, a: 0.5 }, c('#FFFFFF')))).toBe('#808080')
  })

  it('suggests what the board shows for “Plan your migration”', () => {
    // White text on #5EBFB0; the other buttons are #0F766E.
    const s = suggestions(c('#5EBFB0'), c('#FFFFFF'), AA_NORMAL, AAA_NORMAL, ['#0F766E', '#0F766E', '#5EBFB0'])
    expect(s).toEqual([
      { color: '#0F766E', ratio: 5.4, reason: 'other_buttons' },
      { color: '#348377', ratio: 4.5, reason: 'nearest_aa' },
      { color: '#276259', ratio: 7, reason: 'aaa' },
    ])
  })

  it('judges a size: 26 px bold headline text passes AA as large text', () => {
    const v = verdict(c('#2B9A8C'), c('#FFFFFF'), 26, true)
    expect([formatRatio(v.ratio), v.large, v.passesAA, v.passesAAA, v.aaNormal]).toEqual(['3.4:1', true, true, false, false])
  })
})
