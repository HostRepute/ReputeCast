import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { meQueryKey, type Me, type WorkspaceMe } from '@/lib/session'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { AppShell } from './AppShell'

const me: Me = {
  user: {
    id: 'u-1',
    email: 'linh@acme.io',
    name: 'Linh Nguyen',
    avatarColor: 'info',
    timezone: 'UTC',
    isInstanceAdmin: false,
    breakGlass: false,
    hasPassword: true,
    totp: true,
    passkeys: 0,
    recoveryCodes: 10,
  },
  workspaces: [
    { id: 'ws-1', slug: 'acme-newsletter', name: 'Acme Newsletter', roleKey: 'owner', roleName: 'Owner', memberCount: 6, subscribers: 48213 },
    { id: 'ws-2', slug: 'acme-engineering-blog', name: 'Acme Engineering Blog', roleKey: 'owner', roleName: 'Owner', memberCount: 2, subscribers: 3100 },
  ],
  instance: { name: 'Acme ReputeCast', publicUrl: 'https://reputecast.acme.io', version: 'v1.0.0', setupComplete: true },
}

const owner = (patch: Partial<WorkspaceMe> = {}) =>
  workspaceMe({
    workspace: { id: 'ws-1', slug: 'acme-newsletter', name: 'Acme Newsletter', roleKey: 'owner', roleName: 'Owner', memberCount: 6, subscribers: 48213 },
    hostRepute: { status: 'connected', teamName: 'Acme Inc', lastSyncAt: new Date().toISOString() },
    ...patch,
  })

const nativeMatchMedia = window.matchMedia

/** Renders the shell at phone width (the tab bar and the More sheet). */
async function phoneShell(path = '/w/acme-newsletter', ws = owner()) {
  window.matchMedia = (query: string) =>
    ({
      matches: query === '(max-width: 767px)',
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList
  return renderInWorkspace(
    <AppShell>
      <header className="top">
        <h1 tabIndex={-1}>Screen</h1>
      </header>
      <main id="main" />
    </AppShell>,
    { path, me: ws, seed: (qc) => qc.setQueryData(meQueryKey, me) },
  )
}

const more = () => screen.getByRole('button', { name: /^More:/, hidden: true })
const sidebarLinks = () =>
  within(screen.getByRole('navigation', { name: 'Main', hidden: true }))
    .getAllByRole('link', { hidden: true })
    .map((a) => a.textContent)

afterEach(() => {
  window.matchMedia = nativeMatchMedia
})

describe('phone navigation and the More sheet', () => {
  it('keeps four sections in the tab bar and opens the others in a sheet', async () => {
    await phoneShell()
    expect(sidebarLinks()).toEqual(['Overview', 'Campaigns', 'Automations', 'Transactional', 'Audience', 'Grow', 'Earn', 'Forms & pages', 'Deliverability', 'Sending providers', 'Settings'])
    // The sections that live in the sheet leave the tab bar (`xm`, hidden at ≤767px).
    const inSheet = screen.getAllByRole('link', { hidden: true }).filter((a) => a.classList.contains('xm'))
    expect(inSheet.map((a) => a.textContent)).toEqual(['Automations', 'Transactional', 'Grow', 'Earn', 'Forms & pages', 'Sending providers', 'Settings'])
    expect(more()).toHaveAccessibleName('More: Automations, Transactional, Grow, Earn, Forms & pages, Sending providers and Settings')
    expect(more()).toHaveAttribute('aria-haspopup', 'dialog')
    expect(more()).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(more())
    const sheet = await screen.findByRole('dialog', { name: 'More' })
    expect(sheet).toHaveAttribute('aria-modal', 'true')
    expect(more()).toHaveAttribute('aria-expanded', 'true')
    expect(more()).toHaveClass('on')
    const sections = within(within(sheet).getByRole('navigation', { name: 'More sections' })).getAllByRole('link')
    expect(sections.map((a) => a.textContent)).toEqual(['Automations', 'Transactional', 'Grow', 'Earn', 'Forms & pages', 'Sending providers', 'Settings'])
    expect(sections.map((a) => a.getAttribute('href'))).toEqual([
      '/w/acme-newsletter/automations',
      '/w/acme-newsletter/transactional',
      '/w/acme-newsletter/grow',
      '/w/acme-newsletter/earn',
      '/w/acme-newsletter/forms',
      '/w/acme-newsletter/providers',
      '/w/acme-newsletter/settings',
    ])
    // The switcher and HostRepute's status, which the sidebar shows on larger screens.
    expect(within(sheet).getByRole('button', { name: 'Acme Newsletter Owner · switch workspace' })).toHaveAttribute('aria-haspopup', 'menu')
    expect(within(sheet).getByRole('link', { name: /HostRepute connected/ })).toHaveTextContent('Acme Inc · synced now')
    expect(within(sheet).getByRole('link', { name: 'Import contacts' })).toHaveAttribute('href', '/w/acme-newsletter/import')
    // Focus moves into the sheet.
    expect(sheet).toContainElement(document.activeElement as HTMLElement)
  })

  it('closes with Escape and gives focus back to More', async () => {
    await phoneShell()
    more().focus()
    fireEvent.click(more())
    const sheet = await screen.findByRole('dialog', { name: 'More' })
    fireEvent.keyDown(sheet, { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(more()).toHaveFocus()
    expect(more()).toHaveAttribute('aria-expanded', 'false')
  })

  it('closes with its Close button', async () => {
    await phoneShell()
    fireEvent.click(more())
    const sheet = await screen.findByRole('dialog', { name: 'More' })
    fireEvent.click(within(sheet).getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('opens the section a link names, closes, and moves focus to the new screen', async () => {
    const { router } = await phoneShell()
    fireEvent.click(more())
    const sheet = await screen.findByRole('dialog', { name: 'More' })
    await act(async () => fireEvent.click(within(sheet).getByRole('link', { name: 'Settings' })))
    await waitFor(() => expect(router.state.location.pathname).toBe('/w/acme-newsletter/settings'))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(screen.getByRole('heading', { name: 'Screen' })).toHaveFocus()
  })

  it('shows the current section as current, in the sheet and on More', async () => {
    await phoneShell('/w/acme-newsletter/settings/team')
    expect(more()).toHaveAttribute('aria-current', 'true')
    expect(more()).toHaveClass('on')
    fireEvent.click(more())
    const sheet = await screen.findByRole('dialog', { name: 'More' })
    expect(within(sheet).getByRole('link', { name: 'Settings' })).toHaveAttribute('aria-current', 'page')
    expect(within(sheet).getByRole('link', { name: 'Automations' })).not.toHaveAttribute('aria-current')
  })

  it("doesn't show More as current on a tab bar section", async () => {
    await phoneShell('/w/acme-newsletter/audience')
    expect(more()).not.toHaveAttribute('aria-current')
    expect(more()).not.toHaveClass('on')
  })

  it('leaves hidden sections out of the sidebar, the tab bar and the sheet', async () => {
    await phoneShell('/w/acme-newsletter', owner({ hiddenSections: ['transactional', 'earn'] }))
    expect(sidebarLinks()).toEqual(['Overview', 'Campaigns', 'Automations', 'Audience', 'Grow', 'Forms & pages', 'Deliverability', 'Sending providers', 'Settings'])
    expect(more()).toHaveAccessibleName('More: Automations, Grow, Forms & pages, Sending providers and Settings')
    fireEvent.click(more())
    const sheet = await screen.findByRole('dialog', { name: 'More' })
    const sections = within(within(sheet).getByRole('navigation', { name: 'More sections' })).getAllByRole('link')
    expect(sections.map((a) => a.textContent)).toEqual(['Automations', 'Grow', 'Forms & pages', 'Sending providers', 'Settings'])
  })

  it('stays closed above phone width', async () => {
    await phoneShell()
    window.matchMedia = nativeMatchMedia
    fireEvent.click(more())
    // useMediaQuery reads the query on each render: at desktop width the sheet can't open.
    await act(async () => {})
    expect(screen.queryByRole('dialog')).toBeNull()
  })
})
