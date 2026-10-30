import { describe, expect, it } from 'vitest'
import { BLOCK_SPACING, BUTTON_CORNERS, BUTTON_SIZES, COLUMN_SPACING, LINE_LENGTHS, LINE_WEIGHTS, evenPreset, lengthPreset, presetOf, sides, verticalPreset } from './stylePresets'

// R4.U3: the Style panel's presets match what a block holds, however it's written.

describe('style presets', () => {
  it('reads box values the way MJML expands them', () => {
    expect(sides('10px 25px')).toEqual([10, 25, 10, 25])
    expect(sides('8px')).toEqual([8, 8, 8, 8])
    expect(sides('4px 25px 6px')).toEqual([4, 25, 6, 25])
    expect(sides('1em 2em')).toBeNull()
  })

  it('finds the preset a value is, written either way', () => {
    expect(presetOf(BUTTON_SIZES, '10px 25px')).toBe('M')
    expect(presetOf(BUTTON_SIZES, '10px 25px 10px 25px')).toBe('M')
    expect(presetOf(BUTTON_SIZES, '14px 32px')).toBe('L')
    expect(presetOf(BUTTON_SIZES, '12px 30px')).toBeUndefined()
    expect(lengthPreset(BUTTON_CORNERS, '999px')).toBe('pill')
    expect(lengthPreset(BUTTON_CORNERS, '6')).toBe('rounded')
    expect(lengthPreset(BUTTON_CORNERS, '3px')).toBeUndefined()
    expect(lengthPreset(LINE_WEIGHTS, '1px')).toBe('thin')
    expect(lengthPreset(LINE_LENGTHS, '50%')).toBe('short')
    expect(lengthPreset(LINE_LENGTHS, '320px')).toBeUndefined()
  })

  it('names spacing only when it is even', () => {
    expect(verticalPreset(BLOCK_SPACING, 10, 10)).toBe('M')
    expect(verticalPreset(BLOCK_SPACING, 10, 20)).toBeUndefined()
    expect(evenPreset(COLUMN_SPACING, [0, 0, 0, 0])).toBe('none')
    expect(evenPreset(COLUMN_SPACING, [16, 16, 16, 16])).toBe('M')
    expect(evenPreset(COLUMN_SPACING, [16, 8, 16, 8])).toBeUndefined()
  })
})
