import { costRows, DEFAULT_INPUTS, money, num, volumeLabel } from './cost'

describe('num', () => {
  it('reads numbers the way people type them', () => {
    expect(num('200000')).toBe(200000)
    expect(num('200,000')).toBe(200000)
    expect(num('$0.10')).toBe(0.1)
    expect(num(' 24 ')).toBe(24)
  })
  it('treats anything unreadable as zero', () => {
    expect(num('')).toBe(0)
    expect(num('abc')).toBe(0)
  })
})

describe('money', () => {
  it('formats dollars with two decimals and separators', () => {
    expect(money(20)).toBe('$20.00')
    expect(money(1234.5)).toBe('$1,234.50')
  })
})

describe('costRows', () => {
  it('reproduces the board at 200,000 emails', () => {
    const rows = costRows(DEFAULT_INPUTS)
    expect(rows.map((r) => [r.name, r.cost, r.pct])).toEqual([
      ['Amazon SES', '$20.00', '6%'],
      ['Resend', '$80.00', '23%'],
      ['Mailgun', '$160.00', '46%'],
      ['Postmark', '$250.00', '71%'],
      ['Postal, self-hosted', '$24.00', '7%'],
      ['Brevo', '$120.00', '34%'],
      ['Mailchimp', '$350.00', '100%'],
      ['Kit', '$300.00', '86%'],
    ])
    expect(volumeLabel(DEFAULT_INPUTS)).toBe('200,000')
  })

  it('scales per-1,000 prices with the volume and keeps flat plans flat', () => {
    const rows = costRows({ ...DEFAULT_INPUTS, vol: '1,000,000' })
    expect(rows.find((r) => r.name === 'Amazon SES')?.cost).toBe('$100.00')
    expect(rows.find((r) => r.name === 'Brevo')?.cost).toBe('$120.00')
  })

  it('keeps a visible sliver for tiny costs and empty bars when everything is zero', () => {
    const tiny = costRows({ ...DEFAULT_INPUTS, vol: '1000' })
    expect(tiny.find((r) => r.name === 'Amazon SES')?.pct).toBe('2%')
    const zero = costRows({ vol: '0', ses: '0', resend: '0', mailgun: '0', postmark: '0', postal: '0', brevo: '0', mailchimp: '0', kit: '0' })
    expect(zero.every((r) => r.pct === '0%')).toBe(true)
  })
})
