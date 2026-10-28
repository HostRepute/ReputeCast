import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import type { Component } from 'grapesjs'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json } from '@/test/api'
import type { Campaign } from './context'
import type * as Grapes from './editor/grapes'
import { headless } from './editor/testEditor'
import { pageTest, pageWait, renderComposerPage, workspaceBase } from './testPage'
import { testCampaign } from './testing'

// The whole composer on a headless editor (testPage.tsx).
vi.mock('./editor/grapes', async (importOriginal) => ({
  ...(await importOriginal<typeof Grapes>()),
  createComposerEditor: (await import('./editor/testEditor')).headlessComposerEditor,
}))

type Body = Record<string, unknown>

/** The content endpoint: records each save and answers the next version. */
function contentRoute(id: string, saves: Body[]) {
  return {
    [`PUT ${workspaceBase}/campaigns/${id}/content`]: async (r: Request) => {
      saves.push((await r.json()) as Body)
      return json({ version: 14 + saves.length, savedAt: '2026-10-25T09:40:00Z', contentHash: `h${saves.length}`, sizeBytes: 1200, plainText: 'Hi', html: '<p>Hi</p>' })
    },
  }
}

/** The composer is ready once the canvas loaded (the headless editor). */
async function editorReady() {
  await waitFor(() => expect(headless.current?.editor.getWrapper()?.findType('mj-text').length).toBeGreaterThan(0), pageWait)
  return headless.current!
}

describe('the composer page', () => {
  let unmock: (() => void) | undefined
  afterEach(() => {
    unmock?.()
    vi.restoreAllMocks()
  })

  // R4.L1, CO-1: forty minutes in one text must not be lost. Every 30
  // seconds the words being typed are saved, while the typing goes on.
  it('autosaves the words of a text being typed every 30 seconds, without ending the typing', pageTest, async () => {
    const campaign = testCampaign()
    const saves: Body[] = []
    // The composer's own timers: catch the 30-second autosave to run its tick.
    const ticks: (() => void)[] = []
    const setInterval = window.setInterval.bind(window)
    vi.spyOn(window, 'setInterval').mockImplementation(((fn: () => void, ms?: number) => {
      if (ms === 30_000) ticks.push(fn)
      return setInterval(fn, ms)
    }) as typeof window.setInterval)
    unmock = (await renderComposerPage(() => campaign, { routes: contentRoute(campaign.id, saves) })).unmock
    const inst = await editorReady()
    expect(ticks).toHaveLength(1)

    // Typing in the text: the words are on the canvas, not yet in the model.
    const text = inst.editor.getWrapper()!.findType('mj-text')[0]! as Component
    const typed = document.createElement('div')
    typed.innerHTML = 'Hi there, three launches and one deprecation'
    vi.spyOn(text, 'getView').mockReturnValue({ getChildrenContainer: () => typed } as never)
    vi.spyOn(inst.editor, 'getEditing').mockReturnValue(text)
    const flush = vi.spyOn(inst, 'flushEditing')
    act(() => void inst.editor.trigger('component:input', text))
    await waitFor(() => expect(document.getElementById('cstatus')).toHaveTextContent('Saved 09:38 · v14'), pageWait)
    const before = saves.length

    act(() => ticks[0]!())
    await waitFor(() => expect(saves.length).toBe(before + 1), pageWait)
    const saved = saves[before]!
    expect(saved.mjml).toContain('Hi there, three launches and one deprecation')
    expect(saved).toMatchObject({ source: 'autosave', baseVersion: 14 })
    // The typing goes on: the editing wasn't ended for the save.
    expect(flush).not.toHaveBeenCalled()
    expect(text.toHTML()).not.toContain('three launches')
    await waitFor(() => expect(document.getElementById('cstatus')).toHaveTextContent('Saved 09:40 · v15'), pageWait)

    // Nothing new since: the next tick saves nothing.
    act(() => ticks[0]!())
    await new Promise((r) => setTimeout(r, 50))
    expect(saves).toHaveLength(before + 1)
  })

  // R4.L6, CO-6: previewing variant B shows B's body, with its unsaved edits, not A's.
  it("previews variant B's body, as edited on the canvas", pageTest, async () => {
    const bBody = '<mjml><mj-body><mj-section><mj-column><mj-text>Variant B body</mj-text></mj-column></mj-section></mj-body></mjml>'
    const campaign: Campaign = testCampaign({
      mjml: '<mjml><mj-body><mj-section><mj-column><mj-text>Variant A body</mj-text></mj-column></mj-section></mj-body></mjml>',
      abTest: { kind: 'content', variants: [{ key: 'A' }, { key: 'B', mjml: bBody, html: '<p>Variant B body</p>' }], testPercent: 20, metric: 'clicks', durationHours: 4 },
    })
    const previews: Body[] = []
    unmock = (
      await renderComposerPage(() => campaign, {
        routes: {
          [`POST ${workspaceBase}/campaigns/${campaign.id}/preview`]: async (r) => {
            const body = (await r.json()) as Body
            previews.push(body)
            return json({ subject: 'Hi', preheader: '', html: String(body.html ?? ''), text: '', fromName: 'Acme', fromEmail: 'news@acme.io', tokens: [], problems: [], renderedAt: '2026-10-25T09:40:00Z', language: '', languageReason: '' })
          },
        },
      })
    ).unmock
    const inst = await editorReady()

    // Variant B on the canvas, and an edit to it that isn't saved yet.
    fireEvent.click(screen.getByRole('button', { name: 'Variant B · content' }))
    await waitFor(() => expect(inst.getBody()).toContain('Variant B body'), pageWait)
    act(() => {
      inst.editor.getWrapper()!.findType('mj-text')[0]!.components('Variant B body, edited')
    })
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }))
    await waitFor(() => expect(previews.length).toBeGreaterThan(0), pageWait)
    const last = previews[previews.length - 1]!
    expect(last.variant).toBe('B')
    expect(last.html).toContain('Variant B body, edited')
    expect(last.html).not.toContain('Variant A body')
  })
})
