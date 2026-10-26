import { screen } from '@testing-library/react'
import type { Component } from 'grapesjs'
import { afterEach, describe, expect, it } from 'vitest'
import { t } from '@/i18n'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { BrandKitBanner, SyncedNotice, syncedBlockDefs } from './BrandKit'
import { ComposerCtx } from './context'
import { lockSynced, syncedSectionOf } from './editor/synced'
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
