import { describe, expect, it } from 'vitest'
import { contrast, grade, ratioText } from './contrast'

describe('contrast ratios', () => {
  it('read as the BrandKit board shows them', () => {
    expect(ratioText(contrast('#FFFFFF', '#4338CA'))).toBe('7.9')
    expect(ratioText(contrast('#1F2937', '#FFFFFF'))).toBe('14.7')
    expect(ratioText(contrast('#1F2937', '#F5F3FF'))).toBe('13.4')
    expect(grade(contrast('#1F2937', '#F5F3FF'))).toBe('AAA')
  })

  it('never round up past a WCAG threshold', () => {
    expect(ratioText(4.47)).toBe('4.4')
    expect(ratioText(6.96)).toBe('6.9')
    expect(ratioText(2.97)).toBe('2.9')
    expect(ratioText(4.52)).toBe('4.5')
    expect(ratioText(7.04)).toBe('7.0')
  })
})
