import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace } from '@/test/app'
import { ReviewLinkPage } from './ReviewLinkPage'

// CO-10: the page behind a review link, for someone without an account.

const token = 'k3Jx9vQ2m8ZtYbWc4RpLs7NfHd6GaUe1VoXi5TqKy0E'

const page = {
  workspace: 'Acme Newsletter',
  campaign: 'Weekly Digest #142',
  version: 15,
  subject: 'Three launches, one deprecation, and the roadmap for Q4',
  preheader: "Feature flags are GA, audit log exports, and what we're retiring in November.",
  html: '<html><head></head><body><p>Hi Sam,</p></body></html>',
  sample: 'Sam Rivera',
  fromName: 'Acme Newsletter',
  expiresAt: '2026-11-03T09:40:00Z',
  threads: [
    {
      id: 't1',
      anchor: '',
      anchorLabel: '',
      comments: [
        { id: 'c1', body: 'Please add the cut-off time in UTC.', author: 'Ana from Legal', reviewer: true, version: 15, mentions: [], createdAt: '2026-10-27T09:40:00Z' },
        { id: 'c2', body: 'Added, thanks.', author: 'Linh Nguyen', authorColor: 'info', reviewer: false, version: 15, mentions: [], createdAt: '2026-10-27T09:45:00Z' },
      ],
    },
  ],
}

let unmock: (() => void) | undefined
afterEach(() => {
  unmock?.()
  try {
    window.localStorage.clear()
  } catch {
    // no storage
  }
})

describe('review link page', () => {
  it('shows the email and the comments, and posts a comment signed with a name', async () => {
    const posts: unknown[] = []
    unmock = mockApi({
      [`GET /api/v1/review/${token}`]: () => json(page),
      [`POST /api/v1/review/${token}/comments`]: async (r) => {
        posts.push(await r.json())
        return json(page.threads[0], 201)
      },
    })
    await renderInWorkspace(<ReviewLinkPage token={token} />, { path: `/review/${token}` })
    expect(await screen.findByRole('heading', { name: 'Weekly Digest #142' })).toBeInTheDocument()
    expect(screen.getByText('Shared by Acme Newsletter')).toBeInTheDocument()
    expect(screen.getByText('Three launches, one deprecation, and the roadmap for Q4')).toBeInTheDocument()
    expect(screen.getByTitle('The email')).toBeInTheDocument()
    expect(screen.getByText('Personalized for a sample reader, Sam Rivera.')).toBeInTheDocument()
    const thread = screen.getByRole('article', { name: 'Comment by Ana from Legal' })
    expect(within(thread).getByText('Reviewer')).toBeInTheDocument()
    expect(thread).toHaveTextContent('Added, thanks.')
    expect(document.querySelector('meta[name="robots"]')).toHaveAttribute('content', 'noindex, nofollow')
    expect(document.querySelector('meta[name="referrer"]')).toHaveAttribute('content', 'no-referrer')

    const form = screen.getByRole('heading', { name: 'Comments' }).closest('section') as HTMLElement
    const post = within(form).getAllByRole('button', { name: 'Comment' })[0]!
    expect(post).toBeDisabled()
    fireEvent.change(within(form).getAllByLabelText('Your name')[0]!, { target: { value: 'Ana from Legal' } })
    fireEvent.change(within(form).getAllByLabelText('Comment')[0]!, { target: { value: 'The footer address is old.' } })
    fireEvent.click(post)
    await waitFor(() => expect(posts).toEqual([{ name: 'Ana from Legal', body: 'The footer address is old.' }]))
    // The name is remembered for the next comment.
    expect(window.localStorage.getItem('rc-review-name')).toBe('Ana from Legal')
  })

  it('says when the link no longer works', async () => {
    unmock = mockApi({
      [`GET /api/v1/review/${token}`]: () => json({ title: 'Not Found', status: 404, detail: 'This review link has expired or was revoked.', code: 'not_found' }, 404),
    })
    await renderInWorkspace(<ReviewLinkPage token={token} />, { path: `/review/${token}` })
    expect(await screen.findByText('This review link no longer works')).toBeInTheDocument()
    expect(screen.getByText('It expired or was revoked. Ask whoever shared it for a new one.')).toBeInTheDocument()
  })
})
