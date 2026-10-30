import { act, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ComposerCtx } from './context'
import { EmptyGallery } from './Gallery'
import { pictureDocument } from './Thumb'
import { testComposer } from './testing'

// R4.U4: the gallery shows each template's real email, drawn small in a
// sandboxed frame that loads as the tile comes into view.

const ws = '/api/v1/workspaces/acme-newsletter'
const me = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit', 'templates:view'] })
const lastId = '01970000-0000-7000-8000-0000000000c9'
const savedId = '01970000-0000-7000-8000-0000000000d1'
const gallery = {
  starters: [
    { id: 'starter:blank', kind: 'template', name: 'Blank', description: 'Start from nothing', thumb: [], builtin: true, sizeBytes: 1 },
    { id: 'starter:weekly-digest', kind: 'template', name: 'Weekly digest', description: 'A hero, two articles and a footer', thumb: [{ kind: 'header', h: 10, w: 40 }], builtin: true, sizeBytes: 1 },
  ],
  templates: [{ id: savedId, kind: 'template', name: 'Acme digest v3', description: '', thumb: [{ kind: 'text', h: 8 }], builtin: false, sizeBytes: 1, updatedAt: '2026-10-20T09:00:00Z' }],
  sections: [],
  lastCampaign: { id: lastId, name: 'Weekly Digest #141', sentAt: '2026-10-21T09:00:00Z', thumb: [{ kind: 'text', h: 8 }] },
}

/** Observers that see every tile at once, and boxes 240 × 140 px (jsdom has no layout). */
let intersect: (() => void)[] = []
beforeEach(() => {
  intersect = []
  vi.stubGlobal(
    'IntersectionObserver',
    class {
      cb: (e: { isIntersecting: boolean }[]) => void
      constructor(cb: (e: { isIntersecting: boolean }[]) => void) {
        this.cb = cb
      }
      observe() {
        intersect.push(() => this.cb([{ isIntersecting: true }]))
      }
      disconnect() {}
    },
  )
  vi.stubGlobal(
    'ResizeObserver',
    class {
      cb: () => void
      constructor(cb: () => void) {
        this.cb = cb
      }
      observe() {
        this.cb()
      }
      disconnect() {}
    },
  )
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(240)
  vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockReturnValue(140)
})

let unmock: (() => void) | undefined
afterEach(() => {
  unmock?.()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function api(routes: Record<string, (r: Request) => Response | Promise<Response>>) {
  unmock = mockApi(new Proxy(routes, { get: (t, k: string) => t[k] ?? (() => json({ title: 'Not Found', status: 404, detail: 'Not in this test.' }, 404)) }))
}

describe('the template gallery', () => {
  it('draws each template’s real email once its tile is in view, outside the button that picks it', async () => {
    const asked: string[] = []
    const campaignPreview = vi.fn(async (r: Request) => {
      expect(await r.json()).toEqual({ sample: 'no_first_name' })
      return json({ html: '<p>Hi there, last week at Acme</p>', subject: '', preheader: '', text: '', tokens: [], problems: [] })
    })
    api({
      [`GET ${ws}/templates`]: () => json(gallery),
      [`GET ${ws}/templates/starter%3Aweekly-digest/preview`]: () => {
        asked.push('weekly')
        return json({ html: '<html><body><h1>This week at Acme</h1><a href="https://acme.io">Read</a></body></html>' })
      },
      // A preview the server can't make: the miniature stays.
      [`GET ${ws}/templates/${savedId}/preview`]: () => json({ title: 'Unprocessable', status: 422, detail: "This template's MJML doesn't compile" }, 422),
      [`POST ${ws}/campaigns/01970000-0000-7000-8000-0000000000c1/preview`]: campaignPreview,
      [`POST ${ws}/campaigns/${lastId}/preview`]: campaignPreview,
    })
    const loadContent = vi.fn()
    const { container } = await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ loadContent })}>
        <EmptyGallery />
      </ComposerCtx.Provider>,
      { me },
    )
    await screen.findByRole('button', { name: 'Weekly digest' })
    // Nothing loads before the tiles come into view.
    expect(asked).toEqual([])
    act(() => intersect.forEach((f) => f()))

    const frame = await screen.findByTitle('Preview of Weekly digest')
    expect(frame.tagName).toBe('IFRAME')
    expect(frame).toHaveAttribute('sandbox', '')
    expect(frame).toHaveAttribute('tabindex', '-1')
    expect(frame).toHaveAttribute('inert')
    expect(frame.closest('[aria-hidden="true"]')).not.toBeNull()
    expect(frame.closest('button')).toBeNull()
    expect(frame.getAttribute('srcdoc')).toContain('This week at Acme')
    expect(frame.getAttribute('srcdoc')).not.toContain('href=')
    // 600 px laid out, drawn 240 px wide.
    expect(frame.style.width).toBe('600px')
    expect(frame.style.transform).toBe('scale(0.4)')

    // The tile's button names the template and says what it is.
    const pick = screen.getByRole('button', { name: 'Weekly digest' })
    expect(pick).toHaveAccessibleDescription('A hero, two articles and a footer')
    expect(within(pick).queryByTitle(/Preview of/)).toBeNull()

    // The last sent campaign, as a reader without a first name sees it.
    expect(await screen.findByTitle('Preview of Weekly Digest #141')).toBeInTheDocument()
    expect(campaignPreview).toHaveBeenCalledTimes(1)

    // Blank stays the "Empty" miniature; a template without a preview keeps its miniature.
    const blankTile = screen.getByRole('button', { name: 'Blank' }).closest('.tpl')!
    expect(blankTile.querySelector('iframe')).toBeNull()
    expect(blankTile).toHaveTextContent('Empty')
    const savedTile = screen.getByRole('button', { name: 'Acme digest v3' }).closest('.tpl')!
    await waitFor(() => expect(savedTile.querySelector('.thumb.mini .mtx')).not.toBeNull())
    expect(savedTile.querySelector('iframe')).toBeNull()
    expect(container.querySelectorAll('iframe')).toHaveLength(2)
  })

  it('picks a template from anywhere on its tile', async () => {
    api({
      [`GET ${ws}/templates`]: () => json(gallery),
      [`GET ${ws}/templates/starter%3Aweekly-digest`]: () => json({ ...gallery.starters[1], mjml: '<mjml><mj-body></mj-body></mjml>', html: '' }),
    })
    const loadContent = vi.fn()
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ loadContent })}>
        <EmptyGallery />
      </ComposerCtx.Provider>,
      { me },
    )
    fireEvent.click(await screen.findByRole('button', { name: 'Weekly digest' }))
    await waitFor(() => expect(loadContent).toHaveBeenCalledWith('<mjml><mj-body></mj-body></mjml>', { source: 'template', label: 'created from template “Weekly digest”' }))
  })
})

describe('a picture of an email', () => {
  it('keeps how it looks and takes out what could run, link or take focus', () => {
    const doc = pictureDocument(
      '<html><head><style>p{color:#374151}</style></head><body><p onclick="x()" tabindex="0">Hi <a href="https://acme.io">there</a></p>' +
        '<script>alert(1)</script><form><input name="q"><button>Go</button></form><div contenteditable="true">Notes</div><img src="/media/1/a.png" alt="A"></body></html>',
    )
    for (const gone of ['onclick', 'tabindex', 'href=', '<script', '<form', '<input', '<button', 'contenteditable']) expect(doc).not.toContain(gone)
    for (const kept of ['p{color:#374151}', '<a>there</a>', '<img src="/media/1/a.png" alt="A">', 'Notes', 'overflow:hidden']) expect(doc).toContain(kept)
  })
})
