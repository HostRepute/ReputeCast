import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { act } from 'react'
import { getI18n, setCatalogLoader, setLocale } from '@/i18n'
import { renderInWorkspace } from '@/test/app'
import { AuthLanguageMenu } from './AuthLanguageMenu'
import vi from './messages.vi'

describe('AuthLanguageMenu', () => {
  // This feature's translations only: loading every catalog can take longer
  // than waitFor's second on a slow runner.
  beforeAll(() => setCatalogLoader('vi', async () => vi))
  afterEach(async () => {
    window.localStorage.clear()
    await act(() => setLocale('en'))
  })

  it('names each language in its own language and switches the page', async () => {
    await renderInWorkspace(<AuthLanguageMenu languages={[{ code: 'en' }, { code: 'vi' }]} />, { path: '/sign-in' })
    const nav = screen.getByRole('navigation', { name: 'Language' })
    expect(within(nav).getByRole('button', { name: 'English' })).toHaveAttribute('aria-current', 'true')
    const vi = within(nav).getByRole('button', { name: 'Tiếng Việt' })
    expect(vi).toHaveAttribute('lang', 'vi')

    fireEvent.click(vi)
    await waitFor(() => expect(getI18n().locale).toBe('vi'))
    // The choice is remembered in this browser, before an invitation's language.
    expect(window.localStorage.getItem('rc.language')).toBe('"vi"')
  })

  it('stays out of the app and out of single-language instances', async () => {
    const { unmount } = await renderInWorkspace(<AuthLanguageMenu languages={[{ code: 'en' }, { code: 'vi' }]} />)
    expect(screen.queryByRole('navigation', { name: 'Language' })).toBeNull()
    unmount()
    await renderInWorkspace(<AuthLanguageMenu languages={[{ code: 'en' }]} />, { path: '/sign-in' })
    expect(screen.queryByRole('navigation', { name: 'Language' })).toBeNull()
  })
})
