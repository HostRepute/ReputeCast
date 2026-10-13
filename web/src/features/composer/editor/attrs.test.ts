import { describe, expect, it } from 'vitest'
import { fontLabel, formatBorder, headDefaults, hexColor, parseBorder, pxNumber, withPx } from './attrs'
import { collapseBox, escapeAttr, expandBox } from './mjmlHooks'

describe('headDefaults', () => {
  it('reads mj-attributes per tag and mj-class definitions', () => {
    const d = headDefaults(`<mj-title>x</mj-title><mj-attributes>
      <mj-all font-family="Georgia, 'Times New Roman', serif"></mj-all>
      <mj-text color="#1F2937" padding="0px" />
      <mj-class name="muted" color="#5B6472" font-size="12px"></mj-class>
    </mj-attributes>`)
    expect(d.tags['mj-all']).toEqual({ 'font-family': "Georgia, 'Times New Roman', serif" })
    expect(d.tags['mj-text']).toEqual({ color: '#1F2937', padding: '0px' })
    expect(d.classes.muted).toEqual({ color: '#5B6472', 'font-size': '12px' })
  })
})

describe('box shorthands', () => {
  it('expands and collapses CSS boxes', () => {
    expect(expandBox('16px 24px')).toEqual({ top: '16px', right: '24px', bottom: '16px', left: '24px' })
    expect(expandBox('0px 0px 6px')).toEqual({ top: '0px', right: '0px', bottom: '6px', left: '0px' })
    expect(collapseBox('16px', '24px', '16px', '24px')).toBe('16px 24px')
    expect(collapseBox('8px', '8px', '8px', '8px')).toBe('8px')
    expect(collapseBox('20px', '24px', '8px', '24px')).toBe('20px 24px 8px')
    expect(collapseBox('1px', '2px', '3px', '4px')).toBe('1px 2px 3px 4px')
  })

  it('escapes attribute values for MJML', () => {
    expect(escapeAttr('Say "hi" & <wave>')).toBe('Say &quot;hi&quot; &amp; &lt;wave&gt;')
  })
})

describe('inspector values', () => {
  it('converts between bare numbers and px', () => {
    expect(pxNumber('16px')).toBe('16')
    expect(pxNumber('1.5em')).toBe('1.5em')
    expect(withPx('16')).toBe('16px')
    expect(withPx('50%')).toBe('50%')
  })

  it('parses and writes borders', () => {
    expect(parseBorder('1px solid #E3E6EB')).toEqual({ style: 'solid', width: '1px', color: '#E3E6EB' })
    expect(parseBorder('none').style).toBe('none')
    expect(formatBorder({ style: 'dashed', width: '2px', color: '' })).toBe('2px dashed #E3E6EB')
    expect(formatBorder({ style: 'none', width: '1px', color: '#000' })).toBe('')
  })

  it('normalises colours and names font stacks', () => {
    expect(hexColor('#ABC')).toBe('#aabbcc')
    expect(hexColor('white')).toBe('#ffffff')
    expect(hexColor('rgb(1,2,3)')).toBe('')
    expect(fontLabel("'IBM Plex Sans', Helvetica, Arial, sans-serif")).toBe('IBM Plex Sans')
  })
})
