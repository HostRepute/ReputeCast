import { formatRatio } from '../contrast'
import { evaluate } from './ColorPicker'

describe('the color picker’s contrast', () => {
  it('checks a button background against its label', () => {
    const e = evaluate('#5EBFB0', { field: 'background', texts: [{ color: '#FFFFFF', sizePx: 15, bold: true, kind: 'button' }], others: ['#0F766E'] })
    expect([e.state, formatRatio(e.verdict!.ratio), e.verdict!.passesAA, e.against]).toEqual(['ok', '2.1:1', false, '#FFFFFF'])
  })

  it('takes the text that does worst on a background', () => {
    const e = evaluate('#FFFFFF', {
      field: 'background',
      texts: [
        { color: '#111827', sizePx: 15, bold: false, kind: 'text' },
        { color: '#9CA3AF', sizePx: 13, bold: false, kind: 'text' },
      ],
      others: [],
    })
    expect(e.against).toBe('#9CA3AF')
    expect(e.verdict!.passesAA).toBe(false)
  })

  it('lets large text pass at 3:1', () => {
    const e = evaluate('#2B9A8C', { field: 'text', text: { sizePx: 26, bold: true, kind: 'heading' }, background: '#FFFFFF', others: [] })
    expect([formatRatio(e.verdict!.ratio), e.verdict!.large, e.verdict!.passesAA, e.verdict!.aaNormal]).toEqual(['3.4:1', true, true, false])
  })

  it('can’t check text over an image', () => {
    expect(evaluate('#FFFFFF', { field: 'text', text: { sizePx: 28, bold: true, kind: 'heading' }, image: 'scheduled-rollouts.png', others: [] }).state).toBe('image')
    expect(evaluate('not a color', { field: 'text', text: { sizePx: 13, bold: false, kind: 'text' }, background: '#FFFFFF', others: [] }).state).toBe('none')
  })
})
