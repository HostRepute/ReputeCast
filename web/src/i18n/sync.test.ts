import { QueryClient } from '@tanstack/react-query'
import { setCatalogLoader, setLocale, setRegion } from '.'
import { refetchOnLanguageChange } from './sync'

const account = ['get', '/api/v1/me/account']

describe('refetchOnLanguageChange', () => {
  beforeAll(() => setCatalogLoader('vi', async () => ({})))
  afterEach(async () => {
    await setLocale('en')
    setRegion('en-US')
  })

  it("refetches the server's texts when the language or a format changes, but not /me or /meta", async () => {
    const qc = new QueryClient()
    qc.setQueryData(account, { sessionPolicy: { lifetime: '14 days' } })
    qc.setQueryData(['me'], { user: {} })
    qc.setQueryData(['meta'], {})
    const invalidated = (key: unknown[]) => qc.getQueryState(key)?.isInvalidated
    setRegion('en-US')
    const stop = refetchOnLanguageChange(qc)

    setRegion('en-US') // the same format: nothing to refetch
    expect(invalidated(account)).toBe(false)

    await setLocale('vi')
    expect(invalidated(account)).toBe(true)
    expect(invalidated(['me'])).toBe(false)
    expect(invalidated(['meta'])).toBe(false)

    qc.setQueryData(account, { sessionPolicy: { lifetime: '14 ngày' } })
    setRegion('vi-VN')
    expect(invalidated(account)).toBe(true)

    stop()
    qc.setQueryData(account, {})
    await setLocale('en')
    expect(invalidated(account)).toBe(false)
  })
})
