import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { PresenceChip, ShareReviewButton } from './Collab'
import { ComposerCtx } from './context'
import { testCampaign, testComposer } from './testing'

// CO-10: who else has the campaign open, and Share for review.

const base = '/api/v1/workspaces/acme-newsletter/campaigns/01970000-0000-7000-8000-0000000000c1'
const editor = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] })

let unmock: (() => void) | undefined
afterEach(() => unmock?.())

const link = {
  id: 'l1',
  url: 'https://reputecast.acme.io/review/k3Jx9vQ2m8ZtYbWc4RpLs7NfHd6GaUe1VoXi5TqKy0E',
  version: 14,
  createdBy: 'Linh Nguyen',
  createdAt: new Date().toISOString(),
  expiresAt: new Date(Date.now() + 7 * 86_400_000).toISOString(),
}

describe('presence', () => {
  it('shows who else has the campaign open, and says when you leave', async () => {
    const pings: unknown[] = []
    let left = 0
    unmock = mockApi({
      [`PUT ${base}/presence`]: async (r) => {
        pings.push(await r.json())
        return json({ others: [{ userId: 'u-marcus', name: 'Marcus Chen', color: 'purple', editing: false, seenAt: '2026-10-26T09:38:00Z' }] })
      },
      [`DELETE ${base}/presence`]: () => {
        left++
        return new Response(null, { status: 204 })
      },
    })
    const { unmount } = await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer()}>
        <PresenceChip />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    const group = await screen.findByRole('group', { name: 'Also in this campaign' })
    expect(screen.getByRole('img', { name: 'Marcus Chen, viewing' })).toBeInTheDocument()
    expect(group).toHaveTextContent('Marcus Chen viewing')
    expect(pings).toEqual([{ editing: false }])
    unmount()
    await waitFor(() => expect(left).toBe(1))
  })
})

describe('share for review', () => {
  it('makes a link when there is none, copies it, and revokes it', async () => {
    let created = 0
    let revoked = 0
    unmock = mockApi({
      [`GET ${base}/review-links`]: () => json({ items: [] }),
      [`POST ${base}/review-links`]: () => {
        created++
        return json(link, 201)
      },
      [`DELETE ${base}/review-links/l1`]: () => {
        revoked++
        return new Response(null, { status: 204 })
      },
    })
    const writeText = vi.fn(async () => {})
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer()}>
        <ShareReviewButton />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    fireEvent.click(screen.getByRole('button', { name: 'Share for review' }))
    const dialog = await screen.findByRole('dialog', { name: 'Share for review' })
    expect(dialog).toHaveTextContent('Anyone with the link can view and comment without signing in.')
    expect(await screen.findByLabelText('Review link')).toHaveValue(link.url)
    expect(created).toBe(1)
    expect(dialog).toHaveTextContent('Expires in 7 days. Reviewers see this email only, not your audience, settings or other campaigns.')
    expect(dialog).toHaveTextContent('Created by Linh Nguyen just now · shows v14')
    fireEvent.click(screen.getByRole('button', { name: 'Copy link' }))
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(link.url))
    fireEvent.click(screen.getByRole('button', { name: 'Revoke' }))
    expect(await screen.findByText('Link revoked. Anyone who opens it sees that it no longer works.')).toBeInTheDocument()
    expect(revoked).toBe(1)
    expect(screen.getByRole('button', { name: 'Create a new link' })).toBeInTheDocument()
  })

  it('shows the newest link, and offers the latest version when it is older', async () => {
    unmock = mockApi({ [`GET ${base}/review-links`]: () => json({ items: [{ ...link, version: 12 }] }) })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer()}>
        <ShareReviewButton />
      </ComposerCtx.Provider>,
      { me: editor },
    )
    fireEvent.click(screen.getByRole('button', { name: 'Share for review' }))
    expect(await screen.findByText('Reviewers see v12. You have saved v14 since.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Share v14 instead' })).toBeInTheDocument()
  })

  it('is not offered to people who can’t edit the campaign', async () => {
    const campaign = testCampaign({ access: { ...testCampaign().access, canEdit: false } })
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, canEdit: false })}>
        <ShareReviewButton />
      </ComposerCtx.Provider>,
    )
    expect(screen.queryByRole('button', { name: 'Share for review' })).toBeNull()
  })
})
