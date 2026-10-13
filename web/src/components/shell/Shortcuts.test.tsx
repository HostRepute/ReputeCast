import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { dismissToast } from '@/components/ui/Toast'
import { useShortcutHandler } from '@/lib/shortcuts'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { GlobalShortcuts } from './Shortcuts'

/** Presses a key; returns false when the page took it (preventDefault). */
const press = (key: string, init: KeyboardEventInit = {}, target: Element = document.body) => fireEvent.keyDown(target, { key, ...init })

afterEach(() => act(() => dismissToast()))

describe('GlobalShortcuts', () => {
  it('G then a letter goes to that section', async () => {
    const { router } = await renderInWorkspace(<GlobalShortcuts />)
    await act(async () => {
      press('g')
      press('o')
    })
    expect(router.state.location.pathname).toBe('/w/acme-newsletter')
    await act(async () => {
      press('g')
      press('a')
    })
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/audience')
    // A second key that isn't a section ends the sequence.
    await act(async () => {
      press('g')
      press('x')
      press('o')
    })
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/audience')
  })

  it('never takes single keys while someone is typing', async () => {
    const { router } = await renderInWorkspace(
      <>
        <textarea aria-label="Preheader" />
        <GlobalShortcuts />
      </>,
    )
    const field = screen.getByLabelText('Preheader')
    field.focus()
    await act(async () => {
      for (const k of ['g', 'o', 'n']) expect(press(k, {}, field)).toBe(true)
      expect(press('?', { shiftKey: true }, field)).toBe(true)
    })
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/campaigns')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('? opens the sheet, where keys do nothing; Done closes it', async () => {
    const { router } = await renderInWorkspace(<GlobalShortcuts />)
    act(() => void press('?', { shiftKey: true }))
    const sheet = await screen.findByRole('dialog', { name: 'Keyboard shortcuts' })
    expect(sheet).toHaveTextContent('Search everything')
    expect(sheet).toHaveTextContent('Sending providers')
    expect(sheet).toHaveTextContent('Every shortcut has a visible button equivalent; none is required.')
    await act(async () => {
      press('g', {}, sheet)
      press('o', {}, sheet)
    })
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/campaigns')
    fireEvent.click(within(sheet).getByRole('button', { name: 'Done' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('N starts a campaign for editors', async () => {
    const { router } = await renderInWorkspace(<GlobalShortcuts />, { me: workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] }) })
    await act(async () => void press('n'))
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/campaigns/new')
  })

  it('N tells a viewer why nothing happened', async () => {
    const { router } = await renderInWorkspace(<GlobalShortcuts />)
    act(() => void press('n'))
    expect(await screen.findByText('Editing campaigns needs the campaigns:edit permission.')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/w/acme-newsletter/campaigns')
  })

  it('⌘K focuses the page search; ⌘S goes to the screen that saves', async () => {
    const save = vi.fn()
    function Screen() {
      useShortcutHandler('save', save)
      return (
        <main>
          <input type="search" aria-label="Search campaigns" />
        </main>
      )
    }
    await renderInWorkspace(
      <>
        <Screen />
        <GlobalShortcuts />
      </>,
    )
    expect(press('k', { metaKey: true })).toBe(false)
    const search = screen.getByLabelText('Search campaigns')
    expect(search).toHaveFocus()
    // Works from inside a field, too.
    expect(press('s', { ctrlKey: true }, search)).toBe(false)
    expect(save).toHaveBeenCalledOnce()
  })

  it("leaves ⌘S and ⌘K to the browser when the page doesn't handle them", async () => {
    await renderInWorkspace(<GlobalShortcuts />)
    expect(press('s', { metaKey: true })).toBe(true)
    expect(press('k', { metaKey: true })).toBe(true)
  })
})
