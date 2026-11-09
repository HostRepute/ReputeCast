import { act, fireEvent, screen } from '@testing-library/react'
import { QueryClient } from '@tanstack/react-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { resetThemeForTests } from '@/lib/theme'
import { meQueryKey } from '@/lib/session'
import { AppearanceCard } from './AppearanceCard'
import { connectTheme } from './useTheme'

const html = () => document.documentElement

// Account & security › Appearance (M54): the theme changes the moment one
// is picked, is saved to the account, and goes back when the server refuses.
describe('Appearance', () => {
  let unmock = () => {}
  beforeEach(() => {
    resetThemeForTests()
    // The device is light.
    vi.spyOn(window, 'matchMedia').mockImplementation(
      (q: string) => ({ matches: false, media: q, addEventListener: () => {}, removeEventListener: () => {} }) as unknown as MediaQueryList,
    )
  })
  afterEach(() => {
    unmock()
    vi.restoreAllMocks()
  })

  it('offers Same as device, Light and Dark, following the device until one is picked', async () => {
    unmock = mockApi({})
    await renderInWorkspace(<AppearanceCard />)
    expect(screen.getByRole('heading', { name: 'Appearance' })).toBeInTheDocument()
    const group = screen.getByRole('group', { name: 'Theme' })
    expect(group).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'Same as device' })).toBeChecked()
    expect(screen.getByRole('radio', { name: 'Light' })).not.toBeChecked()
    expect(screen.getByRole('radio', { name: 'Dark' })).not.toBeChecked()
    // Each choice says what it does; Same as device says what it is now.
    expect(screen.getByRole('radio', { name: 'Same as device' })).toHaveAccessibleDescription('Switches with your device’s light or dark setting. It’s light now.')
  })

  it('switches at once and saves to the account', async () => {
    const bodies: unknown[] = []
    let answer: (r: Response) => void = () => {}
    unmock = mockApi({
      'PUT /api/v1/me/appearance': async (req) => {
        bodies.push(await req.json())
        return new Promise<Response>((resolve) => (answer = resolve))
      },
    })
    await renderInWorkspace(<AppearanceCard />)
    fireEvent.click(screen.getByRole('radio', { name: 'Dark' }))
    // Before the server answers.
    expect(html().dataset.theme).toBe('dark')
    expect(screen.getByRole('radio', { name: 'Dark' })).toBeChecked()
    await vi.waitFor(() => expect(bodies).toEqual([{ theme: 'dark' }]))
    await act(async () => answer(json({ theme: 'dark' })))
    expect(localStorage.getItem('rc.theme')).toBe('dark')
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('goes back with a message when the server refuses', async () => {
    unmock = mockApi({ 'PUT /api/v1/me/appearance': () => json({ status: 500, title: 'Internal Server Error', detail: 'The database is down.' }, 500) })
    await renderInWorkspace(<AppearanceCard />)
    fireEvent.click(screen.getByRole('radio', { name: 'Dark' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Your theme wasn’t saved')
    expect(html().dataset.theme).toBe('light')
    expect(screen.getByRole('radio', { name: 'Same as device' })).toBeChecked()
    expect(localStorage.getItem('rc.theme')).toBeNull()
  })

  it('takes the account’s theme from /me, so a choice made elsewhere applies here', async () => {
    const qc = new QueryClient()
    connectTheme(qc)
    // The query cache tells its listeners on the next tick.
    qc.setQueryData(meQueryKey, { user: { theme: 'dark' } })
    await vi.waitFor(() => expect(html().dataset.theme).toBe('dark'))
    qc.setQueryData(meQueryKey, { user: { theme: 'system' } })
    await vi.waitFor(() => expect(html().dataset.theme).toBe('light'))
  })
})
