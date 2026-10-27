import { describe, expect, it } from 'vitest'
import { applyBrand, fontStack } from './brand'

// CO-11: new blocks start from the brand kit, as starters do on the server
// (brandkit.Apply, same fixture as internal/brandkit's test).

const kit = {
  saved: true,
  colors: { button: '#4338CA', text: '#111111', background: '#F5F3FF', link: '#4338CA' },
  heading: { name: 'Source Serif 4', fallback: "Georgia, 'Times New Roman', serif" as const },
  body: { name: '', fallback: 'Arial, Helvetica, sans-serif' as const },
  button: { shape: 'square' as const, size: 15 as const },
}

const src =
  `<mj-section background-color="#F3F4F6"><mj-column><mj-text color="#1F2937"><h1 style="margin:0">Hi</h1></mj-text>` +
  `<mj-text color="#1F2937">See <a href="x" style="color:#0F766E">this</a></mj-text>` +
  `<mj-button background-color="#0F766E" font-family="'IBM Plex Sans', Helvetica, Arial, sans-serif" font-size="13px" border-radius="6px" inner-padding="9px 14px">Go</mj-button></mj-column></mj-section>`

describe('applyBrand', () => {
  it('puts the kit on content written in the default style', () => {
    const out = applyBrand(src, kit)
    for (const want of [
      'background-color="#F5F3FF"',
      'color="#111111"',
      'style="color:#4338CA"',
      'background-color="#4338CA"',
      'border-radius="0px"',
      'font-size="15px"',
      'inner-padding="14px 22px"',
      `font-family="'Source Serif 4', Georgia, 'Times New Roman', serif"`,
      'font-family="Arial, Helvetica, sans-serif"',
    ]) {
      expect(out).toContain(want)
    }
    expect(out).not.toContain('#0F766E')
  })

  it('leaves content alone without a saved kit', () => {
    expect(applyBrand(src, { ...kit, saved: false })).toBe(src)
    expect(applyBrand(src, undefined)).toBe(src)
  })

  it('writes font stacks with the web font first', () => {
    expect(fontStack({ name: 'Inter', fallback: 'Arial, Helvetica, sans-serif' })).toBe("'Inter', Arial, Helvetica, sans-serif")
    expect(fontStack({ name: ' ', fallback: 'Arial, Helvetica, sans-serif' })).toBe('Arial, Helvetica, sans-serif')
  })
})
