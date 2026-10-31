import { fireEvent, screen, waitFor } from '@testing-library/react'
import { vi } from 'vitest'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ComposerCtx } from '@/features/composer/context'
import { testComposer } from '@/features/composer/testing'
import { renderInWorkspace, workspaceMe } from '@/test/app'
import { useLanguageNav } from './composerParts'

function Switcher() {
  const go = useLanguageNav()
  return (
    <button type="button" onClick={() => void go('es')}>
      Spanish
    </button>
  )
}

const me = workspaceMe({ permissions: ['campaigns:view', 'campaigns:edit'] })

async function open(save: () => Promise<boolean>) {
  const state = testComposer({ save: vi.fn(save), hasUnsaved: () => true })
  const utils = await renderInWorkspace(
    <ComposerCtx.Provider value={state}>
      <Switcher />
      <UnsavedChangesGuard dirty description="You changed the subject." />
    </ComposerCtx.Provider>,
    { me, path: `/w/acme-newsletter/campaigns/${state.campaign.id}/edit` },
  )
  return { ...utils, state }
}

// CO-2: switching the language version saves first, and a failed save
// leaves the decision to the leave guard instead of dropping the edits.
describe('the language switch', () => {
  it('saves the edits, then opens the other version', async () => {
    const { router, state } = await open(async () => true)
    fireEvent.click(screen.getByRole('button', { name: 'Spanish' }))
    await waitFor(() => expect(router.state.location.search).toMatchObject({ language: 'es' }))
    expect(state.save).toHaveBeenCalledWith({ manual: true, quiet: true })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('asks before leaving when the edits could not be saved', async () => {
    const { router } = await open(async () => false)
    fireEvent.click(screen.getByRole('button', { name: 'Spanish' }))
    expect(await screen.findByRole('dialog')).toHaveTextContent('You changed the subject.')
    expect(router.state.location.search).not.toMatchObject({ language: 'es' })
  })
})
