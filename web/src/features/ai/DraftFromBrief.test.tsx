import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { DraftFromBriefDialog } from './DraftFromBrief'

const base = '/api/v1/workspaces/acme-newsletter'
const endpoint = `${base}/ai/drafts/campaigns`
const editor = workspaceMe({ permissions: ['ai:use', 'campaigns:view', 'campaigns:edit', 'templates:view'] })
const draft = { key: 'draft', on: true, available: true, state: 'ready', personal: false, providerName: 'Fixture', modelName: 'Local model', prices: { input: 1, output: 2 }, cap: { kind: 'usd', usd: 10 }, usage: { costMicros: 0 } }
const status = { state: 'ready', providerName: 'Fixture', modelName: 'Local model', features: [draft], prices: { input: 1, output: 2 }, cap: { kind: 'usd', usd: 10 }, usage: { costMicros: 0 } }
const content = { subject: 'Faster exports', preheader: 'Try the new export', mjml: '<mjml><mj-body><mj-section><mj-column><mj-text>New export</mj-text></mj-column></mj-section></mj-body></mjml>' }
const proposal = { id: 'proposal-1', name: 'Export update', content, html: '<html><head></head><body><p>New export</p><a href="https://example.test">Read</a></body></html>', subjectCheck: { heuristics: [] }, sent: { modelName: 'Local model', inputTokens: 120, outputTokens: 90, costMicros: 1000, voiceRevision: 3 }, expiresAt: '2026-11-16T09:00:00Z', decision: '' }
const reads = {
  [`GET ${endpoint}`]: () => json({ items: [] }),
  [`GET ${base}/ai/status`]: () => json(status),
  [`GET ${base}/brand-kit/voice/history`]: () => json({ items: [] }),
  [`GET ${base}/templates`]: () => json({ starters: [], templates: [] }),
}
let unmock: (() => void) | undefined
afterEach(() => unmock?.())

describe('draft from a brief', () => {
  it('recovers a saved proposal without AI access or another generation request', async () => {
    const accepted = vi.fn()
    unmock = mockApi({ ...reads,
      [`GET ${endpoint}`]: () => json({ items: [{ id: proposal.id, name: proposal.name, expiresAt: proposal.expiresAt }] }),
      [`GET ${endpoint}/proposal-1`]: () => json(proposal),
      [`POST ${endpoint}/proposal-1/decision`]: () => json({ decision: 'accepted', id: 'campaign-1', revision: 1 }),
    })
    await renderInWorkspace(<DraftFromBriefDialog onClose={vi.fn()} onAccepted={accepted} />, { me: workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] }) })
    fireEvent.click(await screen.findByRole('button', { name: 'Review proposal: Export update' }))
    const accept = await screen.findByRole('button', { name: 'Accept and save draft' })
    expect(screen.getByText('Faster exports')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Review later' })).toBeEnabled()
    expect(accepted).not.toHaveBeenCalled()
    fireEvent.click(accept)
    await waitFor(() => expect(accepted).toHaveBeenCalledWith('campaign-1'))
  })

  it('shows an inert proposal and writes only after explicit acceptance', async () => {
    const requests: unknown[] = []
    const decisions: unknown[] = []
    const accepted = vi.fn()
    unmock = mockApi({
      ...reads,
      [`POST ${endpoint}`]: async (r) => { requests.push(await r.json()); return json(proposal) },
      [`POST ${endpoint}/proposal-1/decision`]: async (r) => { decisions.push(await r.json()); return json({ decision: 'accepted', id: 'campaign-1', revision: 1 }) },
    })
    await renderInWorkspace(<DraftFromBriefDialog onClose={vi.fn()} onAccepted={accepted} />, { me: editor })
    const generate = await screen.findByRole('button', { name: 'Generate proposal' })
    await waitFor(() => expect(generate).toBeEnabled())
    fireEvent.change(screen.getByLabelText('What should this email say?'), { target: { value: 'Explain faster exports.' } })
    fireEvent.change(screen.getByLabelText('Draft name'), { target: { value: 'Export update' } })
    fireEvent.change(screen.getByLabelText('Links to preserve'), { target: { value: 'https://example.test/read' } })
    fireEvent.click(generate)
    const sharing = within(await screen.findByRole('dialog', { name: 'Review before sharing' }))
    expect(sharing.getByText(/Fixture · Local model/)).toBeVisible()
    expect(sharing.getByText(/can miss personal information/)).toBeVisible()
    expect(requests).toEqual([])
    fireEvent.click(sharing.getByRole('button', { name: 'Cancel' }))
    expect(requests).toEqual([])
    fireEvent.change(screen.getByLabelText('What should this email say?'), { target: { value: 'Explain simpler exports.' } })
    fireEvent.click(generate)
    const confirm = await screen.findByRole('button', { name: 'Share reviewed content' })
    fireEvent.click(confirm)
    fireEvent.click(confirm)
    const accept = await screen.findByRole('button', { name: 'Accept and save draft' })
    expect(requests).toEqual([{ sharingReviewed: true, brief: 'Explain simpler exports.', name: 'Export update', language: 'en', templateId: '', links: ['https://example.test/read'], baseRevision: 0 }])
    expect(decisions).toEqual([])
    expect(accepted).not.toHaveBeenCalled()
    expect(screen.getByRole('region', { name: 'Proposed · not saved' })).toHaveTextContent('Faster exports')
    const iframe = screen.getByTitle('Proposed email preview')
    expect(iframe).toHaveAttribute('sandbox', '')
    expect(iframe.getAttribute('srcdoc')).toContain("default-src 'none'")
    expect(iframe.getAttribute('srcdoc')).not.toContain('href=')
    fireEvent.click(accept)
    await waitFor(() => expect(accepted).toHaveBeenCalledWith('campaign-1'))
    expect(decisions).toEqual([{ decision: 'accepted' }])
  })

  it('preserves a conflicting proposal and never accepts over unsaved editor changes', async () => {
    let dirty = false
    const decisions: unknown[] = []
    const accepted = vi.fn()
    const before = { ...content, subject: 'Original subject' }
    unmock = mockApi({
      ...reads,
      [`POST ${endpoint}`]: () => json({ ...proposal, before, campaignId: 'existing', baseRevision: 7 }),
      [`POST ${endpoint}/proposal-1/decision`]: async (r) => {
        const body = await r.json()
        decisions.push(body)
        return body.decision === 'accepted' ? json({ status: 409, detail: 'Changed since this proposal.' }, 409) : json({ decision: 'rejected' })
      },
    })
    await renderInWorkspace(<DraftFromBriefDialog onClose={vi.fn()} onAccepted={accepted} target={{ id: 'existing', name: 'Existing draft', baseRevision: 7, hasUnsaved: () => dirty }} />, { me: editor })
    const generate = await screen.findByRole('button', { name: 'Generate proposal' })
    await waitFor(() => expect(generate).toBeEnabled())
    fireEvent.change(screen.getByLabelText('What should this email say?'), { target: { value: 'Make it shorter.' } })
    fireEvent.click(generate)
    fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
    const accept = await screen.findByRole('button', { name: 'Accept and save draft' })
    expect(within(screen.getByRole('region', { name: 'Before · saved content' })).getByText('Original subject')).toBeInTheDocument()
    dirty = true
    fireEvent.click(accept)
    expect(await screen.findByRole('alert')).toHaveTextContent('Save or discard your composer edits')
    expect(decisions).toEqual([])
    dirty = false
    fireEvent.click(accept)
    expect(await screen.findByRole('alert')).toHaveTextContent('Changed since this proposal.')
    expect(accepted).not.toHaveBeenCalled()
    expect(screen.getByText('Faster exports')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Reject proposal' }))
    await screen.findByText('Proposal rejected. No campaign was changed.')
    expect(decisions).toEqual([{ decision: 'accepted' }, { decision: 'rejected' }])
  })

  it('closes the sharing confirmation while generating so the person can stop waiting', async () => {
    let release: ((response: Response) => void) | undefined
    let signal: AbortSignal | undefined
    unmock = mockApi({ ...reads,
      [`POST ${endpoint}`]: (request) => {
        signal = request.signal
        return new Promise<Response>((resolve) => { release = resolve })
      },
    })
    await renderInWorkspace(<DraftFromBriefDialog onClose={vi.fn()} onAccepted={vi.fn()} />, { me: editor })
    const generate = await screen.findByRole('button', { name: 'Generate proposal' })
    await waitFor(() => expect(generate).toBeEnabled())
    fireEvent.change(screen.getByLabelText('What should this email say?'), { target: { value: 'Explain exports.' } })
    fireEvent.change(screen.getByLabelText('Draft name'), { target: { value: 'Export update' } })
    fireEvent.click(generate)
    fireEvent.click(await screen.findByRole('button', { name: 'Share reviewed content' }))
    const stop = await screen.findByRole('button', { name: 'Stop waiting' })
    expect(screen.queryByRole('dialog', { name: 'Review before sharing' })).not.toBeInTheDocument()
    await waitFor(() => expect(signal).toBeDefined())
    fireEvent.click(stop)
    expect(signal!.aborted).toBe(true)
    release!(json(proposal))
    await waitFor(() => expect(screen.getByLabelText('What should this email say?')).toHaveValue('Explain exports.'))
    expect(screen.queryByRole('button', { name: 'Accept and save draft' })).not.toBeInTheDocument()
  })

  it('does not allow model generation without edit permission', async () => {
    unmock = mockApi(reads)
    await renderInWorkspace(<DraftFromBriefDialog onClose={vi.fn()} onAccepted={vi.fn()} />, { me: workspaceMe({ permissions: ['ai:use', 'campaigns:view'] }) })
    expect(await screen.findByRole('button', { name: 'Generate proposal' })).toBeDisabled()
  })

  it('generates on the person’s own model for drafts while the workspace is at its cap', async () => {
    const own = { ...draft, personal: true, providerName: 'Own', modelName: 'Own model', cap: { kind: 'usd', usd: 5 }, usage: { costMicros: 1_000_000 } }
    unmock = mockApi({ ...reads, [`GET ${base}/ai/status`]: () => json({ ...status, state: 'cap_reached', features: [own] }) })
    await renderInWorkspace(<DraftFromBriefDialog onClose={vi.fn()} onAccepted={vi.fn()} />, { me: editor })
    const generate = await screen.findByRole('button', { name: 'Generate proposal' })
    await waitFor(() => expect(generate).toBeEnabled())
    expect(screen.getByText('Own · Own model')).toBeVisible()
    expect(screen.getByText(/^Own model · \$1\.00 of \$5\.00 this month/)).toBeVisible()
  })

  it('does not generate when the draft’s own model is at its cap, even with the workspace ready', async () => {
    const spent = { ...draft, personal: true, state: 'cap_reached', usage: { costMicros: 5_000_000, resetsAt: '2026-12-01T00:00:00Z' } }
    unmock = mockApi({ ...reads, [`GET ${base}/ai/status`]: () => json({ ...status, features: [spent] }) })
    await renderInWorkspace(<DraftFromBriefDialog onClose={vi.fn()} onAccepted={vi.fn()} />, { me: editor })
    expect(await screen.findByText(/This month's AI budget is used up/)).toBeVisible()
    expect(screen.getByRole('button', { name: 'Generate proposal' })).toBeDisabled()
  })
})
