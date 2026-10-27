import type { QueryClient } from '@tanstack/react-query'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import type { Component } from 'grapesjs'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { meQueryKey } from '@/lib/session'
import { CommentsTab, commentTime, COMMENT_ATTR } from './Comments'
import { ComposerCtx } from './context'
import { applySections } from './editor/merge'
import { headlessMjmlEditor } from './editor/testEditor'
import { testCampaign, testComposer } from './testing'

// CO-10: comments with @mentions in the composer.

const base = '/api/v1/workspaces/acme-newsletter/campaigns/01970000-0000-7000-8000-0000000000c1'
const editor = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] })
const seedMe = (qc: QueryClient) =>
  qc.setQueryData(meQueryKey, { user: { id: 'u-linh', email: 'linh@acme.io', name: 'Linh Nguyen' }, workspaces: [], instance: {} } as never)

const members = {
  items: [
    { id: 'u-linh', name: 'Linh Nguyen', role: 'Owner', color: 'info' },
    { id: 'u-marcus', name: 'Marcus Chen', role: 'Admin', color: 'purple' },
    { id: 'u-sofia', name: 'Sofia Rossi', role: 'Editor', color: 'info' },
  ],
}

const marcusThread = {
  id: 't1',
  anchor: 'c7k2',
  anchorLabel: 'the button “Read the docs”',
  comments: [{ id: 'c1', body: 'Can we link the changelog instead?', author: 'Marcus Chen', authorId: 'u-marcus', authorColor: 'purple', reviewer: false, version: 14, mentions: [], createdAt: '2026-10-25T09:36:00Z' }],
}
const reviewerThread = {
  id: 't2',
  anchor: '',
  anchorLabel: '',
  comments: [{ id: 'c2', body: 'Please add the cut-off time in UTC.', author: 'Ana from Legal', reviewer: true, version: 14, mentions: [], createdAt: '2026-10-25T09:40:00Z' }],
}

let unmock: (() => void) | undefined
afterEach(() => {
  unmock?.()
  vi.useRealTimers()
})

describe('comments', () => {
  it('lists open threads, replies with an @mention, and resolves', async () => {
    const posts: unknown[] = []
    const resolves: unknown[] = []
    unmock = mockApi({
      [`GET ${base}/comments`]: () => json({ threads: [marcusThread, reviewerThread, { ...marcusThread, id: 't3', resolvedAt: '2026-10-25T10:00:00Z', resolvedBy: 'Linh Nguyen' }], open: 2, resolved: 1 }),
      [`GET ${base}/mentionable`]: () => json(members),
      [`POST ${base}/comments`]: async (r) => {
        posts.push(await r.json())
        return json(marcusThread, 201)
      },
      [`PUT ${base}/comments/t1/resolved`]: async (r) => {
        resolves.push(await r.json())
        return new Response(null, { status: 204 })
      },
    })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer()}>
        <CommentsTab />
      </ComposerCtx.Provider>,
      { me: editor, seed: seedMe },
    )
    const thread = await screen.findByRole('article', { name: 'Comment thread on the button “Read the docs”' })
    expect(thread).toHaveTextContent('Marcus Chen')
    expect(thread).toHaveTextContent('On the button “Read the docs”')
    expect(thread).toHaveTextContent('Can we link the changelog instead?')
    // A reviewer's comment, through a review link, says so.
    const reviewer = screen.getByRole('article', { name: 'Comment thread on the whole email' })
    expect(within(reviewer).getByText('Reviewer')).toBeInTheDocument()
    expect(screen.getByText('1 resolved comment')).toBeInTheDocument()

    fireEvent.click(within(thread).getByRole('button', { name: 'Reply' }))
    const box = within(thread).getByLabelText('Reply to Marcus Chen')
    fireEvent.change(box, { target: { value: 'Done, thanks @So' } })
    const list = await within(thread).findByRole('listbox', { name: 'People to mention' })
    expect(within(list).getAllByRole('option')).toHaveLength(1)
    fireEvent.keyDown(box, { key: 'Enter' })
    expect(box).toHaveValue('Done, thanks @Sofia Rossi ')
    fireEvent.click(within(thread).getByRole('button', { name: 'Reply' }))
    await waitFor(() => expect(posts).toHaveLength(1))
    expect(posts[0]).toEqual({ body: 'Done, thanks @Sofia Rossi ', threadId: 't1', mentions: ['u-sofia'] })

    fireEvent.click(within(thread).getByRole('button', { name: 'Resolve' }))
    await waitFor(() => expect(resolves).toEqual([{ resolved: true }]))
    // The resolved ones, on request.
    fireEvent.click(screen.getByRole('button', { name: 'Show' }))
    expect(await screen.findByText('Resolved by Linh Nguyen')).toBeInTheDocument()
  })

  it('starts a thread on the selected block, which keeps its id', async () => {
    const { editor: ed } = await headlessMjmlEditor(
      '<mjml><mj-body><mj-section><mj-column><mj-button href="https://docs.acme.io/flags">Read the docs</mj-button></mj-column></mj-section></mj-body></mjml>',
    )
    const button = ed.getWrapper()!.findType('mj-button')[0] as Component
    const posts: { anchor: string; anchorLabel: string; body: string; mentions: string[] }[] = []
    unmock = mockApi({
      [`GET ${base}/comments`]: () => json({ threads: [], open: 0, resolved: 0 }),
      [`GET ${base}/mentionable`]: () => json(members),
      [`POST ${base}/comments`]: async (r) => {
        posts.push(await r.json())
        return json(marcusThread, 201)
      },
    })
    const save = vi.fn(async () => true)
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ editor: ed, selected: button, save })}>
        <CommentsTab />
      </ComposerCtx.Provider>,
      { me: editor, seed: seedMe },
    )
    expect(await screen.findByText('No comments yet. Select a block to comment on it, or comment on the whole email.')).toBeInTheDocument()
    expect(screen.getByText('On the button “Read the docs”')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('New comment'), { target: { value: 'Link the changelog?' } })
    fireEvent.click(screen.getByRole('button', { name: 'Comment' }))
    await waitFor(() => expect(posts).toHaveLength(1))
    const anchor = (button.getAttributes() as Record<string, string>)[COMMENT_ATTR]
    expect(anchor).toMatch(/^[a-z0-9]{8}$/)
    expect(posts[0]).toEqual({ body: 'Link the changelog?', anchor, anchorLabel: 'the button “Read the docs”', mentions: [] })
    // The block's new marker is saved with the content.
    await waitFor(() => expect(save).toHaveBeenCalledWith({ background: true }))
  })

  it("lets viewers comment, and says who can resolve", async () => {
    unmock = mockApi({
      [`GET ${base}/comments`]: () => json({ threads: [marcusThread], open: 1, resolved: 0 }),
      [`GET ${base}/mentionable`]: () => json(members),
    })
    const campaign = testCampaign({ access: { ...testCampaign().access, canEdit: false, editable: true } })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, canEdit: false })}>
        <CommentsTab />
      </ComposerCtx.Provider>,
      { seed: seedMe },
    )
    const thread = await screen.findByRole('article', { name: 'Comment thread on the button “Read the docs”' })
    expect(within(thread).queryByRole('button', { name: 'Resolve' })).toBeNull()
    expect(within(thread).getByRole('button', { name: 'Reply' })).toBeInTheDocument()
    expect(screen.getByText('People who edit campaigns can resolve comments.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Comment' })).toBeDisabled()
  })

  it('reads comment times like the board', () => {
    const now = new Date('2026-10-26T12:00:00Z')
    expect(commentTime('2026-10-26T09:36:00Z', now)).toBe('09:36')
    expect(commentTime('2026-10-25T16:12:00Z', now)).toBe('Yesterday 16:12')
  })

  it('puts sections someone else saved into the canvas', async () => {
    const { editor: ed } = await headlessMjmlEditor(
      '<mjml><mj-body><mj-section><mj-column><mj-text>My hero</mj-text></mj-column></mj-section><mj-section><mj-column><mj-text>Footer</mj-text></mj-column></mj-section></mj-body></mjml>',
    )
    expect(applySections(ed, [{ index: 1, mjml: '<mj-section><mj-column><mj-text>New footer</mj-text></mj-column></mj-section>' }])).toBe(1)
    const html = ed.getHtml()
    expect(html).toContain('My hero')
    expect(html).toContain('New footer')
    expect(html).not.toContain('>Footer<')
    expect(html.indexOf('My hero')).toBeLessThan(html.indexOf('New footer'))
  })
})
