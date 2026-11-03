import { render, screen } from '@testing-library/react'
import { act } from 'react'
import { T, compareText, defineMessages, formatList, setCatalogLoader, setLocale, setRegion, sourceMessages, t, translate, useT } from '.'
import { isPlainText } from './format'

const m = defineMessages('test', {
  title: 'Transactional',
  sent: '{count, plural, one {# message} other {# messages}} sent',
  sentTo: 'Sent to <b>{name}</b>',
  rate: 'Delivered {rate, number, ::percent .0}',
  plain: "Don't send #1 yet",
  quoted: "It''s done",
})

const vi = translate(m, {
  title: 'Giao dịch',
  sent: 'Đã gửi {count, plural, other {# thư}}',
})

function Title() {
  const tr = useT()
  return <h1>{tr(m.title)}</h1>
}

describe('i18n', () => {
  beforeAll(() => setCatalogLoader('vi', async () => vi))
  afterEach(async () => {
    await act(() => setLocale('en'))
    setRegion('en-US')
  })

  it('formats English sources with plurals', () => {
    expect(t(m.sent, { count: 1 })).toBe('1 message sent')
    expect(t(m.sent, { count: 48213 })).toBe('48,213 messages sent')
  })

  it('gives plain text back as it is, and formats the rest', () => {
    expect(isPlainText(m.plain.message)).toBe(true)
    expect(t(m.plain)).toBe("Don't send #1 yet")
    // A doubled apostrophe is ICU's escape: the formatter still reads it.
    expect(isPlainText(m.quoted.message)).toBe(false)
    expect(t(m.quoted)).toBe("It's done")
    expect(isPlainText(m.sentTo.message)).toBe(false)
    render(<T msg={m.plain} />)
    expect(document.body.textContent).toBe("Don't send #1 yet")
  })

  it('records every source message by ID', () => {
    expect(sourceMessages().get('test.title')).toBe('Transactional')
  })

  it('uses a translation and falls back to English per message', async () => {
    await setLocale('vi')
    expect(t(m.title)).toBe('Giao dịch')
    expect(t(m.sent, { count: 2 })).toBe('Đã gửi 2 thư')
    expect(t(m.sentTo, { name: 'Sam' })).toBe('Sent to Sam')
    expect(document.documentElement.lang).toBe('vi')
  })

  it('formats numbers by the region format', () => {
    setRegion('vi-VN')
    expect(t(m.sent, { count: 48213 })).toBe('48.213 messages sent')
    expect(t(m.rate, { rate: 0.991 })).toBe('Delivered 99,1%')
  })

  it('renders markup with tag functions', () => {
    render(<T msg={m.sentTo} values={{ name: 'Sam Rivera', b: (chunks) => <strong>{chunks}</strong> }} />)
    expect(screen.getByText('Sam Rivera').tagName).toBe('STRONG')
  })

  it('re-renders when the language changes', async () => {
    render(<Title />)
    expect(screen.getByRole('heading').textContent).toBe('Transactional')
    await act(() => setLocale('vi'))
    expect(screen.getByRole('heading').textContent).toBe('Giao dịch')
  })

  it('sorts and joins by the language', async () => {
    expect(['Zoe', 'émile', 'Anna'].sort(compareText)).toEqual(['Anna', 'émile', 'Zoe'])
    expect(formatList(['English', 'Spanish', 'German'])).toBe('English, Spanish and German')
    await setLocale('vi')
    expect(formatList(['A', 'B'], 'disjunction')).toBe('A hoặc B')
  })
})
