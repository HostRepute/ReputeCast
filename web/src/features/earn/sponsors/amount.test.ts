import { parseAmount } from './dialogs'

describe('a sponsor booking amount', () => {
  it('reads a decimal point or a decimal comma', () => {
    expect(parseAmount('750', 'usd')).toBe(75000)
    expect(parseAmount('1,500.50', 'usd')).toBe(150050)
    expect(parseAmount('750,00', 'eur')).toBe(75000)
    expect(parseAmount('1.500,50', 'eur')).toBe(150050)
    expect(parseAmount('1500', 'jpy')).toBe(1500)
    expect(parseAmount('', 'usd')).toBeUndefined()
    expect(parseAmount('soon', 'usd')).toBeUndefined()
  })
})
