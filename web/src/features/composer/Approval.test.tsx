import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import type { Editor } from 'grapesjs'
import { runShortcutHandler } from '@/lib/shortcuts'
import { json, mockApi } from '@/test/api'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ApprovalBanner, ApprovalDialog, type Approval } from './Approval'
import { ComposerCtx, type Campaign, type Checks } from './context'
import type * as Grapes from './editor/grapes'
import { lockCanvas, unlockCanvas } from './editor/lock'
import { headlessMjmlEditor } from './editor/testEditor'
import { ChecksRunBar } from './RightPanel'
import { SendTestButton, testBlockedReason } from './SendTest'
import { renderComposerPage, workspaceBase } from './testPage'
import { testCampaign, testComposer } from './testing'

// The whole composer renders on a headless editor (testPage.tsx).
vi.mock('./editor/grapes', async (importOriginal) => ({
  ...(await importOriginal<typeof Grapes>()),
  createComposerEditor: (await import('./editor/testEditor')).headlessComposerEditor,
}))

// CO-7: editing an approved campaign asks first; Send test waits for a
// successful save; checks that couldn't run show as stale with a Retry.

const approved: Approval = {
  id: '01970000-0000-7000-8000-00000000a001',
  status: 'approved',
  requestedBy: 'Daniel Park',
  requestedAt: '2026-10-25T09:30:00Z',
  approvedBy: ['Marcus Chen'],
  decidedAt: '2026-10-25T09:52:00Z',
  version: 16,
}

describe('an approved campaign', () => {
  it('opens with a banner that leads to the question', () => {
    const onEdit = vi.fn()
    render(<ApprovalBanner approval={approved} onEdit={onEdit} />)
    expect(screen.getByRole('status')).toHaveTextContent(/Approved by Marcus Chen at .* · v16\./)
    expect(screen.getByRole('status')).toHaveTextContent('Editing cancels the approval, and the campaign needs approval again before it can send.')
    fireEvent.click(screen.getByRole('button', { name: 'Edit' }))
    expect(onEdit).toHaveBeenCalled()
  })

  it('asks before editing, as the board words it', async () => {
    const onEdit = vi.fn(async () => {})
    render(<ApprovalDialog approval={approved} scheduledAt="2026-10-30T09:00:00Z" open onOpenChange={() => {}} onEdit={onEdit} />)
    const dialog = screen.getByRole('dialog', { name: 'Edit an approved campaign?' })
    expect(within(dialog).getByText('Approval will be cancelled')).toBeInTheDocument()
    expect(dialog).toHaveTextContent("Editing cancels Marcus Chen's approval. The campaign needs approval again before it can send.")
    expect(dialog).toHaveTextContent(/By Marcus Chen at .* · snapshot v16/)
    expect(dialog).toHaveTextContent("stays in place, but the send won't start without a new approval")
    expect(dialog).toHaveTextContent('When you request approval again, Marcus Chen get what changed since v16')
    fireEvent.click(within(dialog).getByRole('button', { name: 'Edit anyway' }))
    await waitFor(() => expect(onEdit).toHaveBeenCalled())
  })

  it('words a pending request as withdrawn', () => {
    render(<ApprovalDialog approval={{ ...approved, status: 'pending', approvedBy: [], decidedAt: undefined }} open onOpenChange={() => {}} onEdit={() => {}} />)
    const dialog = screen.getByRole('dialog', { name: 'Edit a campaign waiting for approval?' })
    expect(dialog).toHaveTextContent('Editing withdraws the approval request Daniel Park sent.')
  })
})

// R4.L7: after "Edit anyway", the save that cancels the approval brings the
// campaign back as the server has it, so the header stops saying "Approved";
// a later approval is asked about again.
describe('editing an approved campaign in the composer', () => {
  let unmock: (() => void) | undefined
  afterEach(() => unmock?.())

  it('stops saying "Approved" once the first save cancels the approval', async () => {
    const draft = testCampaign()
    let server: Campaign = { ...draft, status: 'approved', approval: approved }
    const saves: Record<string, unknown>[] = []
    const page = await renderComposerPage(() => server, {
      routes: {
        [`PUT ${workspaceBase}/campaigns/${draft.id}/content`]: async (r) => {
          const body = (await r.json()) as Record<string, unknown>
          saves.push(body)
          const version = 14 + saves.length
          // The server cancels the approval the person agreed to cancel.
          if (body.cancelApproval) server = { ...server, status: 'draft', approval: undefined }
          server = { ...server, version, subject: String(body.subject) }
          return json({ version, savedAt: '2026-10-25T09:40:00Z', contentHash: `h${version}`, sizeBytes: 1200, plainText: 'Hi', html: '<p>Hi</p>' })
        },
      },
    })
    unmock = page.unmock
    const chip = () => document.getElementById('cstatus')!
    await waitFor(() => expect(chip()).toHaveTextContent('Approved · saved 09:38 · v14'))

    const banner = (await screen.findByText(/^Approved by Marcus Chen/)).closest<HTMLElement>('[role="status"]')!
    fireEvent.click(within(banner).getByRole('button', { name: 'Edit' }))
    fireEvent.click(within(await screen.findByRole('dialog', { name: 'Edit an approved campaign?' })).getByRole('button', { name: 'Edit anyway' }))
    // An edit, saved with ⌘S: the save says the approval may go.
    fireEvent.click(within(screen.getByRole('region', { name: 'Inbox' })).getByRole('button', { name: 'Edit' }))
    fireEvent.change(await screen.findByRole('textbox', { name: /^Subject/ }), { target: { value: 'Three launches and a deprecation' } })
    act(() => void runShortcutHandler('save'))
    await waitFor(() => expect(saves).toHaveLength(1))
    expect(saves[0]).toMatchObject({ cancelApproval: true, subject: 'Three launches and a deprecation' })
    await waitFor(() => expect(chip()).toHaveTextContent('Saved 09:40 · v15'))
    expect(chip()).not.toHaveTextContent('Approved')

    // The approval is gone: the next save doesn't cancel whatever comes next.
    fireEvent.change(screen.getByRole('textbox', { name: /^Subject/ }), { target: { value: 'Three launches, one deprecation' } })
    act(() => void runShortcutHandler('save'))
    await waitFor(() => expect(saves).toHaveLength(2))
    expect(saves[1]).not.toHaveProperty('cancelApproval')
  })
})

describe('the read-only canvas', () => {
  let editor: Editor
  beforeAll(async () => {
    ;({ editor } = await headlessMjmlEditor('<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>'))
  })
  afterAll(() => editor.destroy())

  it('gives every component back what it allowed', () => {
    const text = editor.getWrapper()!.findType('mj-text')[0]!
    const body = editor.getWrapper()!.findType('mj-body')[0]!
    const before = { text: [text.get('editable'), text.get('draggable')], body: body.get('draggable') }
    const locked = lockCanvas(editor)
    expect(text.get('editable')).toBe(false)
    expect(text.get('draggable')).toBe(false)
    unlockCanvas(locked)
    expect([text.get('editable'), text.get('draggable')]).toEqual(before.text)
    expect(body.get('draggable')).toEqual(before.body)
    expect(editor.UndoManager.hasUndo()).toBe(false)
  })
})

describe('Send test', () => {
  let unmock: (() => void) | undefined
  afterEach(() => unmock?.())
  const me = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] })
  const campaign = testCampaign({ access: { ...testCampaign().access, canSendTestToOthers: false } })

  it('waits for a successful save, and sends nothing when it fails', async () => {
    const sent = vi.fn(() => json({ sent: ['linh@acme.io'], provider: 'Amazon SES', languages: [] }))
    unmock = mockApi({ [`POST /api/v1/workspaces/acme-newsletter/campaigns/${campaign.id}/test`]: sent })
    const save = vi.fn(async () => false)
    await renderInWorkspace(
      <ComposerCtx.Provider value={testComposer({ campaign, save })}>
        <SendTestButton toMe />
      </ComposerCtx.Provider>,
      { me },
    )
    fireEvent.click(screen.getByRole('button', { name: /Send test/ }))
    await waitFor(() => expect(save).toHaveBeenCalled())
    expect(await screen.findByText('Send test needs the latest save')).toBeInTheDocument()
    expect(sent).not.toHaveBeenCalled()
  })

  it('says why it waits while the last save failed', () => {
    const c = testComposer({ status: { kind: 'error', version: 15, message: 'offline' } })
    expect(testBlockedReason(c)).toBe('Send test needs the latest save')
    expect(testBlockedReason(testComposer())).toBeUndefined()
    // Someone who can't edit has nothing to save: the saved email is the one they see.
    expect(testBlockedReason(testComposer({ canEdit: false, status: { kind: 'error', version: 15, message: 'x' } }))).toBeUndefined()
  })
})

describe('checks that could not run', () => {
  const checks = { checkedAt: '2026-10-25T09:31:00Z', blocks: 2, warnings: 3, issues: [] } as unknown as Checks

  it('keep the last results, say so, and retry', () => {
    const retryChecks = vi.fn()
    render(
      <ComposerCtx.Provider value={testComposer({ checks, checksFailure: { reason: 'network' }, retryChecks })}>
        <ChecksRunBar />
      </ComposerCtx.Provider>,
    )
    expect(screen.getByRole('status')).toHaveTextContent(/Checks couldn't run \(network\) · showing results from \d\d:\d\d/)
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(retryChecks).toHaveBeenCalled()
  })

  it('say when the retry is running, and when it worked', () => {
    const { rerender } = render(
      <ComposerCtx.Provider value={testComposer({ checks, checksFailure: { reason: 'server' }, checksPending: true })}>
        <ChecksRunBar />
      </ComposerCtx.Provider>,
    )
    expect(screen.getByRole('status')).toHaveTextContent(/Running checks… · results from \d\d:\d\d stay until this finishes/)
    rerender(
      <ComposerCtx.Provider value={testComposer({ checks, checksRecovered: true })}>
        <ChecksRunBar />
      </ComposerCtx.Provider>,
    )
    expect(screen.getByRole('status')).toHaveTextContent('Checked just now · 2 blocks and 3 warnings')
  })
})
