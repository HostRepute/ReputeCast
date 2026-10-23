import { render, screen } from '@testing-library/react'
import { act } from 'react'
import { T, defineMessages, setCatalogLoader, setLocale, t, translate } from '.'
import { viNames } from './names'

const m = defineMessages('namestest', {
  add: 'Add {language}',
  label: '{language}',
  readers: '{language} readers get {gets}',
  default: '{language} is the default. {other} is the fallback.',
  status: '{status, select, outdated {{language} is out of date} other {the {language} version}}',
  bold: '<b>{language}</b> is the default',
  code: '{language} · {code}',
  missing: 'Missing · gets {language}',
  list: 'Live in {languages}',
  named: 'Sent to {list}',
  menu: 'Language: {language}',
})

const vi = translate(m, {
  add: 'Thêm {language}',
  label: '{language}',
  readers: 'Người đọc {language} nhận {gets}',
  default: '{language} là ngôn ngữ mặc định. {other} là ngôn ngữ dự phòng.',
  status: '{status, select, outdated {{language} lỗi thời} other {phiên bản {language}}}',
  bold: '<b>{language}</b> là ngôn ngữ mặc định',
  code: '{language} · {code}',
  missing: 'Thiếu · nhận {language}',
  list: 'Đang chạy bằng {languages}',
  named: 'Đã gửi đến {list}',
  menu: 'Ngôn ngữ: {language}',
})

describe('Vietnamese language names', () => {
  beforeAll(() => setCatalogLoader('vi', async () => vi))
  afterEach(async () => {
    await act(() => setLocale('en'))
  })

  it('take the case their place in the sentence needs', async () => {
    await setLocale('vi')
    expect(t(m.add, { language: 'Tiếng Tây Ban Nha' })).toBe('Thêm tiếng Tây Ban Nha')
    expect(t(m.label, { language: 'Tiếng Anh' })).toBe('Tiếng Anh')
    expect(t(m.label, { language: 'tiếng Anh' })).toBe('Tiếng Anh')
    expect(t(m.readers, { language: 'Tiếng Tây Ban Nha', gets: 'Tiếng Anh' })).toBe('Người đọc tiếng Tây Ban Nha nhận tiếng Anh')
    expect(t(m.default, { language: 'tiếng Đức', other: 'Tiếng Anh' })).toBe('Tiếng Đức là ngôn ngữ mặc định. Tiếng Anh là ngôn ngữ dự phòng.')
    expect(t(m.status, { status: 'outdated', language: 'Tiếng Pháp' })).toBe('Tiếng Pháp lỗi thời')
    expect(t(m.status, { status: 'draft', language: 'Tiếng Pháp' })).toBe('phiên bản tiếng Pháp')
    expect(t(m.bold, { language: 'Tiếng Ý' })).toBe('Tiếng Ý là ngôn ngữ mặc định')
    expect(t(m.code, { language: 'Tiếng Anh (Mỹ)', code: 'en-US' })).toBe('Tiếng Anh (Mỹ) · en-US')
    expect(t(m.missing, { language: 'Tiếng Anh' })).toBe('Thiếu · nhận tiếng Anh')
    expect(t(m.menu, { language: 'Tiếng Việt' })).toBe('Ngôn ngữ: Tiếng Việt')
    expect(t(m.list, { languages: 'Tiếng Anh, Tiếng Pháp và Tiếng Ả Rập' })).toBe('Đang chạy bằng tiếng Anh, tiếng Pháp và tiếng Ả Rập')
  })

  it('leave other names and English alone', async () => {
    expect(t(m.add, { language: 'Tiếng Anh' })).toBe('Add Tiếng Anh')
    await setLocale('vi')
    expect(t(m.named, { list: 'Tiếng Anh cho người đi làm' })).toBe('Đã gửi đến Tiếng Anh cho người đi làm')
    expect(t(m.named, { list: 'Weekly Digest và Tiếng Anh' })).toBe('Đã gửi đến Weekly Digest và Tiếng Anh')
    expect(t(m.add, { language: 'Bản tin' })).toBe('Thêm Bản tin')
  })

  it('apply to rich messages too', async () => {
    await act(() => setLocale('vi'))
    render(<T msg={m.readers} values={{ language: 'Tiếng Tây Ban Nha', gets: <b>Tiếng Anh</b> }} />)
    expect(screen.getByText(/Người đọc tiếng Tây Ban Nha nhận/)).toBeInTheDocument()
  })

  it('split lists on the list words only', () => {
    expect(viNames('Tiếng Anh, Tiếng Pháp hoặc Tiếng Đức', false)).toBe('Tiếng Anh, tiếng Pháp hoặc tiếng Đức')
    expect(viNames('Tiếng Trung (Phồn thể)', true)).toBe('tiếng Trung (Phồn thể)')
  })
})
