import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { BrandKitPage } from './BrandKitPage'

// CO-11: Settings › Brand kit (BrandKit board).

const base = '/api/v1/workspaces/acme-newsletter/brand-kit'
const editor = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit', 'templates:view', 'templates:edit'] })
const viewer = workspaceMe({ permissions: ['campaigns:view', 'templates:view'] })

const defaults = {
  saved: false,
  logoAlt: '',
  colors: { button: '#0F766E', text: '#1F2937', background: '#F3F4F6', link: '#0F766E' },
  heading: { name: '', fallback: "Georgia, 'Times New Roman', serif" },
  body: { name: '', fallback: 'Arial, Helvetica, sans-serif' },
  button: { shape: 'rounded', size: 14 },
  website: '',
  synced: [],
}

const imported = {
  website: 'acme.io',
  foundAt: '2026-10-25T09:12:00Z',
  logoLight: { mediaId: 'm1', url: '/media/m1/acme-logo.png', fileName: 'acme-logo.png', width: 240, height: 64 },
  logoDark: { mediaId: 'm2', url: '/media/m2/acme-logo-white.png', fileName: 'acme-logo-white.png', width: 240, height: 64 },
  logoAlt: 'Acme',
  style: {
    colors: { button: '#4338CA', text: '#1F2937', background: '#F5F3FF', link: '#4338CA' },
    heading: { name: 'Source Serif 4', url: 'https://fonts.googleapis.com/css2?family=Source+Serif+4:wght@400;700&display=swap', fallback: "Georgia, 'Times New Roman', serif" },
    body: { name: 'Inter', url: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;700&display=swap', fallback: 'Arial, Helvetica, sans-serif' },
    button: { shape: 'rounded', size: 14 },
  },
  colors: [
    { hex: '#4338CA', used: true },
    { hex: '#1F2937', used: true },
    { hex: '#F5F3FF', used: true },
    { hex: '#6366F1', used: false },
  ],
  fonts: ['Source Serif 4', 'Inter'],
  notes: ["#6366F1 isn't used for links. It's 4.47:1 on white, under the 4.5:1 that small text needs. Links use the next colour on your site."],
}

let unmock: (() => void) | undefined
afterEach(() => unmock?.())

describe('brand kit', () => {
  it('imports from a website, shows what it found, and saves it', async () => {
    const puts: unknown[] = []
    let saved = false
    unmock = mockApi({
      [`GET ${base}`]: () => json(saved ? { ...defaults, ...imported.style, saved: true, logoLight: imported.logoLight, logoDark: imported.logoDark, logoAlt: 'Acme', website: 'acme.io', updatedAt: '2026-10-25T09:13:00Z' } : defaults),
      [`POST ${base}/import`]: async (r) => {
        expect(await r.json()).toEqual({ website: 'acme.io' })
        return json(imported)
      },
      [`PUT ${base}`]: async (r) => {
        puts.push(await r.json())
        saved = true
        return json({ ...defaults, saved: true })
      },
    })
    await renderInWorkspace(<BrandKitPage />, { me: editor })
    expect(await screen.findByText('Not set up yet')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('Website'), { target: { value: 'acme.io' } })
    fireEvent.click(screen.getByRole('button', { name: 'Import' }))
    expect(await screen.findByText('Imported, not saved yet')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Found on acme.io at 09:12: a logo in light and dark versions, 4 colours and 2 fonts.')
    expect(screen.getByText(/#6366F1 isn't used for links/)).toBeInTheDocument()
    expect(screen.getByText('not used')).toBeInTheDocument()
    expect(screen.getByLabelText('Primary button colour')).toHaveValue('#4338CA')
    expect(screen.getByText('White text 7.9:1 · AAA')).toBeInTheDocument()
    expect(screen.getByLabelText('Alt text')).toHaveValue('Acme')
    fireEvent.click(screen.getByRole('button', { name: 'Pill' }))
    expect(screen.getByRole('button', { name: 'Pill' })).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(screen.getByRole('button', { name: 'Save brand kit' }))
    await waitFor(() => expect(puts).toHaveLength(1))
    expect(puts[0]).toMatchObject({
      logoLightId: 'm1',
      logoDarkId: 'm2',
      logoAlt: 'Acme',
      colors: imported.style.colors,
      heading: { name: 'Source Serif 4', fallback: "Georgia, 'Times New Roman', serif" },
      body: { name: 'Inter' },
      button: { shape: 'pill', size: 14 },
      website: 'acme.io',
    })
  })

  it('shows the contrast where each colour is used, and flags a low one', async () => {
    unmock = mockApi({ [`GET ${base}`]: () => json(defaults) })
    await renderInWorkspace(<BrandKitPage />, { me: editor })
    const input = await screen.findByLabelText('Primary button colour')
    fireEvent.change(input, { target: { value: '#6366F1' } })
    expect(screen.getByText('White text 4.4:1 · below AA')).toBeInTheDocument()
    expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
    fireEvent.change(input, { target: { value: '#63' } })
    expect(screen.getAllByText('Use a colour like #4338CA.').length).toBeGreaterThan(0)
    expect(screen.getByRole('button', { name: 'Save brand kit' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Discard changes' }))
    expect(screen.getByLabelText('Primary button colour')).toHaveValue('#0F766E')
  })

  it('lists the synced blocks with what saving them changes', async () => {
    unmock = mockApi({
      [`GET ${base}`]: () =>
        json({
          ...defaults,
          saved: true,
          updatedAt: '2026-10-07T09:00:00Z',
          synced: [
            { kind: 'footer', mjml: '<mj-section rc-synced="footer"></mj-section>', version: 3, updatedBy: 'Linh Nguyen', updatedAt: '2026-10-07T09:00:00Z', uses: { drafts: 3, scheduled: 1, held: 0, sent: 10 } },
            { kind: 'header', mjml: '<mj-section rc-synced="header"></mj-section>', version: 1, updatedBy: '', updatedAt: '2026-10-02T09:00:00Z', uses: { drafts: 2, scheduled: 0, held: 1, sent: 7 } },
          ],
        }),
      'POST /api/v1/workspaces/acme-newsletter/mjml/compile': () => json({ html: '<p>Footer</p>', errors: [], warnings: [], sizeBytes: 12, durationMs: 1 }),
    })
    await renderInWorkspace(<BrandKitPage />, { me: editor })
    const card = (await screen.findByRole('heading', { name: 'Synced blocks' })).closest('section') as HTMLElement
    expect(card).toHaveTextContent('Used in 14 emails')
    expect(card).toHaveTextContent('On save: 3 drafts and 1 scheduled email update; 10 sent stay as sent.')
    expect(card).toHaveTextContent('On save: 2 drafts and no scheduled emails update; 7 sent stay as sent.')
    expect(card).toHaveTextContent('1 email in an approval or sending keeps its version.')
    fireEvent.click(within(card).getAllByRole('button', { name: 'Edit' })[0]!)
    expect(await screen.findByRole('dialog', { name: 'Edit the synced Footer' })).toBeInTheDocument()
  })

  it('is read-only for people who can’t edit templates, and says who can', async () => {
    unmock = mockApi({ [`GET ${base}`]: () => json(defaults) })
    await renderInWorkspace(<BrandKitPage />, { me: viewer })
    expect(await screen.findByText('People who edit templates can change the brand kit.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save brand kit' })).toBeNull()
    expect(screen.getByLabelText('Primary button colour')).toBeDisabled()
  })
})
