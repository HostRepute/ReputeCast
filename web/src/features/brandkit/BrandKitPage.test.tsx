import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { BrandKitPage } from './BrandKitPage'
import { BrandVoiceSection } from './BrandVoice'

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

  // R4.A3: sections saved as synced blocks, listed after the Header and
  // Footer, edited (and renamed) once for every email, or deleted.
  it('lists the named synced blocks, edits one with its name and deletes one', async () => {
    const kind = 'b-0123456789ab'
    const named = { kind, name: 'Webinar banner', mjml: `<mj-section rc-synced="${kind}"><mj-column><mj-text>Join</mj-text></mj-column></mj-section>`, version: 2, updatedBy: 'Linh Nguyen', updatedAt: '2026-10-20T09:00:00Z', uses: { drafts: 2, scheduled: 1, held: 1, sent: 4 } }
    const footer = { kind: 'footer', name: '', mjml: '<mj-section rc-synced="footer"></mj-section>', version: 3, updatedBy: 'Linh Nguyen', updatedAt: '2026-10-07T09:00:00Z', uses: { drafts: 3, scheduled: 1, held: 0, sent: 10 } }
    let blocks: unknown[] = [footer, named]
    const puts: unknown[] = []
    let deleted = 0
    unmock = mockApi({
      [`GET ${base}`]: () => json({ ...defaults, saved: true, updatedAt: '2026-10-07T09:00:00Z', synced: blocks }),
      'POST /api/v1/workspaces/acme-newsletter/mjml/compile': () => json({ html: '<p>Join</p>', errors: [], warnings: [], sizeBytes: 12, durationMs: 1 }),
      [`PUT ${base}/synced/${kind}`]: async (r) => {
        puts.push(await r.json())
        return json({ block: { ...named, name: 'Webinar (moved)', version: 3 }, applied: { drafts: 2, scheduled: 1, held: 1, sent: 4 } })
      },
      [`DELETE ${base}/synced/${kind}`]: () => {
        deleted++
        blocks = [footer]
        return json({ applied: { drafts: 2, scheduled: 1, held: 1, sent: 4 } })
      },
    })
    await renderInWorkspace(<BrandKitPage />, { me: editor })
    const card = (await screen.findByRole('heading', { name: 'Synced blocks' })).closest('section') as HTMLElement
    expect(card).toHaveTextContent('Webinar banner')
    expect(card).toHaveTextContent('Used in 8 emails')
    expect(card).toHaveTextContent('Saved from a section in the composer')
    expect(card).toHaveTextContent('Any section can be a synced block: select it in the composer, then Settings › Save as synced block.')
    // The Footer can't be deleted; the named block can.
    expect(within(card).getAllByRole('button', { name: /^Delete/ })).toHaveLength(1)

    fireEvent.click(within(card).getByRole('button', { name: 'Edit Webinar banner' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit the synced block “Webinar banner”' })
    fireEvent.change(within(dialog).getByRole('textbox', { name: 'Name' }), { target: { value: 'Webinar (moved)' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save and update emails' }))
    await waitFor(() => expect(puts).toEqual([{ mjml: named.mjml, name: 'Webinar (moved)' }]))
    expect(await screen.findByText('“Webinar (moved)” saved')).toBeInTheDocument()

    fireEvent.click(within(card).getByRole('button', { name: 'Delete Webinar banner' }))
    const confirm = await screen.findByRole('dialog', { name: 'Delete “Webinar banner”?' })
    expect(confirm).toHaveTextContent('The 3 drafts and scheduled emails holding it keep its content as ordinary sections.')
    expect(confirm).toHaveTextContent('5 emails in an approval, sending or sent keep it as they are.')
    fireEvent.click(within(confirm).getByRole('button', { name: 'Delete synced block' }))
    await waitFor(() => expect(deleted).toBe(1))
    expect(await screen.findByText('“Webinar banner” deleted')).toBeInTheDocument()
    await waitFor(() => expect(card).not.toHaveTextContent('Webinar banner'))
  })

  it('is read-only for people who can’t edit templates, and says who can', async () => {
    unmock = mockApi({ [`GET ${base}`]: () => json(defaults) })
    await renderInWorkspace(<BrandKitPage />, { me: viewer })
    expect(await screen.findByText('People who edit templates can change the brand kit.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Save brand kit' })).toBeNull()
    expect(screen.getByLabelText('Primary button colour')).toBeDisabled()
  })
})

describe('brand voice', () => {
  const content = { tone: 'Direct', address: 'you', sentenceLength: 'Short', preferredWords: 'try', avoidedWords: 'hype', examples: '' }
  const initial = { revision: 1, content, learned: false, createdBy: 'Linh' }
  const ai = '/api/v1/workspaces/acme-newsletter/ai'
  const aiEditor = workspaceMe({ permissions: [...editor.permissions, 'ai:use'] })

  it('keeps local edits across refetches, rejects a changed base and reloads only on discard', async () => {
    let voice = initial
    let offline = false
    const writes: unknown[] = []
    unmock = mockApi({
      [`GET ${base}/voice`]: () => offline ? json({ detail: 'Temporarily offline' }, 503) : json(voice),
      [`GET ${base}/voice/history`]: () => json({ items: [voice] }),
      [`PUT ${base}/voice`]: async (r) => {
        const body = await r.json()
        writes.push(body)
        if (body.baseRevision !== voice.revision) return json({ status: 409, detail: 'Changed since you started editing.' }, 409)
        voice = { ...voice, revision: voice.revision + 1, content: body.content }
        return json(voice)
      },
    })
    const { queryClient } = await renderInWorkspace(<BrandVoiceSection />, { me: editor })
    fireEvent.change(await screen.findByLabelText('Tone'), { target: { value: 'My local edit' } })
    offline = true
    await queryClient.invalidateQueries()
    expect(screen.getByLabelText('Tone')).toHaveValue('My local edit')
    offline = false
    voice = { ...voice, revision: 2, content: { ...content, tone: 'Another member’s edit' } }
    await queryClient.invalidateQueries()
    expect(screen.getByLabelText('Tone')).toHaveValue('My local edit')
    fireEvent.click(screen.getByRole('button', { name: 'Save voice' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Changed since you started editing.')
    expect(writes).toEqual([{ baseRevision: 1, content: { ...content, tone: 'My local edit' } }])
    expect(screen.getByLabelText('Tone')).toHaveValue('My local edit')
    fireEvent.click(screen.getByRole('button', { name: 'Discard changes' }))
    expect(screen.getByLabelText('Tone')).toHaveValue('Another member’s edit')
    fireEvent.change(screen.getByLabelText('Tone'), { target: { value: 'Calm and precise' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save voice' }))
    expect(await screen.findByText('Voice revision 3 saved')).toBeInTheDocument()
    expect(voice.content.tone).toBe('Calm and precise')
  })

  it('sends selected sources only, reviews before saving and submits edited suggestions with provenance', async () => {
    let voice = initial
    const writes: unknown[] = []
    const learns: unknown[] = []
    unmock = mockApi({
      [`GET ${base}/voice`]: () => json(voice),
      [`GET ${base}/voice/history`]: () => json({ items: [voice] }),
      [`GET ${ai}/status`]: () => json({ state: 'ready', providerName: 'Fixture', modelName: 'Local model', features: [{ key: 'brand_voice', on: true, available: true }], prices: { input: 1, output: 2 }, cap: { kind: 'usd', usd: 10 }, usage: { costMicros: 0 } }),
      'GET /api/v1/workspaces/acme-newsletter/campaigns': () => json({ items: [{ id: 'selected', name: 'Selected campaign' }, { id: 'excluded', name: 'Other campaign' }], total: 2 }),
      [`POST ${ai}/brand-voice`]: async (r) => {
        learns.push(await r.json())
        return json({ requestId: 'suggestion-1', baseRevision: 1, content: { ...content, tone: 'Warm' }, sent: { providerName: 'Fixture', modelName: 'Local model', characters: 123, costMicros: 1000 }, expiresAt: '2026-11-16T09:00:00Z' })
      },
      [`PUT ${base}/voice`]: async (r) => {
        const body = await r.json()
        writes.push(body)
        voice = { ...voice, revision: 2, content: body.content, learned: true }
        return json(voice)
      },
    })
    await renderInWorkspace(<BrandVoiceSection />, { me: aiEditor })
    fireEvent.click(await screen.findByRole('checkbox', { name: 'Selected campaign' }))
    fireEvent.click(screen.getByRole('button', { name: 'Learn a suggested voice' }))
    const sharing = within(await screen.findByRole('dialog', { name: 'Review before sharing' }))
    expect(learns).toEqual([])
    fireEvent.click(sharing.getByRole('button', { name: 'Share reviewed content' }))
    expect(await screen.findByRole('button', { name: 'Save suggested revision' })).toBeInTheDocument()
    expect(learns).toEqual([{ campaignIds: ['selected'], sharingReviewed: true }])
    expect(writes).toEqual([])
    expect(screen.getByLabelText('Tone')).toHaveValue('Warm')
    fireEvent.change(screen.getByLabelText('Tone'), { target: { value: 'Warm and precise' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save suggested revision' }))
    expect(await screen.findByText('Voice revision 2 saved')).toBeInTheDocument()
    expect(writes).toEqual([{ baseRevision: 1, suggestionId: 'suggestion-1', content: { ...content, tone: 'Warm and precise' } }])
  })

  it('shows the saved voice without offering edits or model calls to a viewer', async () => {
    unmock = mockApi({
      [`GET ${base}/voice`]: () => json(initial),
      [`GET ${base}/voice/history`]: () => json({ items: [initial] }),
    })
    await renderInWorkspace(<BrandVoiceSection />, { me: viewer })
    expect(await screen.findByLabelText('Tone')).toHaveAttribute('readonly')
    expect(screen.queryByRole('button', { name: 'Save voice' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Learn a suggested voice' })).toBeNull()
  })
})
