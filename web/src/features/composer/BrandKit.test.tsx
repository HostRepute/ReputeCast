import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Component } from 'grapesjs'
import { afterEach, describe, expect, it } from 'vitest'
import { t } from '@/i18n'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { BrandKitBanner, SaveAsSynced, SyncedNotice, syncedBlockDefs } from './BrandKit'
import { ComposerCtx } from './context'
import { lockSynced, syncedSectionOf, withoutSyncedMark } from './editor/synced'
import { headlessMjmlEditor } from './editor/testEditor'
import { testCampaign, testComposer } from './testing'

// CO-11: the brand kit in the composer.

const campaign = testCampaign()
const me = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] })
const kitUrl = '/api/v1/workspaces/acme-newsletter/brand-kit'

let unmock: (() => void) | undefined
afterEach(() => unmock?.())

const footer = '<mj-section rc-synced="footer" padding="18px 24px"><mj-column><mj-text>Acme Inc · <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>'

describe('brand kit in the composer', () => {
  it('offers to set up the brand kit over the gallery until it is saved', async () => {
    unmock = mockApi({ [`GET ${kitUrl}`]: () => json({ saved: false, synced: [], colors: {}, heading: {}, body: {}, button: {} }) })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign })}>
        <BrandKitBanner />
      </ComposerCtx.Provider>,
      { me },
    )
    expect(await screen.findByText('Start from your brand kit')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Set up brand kit' })).toHaveAttribute('href', '/w/acme-newsletter/settings/brand-kit')
  })

  it('shows nothing once the kit is saved', async () => {
    unmock = mockApi({ [`GET ${kitUrl}`]: () => json({ saved: true, synced: [], colors: {}, heading: {}, body: {}, button: {} }) })
    const { container } = await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign })}>
        <BrandKitBanner />
      </ComposerCtx.Provider>,
      { me },
    )
    await new Promise((r) => setTimeout(r, 30))
    expect(container.textContent).not.toContain('Start from your brand kit')
  })

  it('makes the synced Header and Footer blocks', () => {
    const defs = syncedBlockDefs({ synced: [{ kind: 'footer', mjml: footer }] } as never, t)
    expect(defs).toEqual([expect.objectContaining({ id: 'rc-synced-footer', label: 'Footer + unsubscribe', badge: 'Synced', kind: 'section', content: footer })])
  })

  it('keeps a synced section read-only and says where it changes', async () => {
    const { editor } = await headlessMjmlEditor(`<mjml><mj-body><mj-section><mj-column><mj-text>News</mj-text></mj-column></mj-section>${footer}</mj-body></mjml>`)
    lockSynced(editor)
    const texts = editor.getWrapper()!.findType('mj-text') as Component[]
    const [news, foot] = texts as [Component, Component]
    expect(news.get('editable')).not.toBe(false)
    expect(foot.get('editable')).toBe(false)
    expect(foot.get('removable')).toBe(false)
    expect(syncedSectionOf(foot)?.kind).toBe('footer')
    expect(syncedSectionOf(news)).toBeNull()
    // Locking isn't an edit.
    expect(editor.UndoManager.hasUndo()).toBe(false)
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, editor })}>
        <SyncedNotice component={foot} />
      </ComposerCtx.Provider>,
      { me },
    )
    expect(screen.getByRole('note')).toHaveTextContent('Synced Footer. It changes in the brand kit, and every draft and scheduled email holding it follows.')
    expect(screen.getByRole('link', { name: 'Edit in Brand kit' })).toBeInTheDocument()
    editor.destroy()
  })
})

// R4.A3: synced blocks beyond the Header and Footer.
describe('named synced blocks in the composer', () => {
  const kind = 'b-0123456789ab'
  const webinar = `<mj-section rc-synced="${kind}" padding="16px 24px"><mj-column><mj-text>Join the webinar</mj-text></mj-column></mj-section>`
  const kitWith = (synced: unknown[]) => () => json({ saved: true, synced, colors: {}, heading: {}, body: {}, button: {} })
  const block = { kind, name: 'Webinar banner', mjml: webinar, version: 1, updatedBy: 'Linh Nguyen', updatedAt: '2026-10-25T09:00:00Z', uses: { drafts: 2, scheduled: 0, held: 0, sent: 1 } }

  it('offers each one in Your blocks, by its name', () => {
    const defs = syncedBlockDefs({ synced: [{ kind: 'footer', name: '', mjml: footer }, block] } as never, t)
    expect(defs[1]).toEqual(expect.objectContaining({ id: `rc-synced-${kind}`, label: 'Webinar banner', badge: 'Synced', group: 'reputecast', kind: 'section', content: webinar, icon: 'saved' }))
    expect(defs[1]!.keywords).toContain('webinar banner')
  })

  it('says which block a section is, and unlinks it to edit it in this email only', async () => {
    unmock = mockApi({ [`GET ${kitUrl}`]: kitWith([block]) })
    const { editor } = await headlessMjmlEditor(`<mjml><mj-body>${webinar}</mj-body></mjml>`)
    lockSynced(editor)
    const text = editor.getWrapper()!.findType('mj-text')[0] as Component
    expect(text.get('editable')).toBe(false)
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, editor })}>
        <SyncedNotice component={text} />
      </ComposerCtx.Provider>,
      { me },
    )
    expect(await screen.findByText('Synced block “Webinar banner”.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Edit in Brand kit' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Unlink' }))
    const section = editor.getWrapper()!.findType('mj-section')[0] as Component
    expect(section.getAttributes()['rc-synced']).toBeUndefined()
    const fresh = editor.getWrapper()!.findType('mj-text')[0] as Component
    expect(fresh.get('editable')).not.toBe(false)
    expect(fresh.get('removable')).not.toBe(false)
    expect(editor.getHtml()).toContain('Join the webinar')
    expect(await screen.findByText('Unlinked from the synced block')).toBeInTheDocument()
    // One undo puts the link back: one section, the synced one.
    editor.UndoManager.undo()
    const after = editor.getWrapper()!.findType('mj-section') as Component[]
    expect(after).toHaveLength(1)
    expect(after[0]!.getAttributes()['rc-synced']).toBe(kind)
    editor.destroy()
  })

  it('says when the block was deleted from the brand kit, and still unlinks it', async () => {
    unmock = mockApi({ [`GET ${kitUrl}`]: kitWith([]) })
    const { editor } = await headlessMjmlEditor(`<mjml><mj-body>${webinar}</mj-body></mjml>`)
    const text = editor.getWrapper()!.findType('mj-text')[0] as Component
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, editor })}>
        <SyncedNotice component={text} />
      </ComposerCtx.Provider>,
      { me },
    )
    expect(await screen.findByText('This synced block was deleted from the brand kit.')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Edit in Brand kit' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Unlink' })).toBeInTheDocument()
    editor.destroy()
  })

  it('saves any section as a synced block, and the email holds the block in its place', async () => {
    const posted: { name: string; mjml: string }[] = []
    unmock = mockApi({
      [`GET ${kitUrl}`]: kitWith([]),
      [`POST ${kitUrl}/synced`]: async (r) => {
        const body = (await r.json()) as { name: string; mjml: string }
        posted.push(body)
        return json({ ...block, name: body.name, mjml: body.mjml.replace('<mj-section', `<mj-section rc-synced="${kind}"`) })
      },
    })
    const { editor } = await headlessMjmlEditor(`<mjml><mj-body><mj-section padding="16px 24px"><mj-column><mj-text>Join the webinar</mj-text></mj-column></mj-section>${footer}</mj-body></mjml>`)
    lockSynced(editor)
    const [section, foot] = editor.getWrapper()!.findType('mj-section') as Component[]
    const text = section!.findType('mj-text')[0] as Component
    const first = await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, editor })}>
        <SaveAsSynced component={text} />
      </ComposerCtx.Provider>,
      { me },
    )
    fireEvent.click(screen.getByRole('button', { name: 'Save as synced block…' }))
    const dialog = await screen.findByRole('dialog', { name: 'Save as synced block' })
    const name = within(dialog).getByRole('textbox', { name: 'Name' }) as HTMLInputElement
    expect(name.value).not.toBe('')
    fireEvent.change(name, { target: { value: 'Webinar banner' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Save synced block' }))
    await waitFor(() => expect(posted).toHaveLength(1))
    expect(posted[0]!.name).toBe('Webinar banner')
    expect(posted[0]!.mjml).toContain('Join the webinar')
    expect(posted[0]!.mjml).not.toContain('rc-synced')
    // The section on the canvas is the block now, read-only like the others.
    await waitFor(() => expect((editor.getWrapper()!.findType('mj-section')[0] as Component).getAttributes()['rc-synced']).toBe(kind))
    const held = editor.getWrapper()!.findType('mj-section')[0] as Component
    expect((held.findType('mj-text')[0] as Component).get('editable')).toBe(false)
    expect(editor.getWrapper()!.findType('mj-section')).toHaveLength(2)
    expect(await screen.findByText('“Webinar banner” saved as a synced block')).toBeInTheDocument()
    first.unmount()

    // Nothing to offer on a synced section, or in a transactional message.
    const again = await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, editor })}>
        <SaveAsSynced component={foot!} />
      </ComposerCtx.Provider>,
      { me },
    )
    expect(screen.queryByRole('button', { name: 'Save as synced block…' })).toBeNull()
    again.unmount()
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, editor, kind: 'transactional' } as never)}>
        <SaveAsSynced component={held} />
      </ComposerCtx.Provider>,
      { me },
    )
    expect(screen.queryByRole('button', { name: 'Save as synced block…' })).toBeNull()
    editor.destroy()
  })

  it('takes only the section’s own mark off when it unlinks', () => {
    expect(withoutSyncedMark(`<mj-wrapper rc-synced="${kind}" padding="0px"><mj-section rc-synced="footer"></mj-section></mj-wrapper>`)).toBe(
      '<mj-wrapper padding="0px"><mj-section rc-synced="footer"></mj-section></mj-wrapper>',
    )
  })
})
