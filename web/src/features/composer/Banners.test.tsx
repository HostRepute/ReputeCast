import { fireEvent, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { ComposerBanners, type Recovered } from './Banners'
import { ComposerCtx, type ComposerState } from './context'
import type { JournalEntry } from './journal'
import { testCampaign, testComposer } from './testing'

const me = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] })

const entry: JournalEntry = {
  v: 1,
  campaignId: 'c1',
  language: '',
  baseVersion: 13,
  startedAt: '2026-10-25T09:31:00Z',
  updatedAt: '2026-10-25T09:35:00Z',
  subject: 'S',
  preheader: 'P',
  mjml: '<mjml/>',
  abTest: null,
}

function show(state: ComposerState, recovered?: Recovered) {
  return renderInWorkspace(
    <ComposerCtx.Provider value={state}>
      <div className="cmp">
        <ComposerBanners recovered={recovered} />
      </div>
    </ComposerCtx.Provider>,
    { me },
  )
}

describe('composer banners', () => {
  it('offers edits this browser kept, as ComposerStates draws them', async () => {
    const restore = vi.fn(async () => true)
    const discard = vi.fn()
    await show(testComposer({ campaign: testCampaign({ version: 13 }), currentVersion: () => 13 }), { entry, restore, discard })
    expect(screen.getByText("4 minutes of edits from this browser weren't saved (09:31–09:35).")).toBeInTheDocument()
    expect(screen.getByText("They're only on this device. Restoring saves them as v14 on top of v13; discarding can't be undone.")).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Restore them' }))
    await waitFor(() => expect(restore).toHaveBeenCalled())
    fireEvent.click(screen.getByRole('button', { name: 'Discard' }))
    expect(discard).toHaveBeenCalled()
  })

  it('says when someone saved since the edits were made', async () => {
    await show(testComposer({ campaign: testCampaign({ version: 15, updatedBy: 'Marcus Chen' }), currentVersion: () => 15 }), { entry, restore: vi.fn(), discard: vi.fn() })
    expect(screen.getByText(/made on v13; Marcus Chen saved v15 since\. Restoring saves them as v16 on top/)).toBeInTheDocument()
  })

  it('resolves a conflict three ways: keep mine, compare, or load theirs after keeping mine', async () => {
    const save = vi.fn(async () => true)
    const reload = vi.fn(async () => {})
    const openDialog = vi.fn()
    const state = testComposer({
      save,
      reloadFromServer: reload,
      openDialog,
      status: { kind: 'conflict', version: 14, theirs: { version: 15, by: 'Marcus Chen', at: '2026-10-25T09:40:00Z', theirChanges: ['hero'], yourChanges: ['hero', 'subject'] } },
    })
    await show(state)
    expect(screen.getByRole('alert')).toHaveTextContent('Marcus Chen saved v15 while you were editing.')
    expect(screen.getByText(/You both changed the hero\./)).toBeInTheDocument()
    expect(screen.getByText("Keeps yours in version history as v16, then opens Marcus's v15.")).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Compare' }))
    expect(openDialog).toHaveBeenCalledWith('compare')

    fireEvent.click(screen.getByRole('button', { name: 'Load theirs' }))
    await waitFor(() => expect(reload).toHaveBeenCalled())
    expect(save).toHaveBeenCalledWith({ stash: true })

    fireEvent.click(screen.getByRole('button', { name: 'Keep mine' }))
    await waitFor(() => expect(save).toHaveBeenCalledWith({ force: true, manual: true, quiet: true }))
  })
})
