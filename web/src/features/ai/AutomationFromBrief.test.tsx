import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { AutomationFromBriefDialog } from './AutomationFromBrief'

const base = '/api/v1/workspaces/acme-newsletter'
const endpoint = `${base}/ai/drafts/automations`
const editor = workspaceMe({ permissions: ['ai:use', 'automations:view', 'automations:edit', 'audience:view', 'templates:view'] })
const status = { state: 'ready', providerName: 'Fixture', modelName: 'Local model', features: [{ key: 'draft', on: true, available: true }], prices: { input: 1, output: 2 }, cap: { kind: 'usd', usd: 10 }, usage: { costMicros: 0 } }
const graph = {
  nodes: [
    { id: 'start', type: 'trigger', data: { trigger: { kind: 'joined_list', listId: 'list-1', confirmedOnly: true, includeImports: false, reentry: 'once', exitOnLeave: true } } },
    { id: 'intro', type: 'email', data: { email: { emailId: 'welcome', emailVersion: 1, subject: 'Welcome to exports' } } },
    { id: 'branch', type: 'branch', data: { condition: { kind: 'clicked', step: 'intro', within: { amount: 3, unit: 'days' }, humanOnly: true } } },
    { id: 'tip', type: 'email', data: { email: { emailId: 'tip', emailVersion: 1, subject: 'Your second tip' } } },
    { id: 'end', type: 'exit', data: {} },
  ],
  edges: [
    { id: 'a', source: 'start', target: 'intro', data: {} },
    { id: 'b', source: 'intro', target: 'branch', data: {} },
    { id: 'c', source: 'branch', target: 'tip', data: { path: 'yes' } },
    { id: 'd', source: 'branch', target: 'end', data: { path: 'no' } },
    { id: 'e', source: 'tip', target: 'end', data: {} },
  ],
}
const proposal = { id: 'proposal-1', name: 'Export welcome', listId: 'list-1', content: { graph, emails: [
  { key: 'welcome', name: 'Introduction', subject: 'Welcome to exports', preheader: 'Start here', mjml: '<mjml><mj-body /></mjml>' },
  { key: 'tip', name: 'Export shortcut', subject: 'Your second tip', preheader: 'Pick your columns', mjml: '<mjml><mj-body /></mjml>' },
] }, beforeEmails: [], previews: ['<html><body>Welcome</body></html>', '<html><body>Shortcut</body></html>'], sent: { modelName: 'Local model', inputTokens: 120, outputTokens: 500, costMicros: 1100 }, expiresAt: '2026-11-16T09:00:00Z', decision: '' }
const reads = {
  [`GET ${endpoint}`]: () => json({ items: [] }),
  [`GET ${base}/ai/status`]: () => json(status),
  [`GET ${base}/brand-kit/voice/history`]: () => json({ items: [] }),
  [`GET ${base}/lists`]: () => json({ items: [{ id: 'list-1', name: 'Weekly readers' }] }),
  [`GET ${base}/templates`]: () => json({ starters: [], templates: [] }),
}
let unmock: (() => void) | undefined
afterEach(() => unmock?.())

async function generate() {
  const button = await screen.findByRole('button', { name: 'Generate proposal' })
  await waitFor(() => expect(button).toBeEnabled())
  fireEvent.change(screen.getByLabelText('What should this sequence do?'), { target: { value: 'Welcome new readers, follow up after three days if they clicked.' } })
  fireEvent.change(screen.getByLabelText('Automation name'), { target: { value: 'Export welcome' } })
  fireEvent.change(screen.getByLabelText('Selected list'), { target: { value: 'list-1' } })
  fireEvent.click(button)
  fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
  return screen.findByRole('button', { name: 'Accept and save draft' })
}

describe('automation from a brief', () => {
  it('recovers an earlier proposal and does not apply a decision made in another tab', async () => {
    let decision = 'accepted'
    const accepted = vi.fn()
    unmock = mockApi({ ...reads,
      [`GET ${endpoint}`]: () => json({ items: [{ id: proposal.id, name: proposal.name, expiresAt: proposal.expiresAt }] }),
      [`GET ${endpoint}/proposal-1`]: () => json({ ...proposal, decision }),
    })
    await renderInWorkspace(<AutomationFromBriefDialog onClose={vi.fn()} onAccepted={accepted} />, { me: editor })
    fireEvent.click(await screen.findByRole('button', { name: 'Review proposal: Export welcome' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('already been decided')
    expect(screen.queryByRole('button', { name: 'Accept and save draft' })).not.toBeInTheDocument()
    decision = ''
    const review = screen.getByRole('button', { name: 'Review proposal: Export welcome' })
    await waitFor(() => expect(review).toBeEnabled())
    fireEvent.click(review)
    expect(await screen.findByRole('region', { name: 'Steps and paths · Proposed · not saved' })).toHaveTextContent('Export shortcut')
    expect(accepted).not.toHaveBeenCalled()
  })

  it('reviews both branch paths and every email before saving the bundle', async () => {
    const requests: unknown[] = []
    const decisions: unknown[] = []
    const accepted = vi.fn()
    unmock = mockApi({ ...reads,
      [`POST ${endpoint}`]: async (r) => { requests.push(await r.json()); return json(proposal) },
      [`POST ${endpoint}/proposal-1/decision`]: async (r) => { decisions.push(await r.json()); return json({ decision: 'accepted', id: 'flow-1', revision: 1 }) },
    })
    await renderInWorkspace(<AutomationFromBriefDialog onClose={vi.fn()} onAccepted={accepted} />, { me: editor })
    const accept = await generate()
    expect(requests).toEqual([{ sharingReviewed: true, brief: 'Welcome new readers, follow up after three days if they clicked.', name: 'Export welcome', language: 'en', listId: 'list-1', templateId: '', links: [], baseRevision: 0 }])
    expect(decisions).toEqual([])
    const review = within(screen.getByRole('region', { name: 'Steps and paths · Proposed · not saved' }))
    expect(review.getByText('Yes → Next: Export shortcut')).toBeInTheDocument()
    expect(review.getByText(/No → Next:/)).toBeInTheDocument()
    expect(screen.getByText('Welcome to exports')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Export shortcut', { selector: 'summary' }))
    expect(screen.getByText('Your second tip')).toBeInTheDocument()
    expect(screen.getAllByTitle('Proposed email preview')).toHaveLength(2)
    fireEvent.click(accept)
    await waitFor(() => expect(accepted).toHaveBeenCalledWith('flow-1'))
    expect(decisions).toEqual([{ decision: 'accepted' }])
  })

  it('keeps a stale proposal reviewable, blocks unsaved edits and permits rejection', async () => {
    let dirty = false
    const decisions: unknown[] = []
    const accepted = vi.fn()
    unmock = mockApi({ ...reads,
      [`POST ${endpoint}`]: () => json({ ...proposal, before: graph, automationId: 'flow-1', baseRevision: 4 }),
      [`POST ${endpoint}/proposal-1/decision`]: async (r) => {
        const body = await r.json()
        decisions.push(body)
        return body.decision === 'accepted' ? json({ status: 409, detail: 'Changed since this proposal.' }, 409) : json({ decision: 'rejected' })
      },
    })
    await renderInWorkspace(<AutomationFromBriefDialog onClose={vi.fn()} onAccepted={accepted} target={{ id: 'flow-1', name: 'Export welcome', listId: 'list-1', baseRevision: 4, hasUnsaved: () => dirty }} />, { me: editor })
    const accept = await generate()
    expect(screen.getByRole('region', { name: 'Steps and paths · Before · saved content' })).toBeInTheDocument()
    dirty = true
    fireEvent.click(accept)
    expect(await screen.findByRole('alert')).toHaveTextContent('Wait for the flow to save')
    expect(decisions).toEqual([])
    dirty = false
    fireEvent.click(accept)
    expect(await screen.findByRole('alert')).toHaveTextContent('Changed since this proposal.')
    expect(accepted).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Reject proposal' }))
    await screen.findByText('Proposal rejected. No flow or email was changed.')
    expect(decisions).toEqual([{ decision: 'accepted' }, { decision: 'rejected' }])
  })

  it('does not generate without normal automation edit permission', async () => {
    unmock = mockApi(reads)
    await renderInWorkspace(<AutomationFromBriefDialog onClose={vi.fn()} onAccepted={vi.fn()} />, { me: workspaceMe({ permissions: ['ai:use', 'automations:view', 'audience:view'] }) })
    expect(await screen.findByRole('button', { name: 'Generate proposal' })).toBeDisabled()
  })
})
