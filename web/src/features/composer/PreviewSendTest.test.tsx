import type { QueryClient } from '@tanstack/react-query'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { meQueryKey } from '@/lib/session'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ComposerCtx } from './context'
import { PreviewView } from './PreviewView'
import { testCampaign, testComposer } from './testing'

// R4.L5, CO-5: Send test from Preview goes out personalized as the reader
// the preview shows: the "Preview as" contact (Mary Ann stays whole), or
// the reader without a first name (who gets "Hi there").

const campaign = testCampaign()
const base = `/api/v1/workspaces/acme-newsletter/campaigns/${campaign.id}`
const me = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit', 'subscribers:view'] })
const seedMe = (qc: QueryClient) =>
  qc.setQueryData(meQueryKey, { user: { id: 'u-linh', email: 'linh@acme.io', name: 'Linh Nguyen' }, workspaces: [], instance: {} } as never)

const contacts = [
  { id: 'c-mary', name: 'Mary Ann Smith', firstName: 'Mary Ann', lastName: 'Smith', email: 'maryann@example.com', label: 'First name set', language: '' },
  { id: 'c-nguyen', name: 'Nguyen', firstName: '', lastName: 'Nguyen', email: 'nguyen@example.com', label: 'No first name', language: '' },
]

/** What the server's preview answers for a pick, as campaigns.Preview does. */
function previewFor(body: { sample?: string; contactId?: string }) {
  const who = body.sample === 'no_first_name' ? { name: '', firstName: '', lastName: '', email: 'someone@example.com', label: 'No first name' } : (contacts.find((c) => c.id === body.contactId) ?? contacts[0]!)
  const first = who.firstName || 'there'
  return {
    subject: `Hi ${first}`,
    preheader: '',
    html: `<p>Hi ${first}</p>`,
    text: `Hi ${first}`,
    fromName: 'Acme',
    fromEmail: 'news@acme.io',
    contact: who,
    tokens: [],
    problems: [],
    renderedAt: '2026-10-25T09:40:00Z',
    language: '',
    languageReason: '',
  }
}

describe('Send test from Preview', () => {
  let unmock: (() => void) | undefined
  // jsdom has no ResizeObserver; the preview frames size themselves with one.
  beforeAll(() => {
    vi.stubGlobal(
      'ResizeObserver',
      class {
        observe() {}
        unobserve() {}
        disconnect() {}
      },
    )
  })
  afterAll(() => vi.unstubAllGlobals())
  afterEach(() => {
    unmock?.()
    document.body.innerHTML = ''
  })

  it('goes out as the contact the preview shows, or as the reader without a first name', async () => {
    const tests: Record<string, unknown>[] = []
    const routes: Record<string, (r: Request) => Response | Promise<Response>> = {
      [`GET ${base}/preview-contacts`]: () => json({ items: contacts }),
      [`POST ${base}/preview`]: async (r) => json(previewFor((await r.json()) as { sample?: string; contactId?: string })),
      [`POST ${base}/test`]: async (r) => {
        tests.push((await r.json()) as Record<string, unknown>)
        return json({ sent: ['linh@acme.io'], provider: 'Amazon SES', languages: [] })
      },
    }
    // Everything else the preview's cards ask for isn't part of this test.
    unmock = mockApi(new Proxy(routes, { get: (t, k: string) => t[k] ?? (() => json({ title: 'Not Found', status: 404, detail: 'Not in this test.' }, 404)) }))
    const slot = document.createElement('div')
    document.body.appendChild(slot)
    const save = vi.fn(async () => true)
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, previewSlot: slot, save })}>
        <PreviewView onReview={() => {}} />
      </ComposerCtx.Provider>,
      { me, seed: seedMe },
    )
    const send = await screen.findByRole('button', { name: 'Send test to linh@acme.io' })
    const previewAs = screen.getByRole('combobox', { name: /Preview as/ })

    // Without a pick, the preview shows the audience's first contact: Mary Ann.
    await waitFor(() => expect(previewAs).toHaveValue('c:c-mary'))
    fireEvent.click(send)
    await waitFor(() => expect(tests).toHaveLength(1))
    expect(save).toHaveBeenCalled()
    expect(tests[0]).toMatchObject({ to: ['linh@acme.io'], contactId: 'c-mary' })
    expect(tests[0]).not.toHaveProperty('sample')
    expect(await screen.findByText(/Personalized as Mary Ann Smith, via Amazon SES/)).toBeInTheDocument()

    // A contact without a first name.
    fireEvent.change(previewAs, { target: { value: 'c:c-nguyen' } })
    fireEvent.click(send)
    await waitFor(() => expect(tests).toHaveLength(2))
    expect(tests[1]).toMatchObject({ contactId: 'c-nguyen' })

    // The reader without a first name: the server fills in the fallbacks.
    fireEvent.change(previewAs, { target: { value: 'no_first_name' } })
    fireEvent.click(send)
    await waitFor(() => expect(tests).toHaveLength(3))
    expect(tests[2]).toMatchObject({ sample: 'no_first_name' })
    expect(tests[2]).not.toHaveProperty('contactId')
    expect(await screen.findByText(/Personalized as a reader without a first name, via Amazon SES/)).toBeInTheDocument()
  })
})
