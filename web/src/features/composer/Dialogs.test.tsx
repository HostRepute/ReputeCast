import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ComposerCtx, type ComposerState } from './context'
import { Dialogs } from './Dialogs'
import { headlessMjmlEditor } from './editor/testEditor'
import { testCampaign, testComposer } from './testing'

const campaign = testCampaign()
const base = `/api/v1/workspaces/acme-newsletter/campaigns/${campaign.id}`
const me = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] })

function version(n: number, patch: Record<string, unknown> = {}) {
  return { version: n, savedAt: '2026-10-25T09:30:00Z', savedBy: 'Linh Nguyen', label: `edit ${n}`, source: 'autosave', sizeBytes: 64000, current: false, name: '', ...patch }
}

function page(items: ReturnType<typeof version>[], nextBefore?: number) {
  return { items, total: 14, named: 1, ...(nextBefore ? { nextBefore } : {}) }
}

let unmock: (() => void) | undefined
afterEach(() => unmock?.())

function open(state: ComposerState) {
  return renderInWorkspace(
    <ComposerCtx.Provider value={state}>
      <Dialogs dialog="versions" onClose={() => {}} onSavedTemplate={() => {}} />
    </ComposerCtx.Provider>,
    { me },
  )
}

describe('version history', () => {
  it('saves unsaved edits first, then restores on top of the version that save made', async () => {
    let current = 14
    const restoreBodies: unknown[] = []
    unmock = mockApi({
      [`GET ${base}/versions`]: () => json(page([version(14, { current: true }), version(13)])),
      [`POST ${base}/versions/13/restore`]: async (r) => {
        restoreBodies.push(await r.json())
        return json({ version: 16, savedAt: '2026-10-25T09:44:00Z', contentHash: 'x', sizeBytes: 1, plainText: '', html: '' })
      },
    })
    const save = vi.fn(async () => {
      current = 15
      return true
    })
    const reload = vi.fn(async () => {})
    await open(testComposer({ campaign, save, hasUnsaved: () => true, currentVersion: () => current, reloadFromServer: reload }))
    const row = (await screen.findByText(/edit 13/)).closest('.ver') as HTMLElement
    fireEvent.click(within(row).getByRole('button', { name: 'Restore version 13' }))
    await waitFor(() => expect(reload).toHaveBeenCalled())
    expect(save).toHaveBeenCalledWith(expect.objectContaining({ manual: true, label: 'edits before restoring v13' }))
    // Based on v15, the version the save made: no conflict.
    expect(restoreBodies).toEqual([{ baseVersion: 15 }])
  })

  it("doesn't restore when the unsaved edits can't be saved", async () => {
    const restore = vi.fn(() => json({}))
    unmock = mockApi({
      [`GET ${base}/versions`]: () => json(page([version(14, { current: true }), version(13)])),
      [`POST ${base}/versions/13/restore`]: restore,
    })
    await open(testComposer({ campaign, save: vi.fn(async () => false), hasUnsaved: () => true }))
    const row = (await screen.findByText(/edit 13/)).closest('.ver') as HTMLElement
    fireEvent.click(within(row).getByRole('button', { name: 'Restore version 13' }))
    expect(await screen.findByText(/couldn't be saved, so v13 wasn't restored/)).toBeInTheDocument()
    expect(restore).not.toHaveBeenCalled()
  })

  it('pages older versions, lists named ones and names a version', async () => {
    const named: unknown[] = []
    unmock = mockApi({
      [`GET ${base}/versions`]: (r) => {
        const q = new URL(r.url).searchParams
        if (q.get('named') === 'true') return json(page([version(9, { name: 'Sent to legal' })]))
        if (q.get('before') === '13') return json(page([version(12), version(11)]))
        return json(page([version(14, { current: true }), version(13)], 13))
      },
      [`PUT ${base}/versions/13/name`]: async (r) => {
        named.push(await r.json())
        return json(version(13, { name: 'Final copy' }))
      },
    })
    await open(testComposer({ campaign }))
    fireEvent.click(await screen.findByRole('button', { name: 'Show older versions' }))
    expect(await screen.findByText(/edit 11/)).toBeInTheDocument()
    // The last page: nothing older to show.
    expect(screen.queryByRole('button', { name: 'Show older versions' })).not.toBeInTheDocument()

    const row = screen.getByText(/edit 13/).closest('.ver') as HTMLElement
    fireEvent.click(within(row).getByRole('button', { name: 'Name version 13' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'Name of version 13' }), { target: { value: 'Final copy' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(named).toEqual([{ name: 'Final copy' }]))

    fireEvent.click(screen.getByRole('button', { name: 'Named · 1' }))
    expect(await screen.findByText('Sent to legal')).toBeInTheDocument()
  })
})

describe('image library (CO-12)', () => {
  const hero = {
    id: 'a1',
    url: '/media/a1/hero.jpg',
    fileName: 'hero.jpg',
    contentType: 'image/jpeg',
    sizeBytes: 217088,
    width: 1200,
    height: 900,
    hasAlpha: false,
    createdAt: '2026-10-25T09:30:00Z',
    createdBy: 'Linh Nguyen',
    original: { sizeBytes: 3250586, width: 4032, height: 3024, contentType: 'image/jpeg' },
  }

  async function openLibrary() {
    const { editor } = await headlessMjmlEditor('<mjml><mj-body><mj-section><mj-column><mj-image src="https://placehold.co/600x300"></mj-image></mj-column></mj-section></mj-body></mjml>')
    const img = editor.getWrapper()!.findType('mj-image')[0]!
    const onClose = vi.fn()
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, editor, assetTarget: { component: img, attr: 'src' } })}>
        <Dialogs dialog="assets" onClose={onClose} onSavedTemplate={() => {}} />
      </ComposerCtx.Provider>,
      { me },
    )
    return { editor, img, onClose }
  }

  it('says what the server did to an upload', async () => {
    unmock = mockApi({ 'GET /api/v1/workspaces/acme-newsletter/media': () => json({ items: [hero, { ...hero, id: 'a2', fileName: 'logo.png', original: undefined }] }) })
    await openLibrary()
    expect(await screen.findByText('Resized from 4032×3024 (3.1 MB) for email')).toBeInTheDocument()
    expect(screen.getAllByText(/for email$/)).toHaveLength(1)
  })

  it('asks for alt text once an image is chosen, and the image gets it', async () => {
    unmock = mockApi({ 'GET /api/v1/workspaces/acme-newsletter/media': () => json({ items: [hero] }) })
    const { editor, img, onClose } = await openLibrary()
    fireEvent.click(await screen.findByRole('button', { name: 'Use hero.jpg' }))
    const dialog = screen.getByRole('dialog', { name: 'Describe the image' })
    expect(dialog).toHaveTextContent('1200×900 · 212 KB')
    expect(within(dialog).getByText('No alt text')).toBeInTheDocument()
    const input = within(dialog).getByLabelText('Alt text for hero.jpg')
    expect(input).toHaveFocus()
    expect(within(dialog).getByRole('button', { name: 'Save' })).toBeDisabled()
    fireEvent.change(input, { target: { value: '  Runners crossing the finish line ' } })
    expect(within(dialog).getByText('Alt text added')).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    expect(img.getAttributes()).toMatchObject({ src: '/media/a1/hero.jpg', alt: 'Runners crossing the finish line' })
    expect(editor.getHtml()).toContain('alt="Runners crossing the finish line"')
    expect(onClose).toHaveBeenCalled()
  })

  it('marks the image decorative, or leaves it for later', async () => {
    unmock = mockApi({ 'GET /api/v1/workspaces/acme-newsletter/media': () => json({ items: [hero] }) })
    const first = await openLibrary()
    fireEvent.click(await screen.findByRole('button', { name: 'Use hero.jpg' }))
    fireEvent.click(screen.getByRole('button', { name: "It's decorative" }))
    expect(first.img.getAttributes().alt).toBe('')
    expect(first.onClose).toHaveBeenCalled()
    cleanup()

    const second = await openLibrary()
    fireEvent.click(await screen.findByRole('button', { name: 'Use hero.jpg' }))
    fireEvent.click(screen.getByRole('button', { name: 'Skip' }))
    expect(second.img.getAttributes().alt).toBeUndefined()
    expect(second.img.getAttributes().src).toBe('/media/a1/hero.jpg')
    expect(second.onClose).toHaveBeenCalled()
  })
})
